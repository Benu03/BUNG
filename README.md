# BUNG

Modular web platform, SAP-style: each business module is a self-contained
pair of a React frontend and a Go backend, fronted by a single nginx
gateway. Adding a new module is meant to be "add a folder", not "rewire the
whole stack".

## Stack

- **Frontend:** React + Vite (one app per module)
- **Backend:** Go, standard library `net/http` (one service per module)
- **Database:** Postgres, one shared instance/database, **one schema per
  module** (see [Database](#database) below)
- **Gateway:** nginx, path-based routing (`/<module>/`, `/<module>/api/`)
- **Orchestration:** Docker Compose (root file `include`s each module's own
  compose file)

## Structure

```
BUNG/
├── docker-compose.yml        # root: gateway, postgres + includes every module
├── .env / .env.example       # shared env vars for all modules
├── nginx/
│   └── nginx.conf            # gateway: includes modules/*/nginx.conf
├── portal/
│   └── frontend/              # landing page at "/", lists modules
└── modules/
    ├── app-maintenance/
    │   ├── docker-compose.yml   # this module's frontend+backend services
    │   ├── nginx.conf            # this module's gateway location blocks
    │   ├── frontend/              # React (Vite) app
    │   └── backend/               # Go app
    └── kanban/
        └── ... (same shape)
```

## Current modules

- **app-maintenance** (`/app-maintenance/`): user, role, and module
  administration. Assign roles to users and control which modules a role
  can access. Also the source of truth for the module list shown on the
  portal landing page (`/`).
- **kanban** (`/kanban/`): boards, columns and cards.

## Running

```bash
cp .env.example .env   # first time only - then set your own POSTGRES_PASSWORD
docker compose up -d --build
```

Open http://localhost:8080/ (port from `NGINX_PORT` in `.env`, default
`8080`) - the portal landing page links out to each module.

## Database

All modules share **one Postgres instance and one database** (`POSTGRES_DB`
in `.env`), but each module connects using its own **schema**
(`<MODULE>_DB_SCHEMA`), set via the `search_path` connection parameter in
each backend's `db.go`. A module's backend creates its own schema and
tables on startup (`CREATE SCHEMA IF NOT EXISTS ...`) - there's no manual
provisioning step for a new module's database.

```
postgres (bung_db)
├── schema: app_maintenance   (modules, roles, users, role_modules, user_roles)
└── schema: kanban            (boards, columns, cards)
```

> If you change `POSTGRES_PASSWORD`/`POSTGRES_DB` in `.env` **after** the
> `postgres` volume already exists, Postgres will NOT pick up the new
> credentials (it only runs its init step once, on an empty data
> directory). Either reset the dev volume
> (`docker compose down -v && docker compose up -d`, re-seeds everything),
> or `ALTER USER`/`ALTER DATABASE` manually to match.

## Adding a new module

Say you want to add `finance`:

1. Copy the folder: `modules/app-maintenance` → `modules/finance`
2. `modules/finance/docker-compose.yml`: rename the two services
   (`app-maintenance-frontend`/`-backend` → `finance-frontend`/`-backend`),
   their `PORT`/`VITE_BASE` env references, and set `DB_SCHEMA` to
   `finance`.
3. `modules/finance/nginx.conf`: change the location prefixes and
   `proxy_pass` upstream names from `app-maintenance` to `finance`.
4. `modules/finance/frontend/vite.config.js`: update the default `base`
   to `/finance/`.
5. `modules/finance/backend/go.mod`: rename the module (e.g.
   `finance-backend`). Replace `store.go`/`migrate.go`/`models.go` with
   finance's own tables - the users/roles/modules schema in
   app-maintenance is specific to that module, not a shared base class.
6. Add `FINANCE_BACKEND_PORT` / `FINANCE_VITE_BASE` / `FINANCE_DB_SCHEMA`
   to `.env` (see `.env.example` for the pattern).
7. In the **root** `docker-compose.yml`, add one line under `include:` and
   add `finance-frontend`/`finance-backend` to nginx's `depends_on`:
   ```yaml
   include:
     - modules/app-maintenance/docker-compose.yml
     - modules/finance/docker-compose.yml
   ```
8. Register it in App Maintenance (Modules tab, code `finance`) so it
   shows up on the portal landing page - or insert it via the seed data
   in `modules/app-maintenance/backend/migrate.go`.
9. `docker compose up -d --build`

nginx picks up the new module's `nginx.conf` automatically (it's mounted
as a volume and globbed) - no edits to `nginx/nginx.conf` needed. The
required edits to shared files are: the one `include:` + `depends_on` line
in the root compose file (Compose doesn't support wildcard includes), and
registering the module's code in App Maintenance so the portal can list it.

## Public access (Cloudflare)

The gateway doesn't hardcode any domain/server_name, so it's ready to sit
behind a Cloudflare Tunnel or DNS proxy in front of the `nginx` service's
published port. When you set that up, consider restricting
`set_real_ip_from` in `nginx/nginx.conf` to Cloudflare's IP ranges so
`X-Forwarded-For` can be trusted.

## Login / SSO (not implemented yet)

The portal (`/`) currently has **no login enforced** - it's a plain
landing page listing active modules from App Maintenance, and every module
is reachable directly. This is intentional for now while the module
scaffolding settles.

When auth is added, the plan (informed by reviewing an internal reference
SSO implementation) is:

- Keep the **module + per-module-role** model App Maintenance already has
  (`modules`, `roles` scoped by module via `role_modules`, users linked via
  `user_roles`) as the authorization source of truth - it's already shaped
  right for this.
- Prefer a **JWT issued by App Maintenance** (short-TTL, e.g. RS256) over a
  design where every module calls back to App Maintenance synchronously on
  every request - avoids a hard runtime dependency between modules and
  keeps each module's Go middleware simple (verify a signature, no network
  call).
- Route auth through nginx so it's same-origin (already the case here -
  everything is under one host/port), avoiding CORS/cookie-domain issues.
- Avoid passing long-lived tokens as URL query params (visible in browser
  history/referrer/logs) - prefer an httpOnly cookie or a short-lived
  one-time exchange token if a redirect-based handoff is needed between
  the portal/App Maintenance and a module.

## Notes on the app-maintenance module

`modules/app-maintenance/backend/migrate.go` seeds an `admin` user, an
`Administrator` role (granted every known module), and one `modules` row
per module that ships with this repo, so the UI and the portal are usable
immediately after `docker compose up`. When you add a new module, add a
matching entry there (or just add it once via the Modules tab - either
way, it needs a `modules` row to show up on the portal).

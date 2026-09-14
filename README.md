# BUNG

Modular web platform, SAP-style: each business module is a self-contained
pair of a React frontend and a Go backend, fronted by a single nginx
gateway. Adding a new module is meant to be "add a folder", not "rewire the
whole stack".

## Stack

- **Frontend:** React + Vite + Tailwind CSS v4, shadcn/ui-style components
  (one app per module, see [Design system](#design-system))
- **Backend:** Go, standard library `net/http` (one service per module)
- **Database:** Postgres, one shared instance/database, **one schema per
  module** (see [Database](#database) below)
- **Auth:** session JWT issued/verified only by App Maintenance, enforced
  centrally at the nginx gateway (see [Login / auth](#login--auth))
- **Gateway:** nginx, path-based routing (`/<module>/`, `/<module>/api/`)
- **Orchestration:** Docker Compose (root file `include`s each module's own
  compose file)

## Structure

```
BUNG/
├── docker-compose.yml        # root: gateway, postgres + includes every module
├── .env / .env.example       # shared env vars for all modules
├── nginx/
│   ├── nginx.conf            # gateway: includes modules/*/nginx.conf
│   └── auth-common.conf      # shared auth_request snippet, included per protected location
├── portal/
│   └── frontend/              # landing page at "/": login form + module grid
└── modules/
    ├── app-maintenance/
    │   ├── docker-compose.yml   # this module's frontend+backend services
    │   ├── nginx.conf            # this module's gateway location blocks
    │   ├── frontend/              # React (Vite) app
    │   └── backend/               # Go app (also the platform's auth authority)
    └── kanban/
        └── ... (same shape)
```

## Current modules

- **app-maintenance** (`/app-maintenance/`): users, roles, modules, login,
  portal content settings, and the audit log. The source of truth for
  authentication and for the module list shown on the portal.
- **kanban** (`/kanban/`): boards, columns and cards.

## Running

```bash
cp .env.example .env   # first time only
# then edit .env: set your own POSTGRES_PASSWORD, JWT_SECRET
# (`openssl rand -hex 32`), and SEED_ADMIN_PASSWORD
docker compose up -d --build
```

Open http://localhost:8080/ (port from `NGINX_PORT` in `.env`, default
`8080`) and sign in with `admin` / whatever you set `SEED_ADMIN_PASSWORD`
to (only used the first time the `admin` user is seeded).

> **Port conflicts:** if a host tool (DBeaver, psql, ...) fails to connect
> to Postgres with "password authentication failed" even though the
> password in `.env` is right, you likely have another Postgres (native
> install, another project's container) already bound to that port -
> `localhost` silently routes to the wrong server instead of erroring.
> Check with `netstat -ano | grep LISTEN` (Windows) and change
> `POSTGRES_PORT` in `.env` to something clearly free (the default is a
> high, uncommon port for exactly this reason).

## Database

All modules share **one Postgres instance and one database** (`POSTGRES_DB`
in `.env`), but each module connects using its own **schema**
(`<MODULE>_DB_SCHEMA`), set via the `search_path` connection parameter in
each backend's `db.go`. A module's backend creates its own schema and
tables on startup - there's no manual provisioning step for a new module's
database.

```
postgres (bung_db)
├── schema: app_maintenance   (modules, roles, users, role_modules, user_roles, site_settings)
├── schema: kanban            (boards, columns, cards)
└── schema: audit             (activity_log - shared, cross-module, see Audit log below)
```

> If you change `POSTGRES_PASSWORD`/`POSTGRES_DB` in `.env` **after** the
> `postgres` volume already exists, Postgres will NOT pick up the new
> credentials (it only runs its init step once, on an empty data
> directory). Either reset the dev volume
> (`docker compose down -v && docker compose up -d`, re-seeds everything),
> or `ALTER USER`/`ALTER DATABASE` manually to match.

### Schema upgrades (adding a column later)

Each backend's `migrate.go` has two SQL blocks, run in order on every
startup:

- `schemaSQL` - `CREATE TABLE IF NOT EXISTS ...`. Only affects brand-new
  tables; a no-op once a table exists (adding a column here does **not**
  add it to an existing deployment).
- `alterSQL` - idempotent `ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...`
  statements. **This is the upgrade path**: when a table needs a new
  column, add the `CREATE TABLE` version (for fresh installs) *and* an
  `ALTER TABLE ADD COLUMN IF NOT EXISTS` line here (for existing ones),
  redeploy, done - no separate migration runner needed for changes this
  simple. If a change is bigger than "add a column" (renames, backfills,
  splitting tables), write it as its own explicit, idempotent statement in
  the same place rather than reaching for a migration framework prematurely.

## Login / auth

Every module (including App Maintenance's own UI/API) requires a session,
enforced centrally at the nginx gateway - no module backend other than
App Maintenance contains any auth code.

- **Login** (`POST /app-maintenance/api/auth/login`): checks username +
  bcrypt password hash, and on success signs a JWT (HS256, `JWT_SECRET`)
  containing the user id and the **module codes** their roles grant access
  to, and sets it as an httpOnly session cookie (`bung_session`,
  `SameSite=Lax`). Failed attempts are rate-limited per username+IP (5
  failures / 5 min → 15 min lockout, see `ratelimit.go`) and every attempt
  is written to the audit log.
- **Gateway enforcement**: every protected `location` block (in each
  module's `nginx.conf`) does `set $auth_module_code "<code>"; include
  /etc/nginx/auth-common.conf;`, which runs nginx's `auth_request` against
  an internal-only endpoint (`/_auth/verify`, proxied to App Maintenance's
  `GET /auth/verify`). That handler only checks the JWT's signature/expiry
  and whether its baked-in module list includes `X-Module-Code` - **no
  database call per request**, so it stays cheap regardless of traffic. On
  success it also returns `X-User-Id`/`X-Username`, which nginx forwards to
  the upstream module - so a module can know who's calling without
  implementing any JWT logic itself.
- Because the module list is baked into the token at login time, a
  role/module change takes effect the next time the affected user logs in
  (or when the token expires - `AUTH_TOKEN_TTL_HOURS`), not instantly.
- The **portal** (`/`) is the one public location: it shows a login form
  when `GET /app-maintenance/api/auth/me` returns 401, or the module grid
  (filtered to what the logged-in user can access) when it returns 200.

Adding a new module gets this for free - just follow the pattern already in
`modules/kanban/nginx.conf` (`set $auth_module_code "kanban"; include
.../auth-common.conf;` on both its frontend and API `location` blocks).

## Audit log

`audit.activity_log` is a **shared, cross-module** table in its own
Postgres schema (`audit`, not any single module's schema), fully-qualified
in every query so it's reachable regardless of a connection's
`search_path`. Only App Maintenance writes to it today (auth events, and
every user/role/module/settings mutation - see `writeAudit` in
`handlers.go` and `audit.go`), but the schema is module-agnostic
(`module_code` says where an entry came from) precisely so another module
can start writing entries the same way later, without a schema change.
Viewable (paginated, 50/page) from the **Audit Log** tab in App
Maintenance; the API also takes `?limit=&offset=`.

## Portal content settings

The **Settings** tab in App Maintenance controls what the public portal
(`/`) shows before anyone signs in: site name, tagline (under the site
name on the login card), and an optional announcement banner (shown on the
module dashboard once logged in, hidden when blank). Backed by a
single-row `site_settings` table; the portal reads it from the public
`GET /app-maintenance/api/settings/public` endpoint (see the exact-match
nginx location that bypasses the auth gate for just that one path).

## Design system

Tailwind CSS v4 + a small set of hand-written shadcn/ui-style primitives
(`Button`, `Input`, `Label`, `Card`, `Badge`, `Alert`) under each frontend's
`src/components/ui/`, with `class-variance-authority` for variants and a
`cn()` helper (`clsx` + `tailwind-merge`) - the same structure/API
`npx shadcn` would scaffold, just committed as plain source instead of
pulled from their registry. Since each module is an independent Vite app
(no shared node_modules/monorepo tooling), the `ui/` folder, `lib/utils.js`
and the color tokens in `index.css` are **duplicated** across
`portal/frontend`, `modules/app-maintenance/frontend` and
`modules/kanban/frontend` on purpose - copy them into a new module's
`frontend/` the same way you copy the rest of the module folder. Color
tokens (`index.css`) use an Odoo-inspired purple (`#714b67`) as `--primary`
and switch light/dark automatically via `prefers-color-scheme` (no theme
toggle). If this grows past 3-4 modules, consider extracting `ui/` into a
local npm workspace package instead of copy-pasting further.

## Adding a new module

Say you want to add `finance`:

1. Copy the folder: `modules/app-maintenance` → `modules/finance` (or
   `modules/kanban` if you don't need app-maintenance's auth-adjacent code)
2. `modules/finance/docker-compose.yml`: rename the two services
   (`app-maintenance-frontend`/`-backend` → `finance-frontend`/`-backend`),
   their `PORT`/`VITE_BASE` env references, and set `DB_SCHEMA` to
   `finance`. Drop the `JWT_SECRET`/`AUTH_TOKEN_TTL_HOURS`/
   `SEED_ADMIN_PASSWORD` env vars - only App Maintenance needs those.
3. `modules/finance/nginx.conf`: change the location prefixes and
   `proxy_pass` upstream names from `app-maintenance`/`kanban` to
   `finance`, keeping the `set $auth_module_code "finance"; include
   /etc/nginx/auth-common.conf;` lines so it's gated the same way as every
   other module.
4. `modules/finance/frontend/vite.config.js`: update the default `base`
   to `/finance/`.
5. `modules/finance/backend/go.mod`: rename the module (e.g.
   `finance-backend`). Replace `store.go`/`migrate.go`/`models.go` with
   finance's own tables - drop `auth.go`/`ratelimit.go`/`audit.go` if you
   copied from app-maintenance, finance doesn't need them (see
   [Login / auth](#login--auth)).
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
   shows up on the portal landing page, and grant it to whichever role(s)
   should see it - or insert both via the seed data in
   `modules/app-maintenance/backend/migrate.go`.
9. `docker compose up -d --build`

nginx picks up the new module's `nginx.conf` automatically (it's mounted
as a volume and globbed) - no edits to `nginx/nginx.conf` needed. The
required edits to shared files are: the one `include:` + `depends_on` line
in the root compose file (Compose doesn't support wildcard includes), and
registering the module's code in App Maintenance so the portal can list it.

## Public access (Cloudflare)

The gateway doesn't hardcode any domain/server_name, so it's ready to sit
behind a Cloudflare Tunnel or DNS proxy in front of the `nginx` service's
published port. When you set that up:

- Restrict `set_real_ip_from` in `nginx/nginx.conf` to Cloudflare's IP
  ranges so `X-Forwarded-For`/`X-Real-IP` (which the audit log and login
  rate limiter rely on) can be trusted instead of spoofed.
- Turn on the `Secure` flag on the session cookie (`setSessionCookie` in
  `modules/app-maintenance/backend/auth.go`) once traffic is HTTPS-only.

## Notes on the app-maintenance module

`modules/app-maintenance/backend/migrate.go` seeds an `admin` user (see
`SEED_ADMIN_PASSWORD`), an `Administrator` role (granted every known
module), one `modules` row per module that ships with this repo, and
default portal content, so the UI and the portal are usable immediately
after `docker compose up`. When you add a new module, add a matching
`modules` entry there (or just add it once via the Modules tab - either
way, it needs a `modules` row, and a role needs to be granted it, to show
up on the portal for anyone but a role with every module).

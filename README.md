# BUNG

Modular web platform, SAP-style: each business module is a self-contained
pair of a React frontend and a Go backend, fronted by a single nginx
gateway. Adding a new module is meant to be "add a folder", not "rewire the
whole stack".

## Stack

- **Frontend:** React + Vite (one app per module)
- **Backend:** Go, standard library `net/http` (one service per module)
- **Gateway:** nginx, path-based routing (`/<module>/`, `/<module>/api/`)
- **Orchestration:** Docker Compose (root file `include`s each module's own
  compose file)

## Structure

```
BUNG/
├── docker-compose.yml        # root: gateway + includes every module
├── .env / .env.example       # shared env vars for all modules
├── nginx/
│   └── nginx.conf            # gateway: includes modules/*/nginx.conf
└── modules/
    └── app-maintenance/
        ├── docker-compose.yml   # this module's frontend+backend services
        ├── nginx.conf            # this module's gateway location blocks
        ├── frontend/              # React (Vite) app
        └── backend/               # Go app
```

## Current modules

- **app-maintenance** (`/app-maintenance/`): user, role, and module
  administration. Assign roles to users and control which modules a role
  can access.

## Running

```bash
cp .env.example .env   # first time only
docker compose up -d --build
```

Open http://localhost:8080/app-maintenance/ (port from `NGINX_PORT` in
`.env`, default `8080`).

## Adding a new module

Say you want to add `finance`:

1. Copy the folder: `modules/app-maintenance` → `modules/finance`
2. `modules/finance/docker-compose.yml`: rename the two services
   (`app-maintenance-frontend`/`-backend` → `finance-frontend`/`-backend`)
   and their `PORT`/`VITE_BASE` env references.
3. `modules/finance/nginx.conf`: change the location prefixes and
   `proxy_pass` upstream names from `app-maintenance` to `finance`.
4. `modules/finance/frontend/vite.config.js`: update the default `base`
   to `/finance/`.
5. `modules/finance/backend/go.mod`: rename the module (e.g.
   `finance-backend`).
6. Add `FINANCE_BACKEND_PORT` / `FINANCE_VITE_BASE` to `.env` (see
   `.env.example` for the pattern).
7. In the **root** `docker-compose.yml`, add one line under `include:`:
   ```yaml
   include:
     - modules/app-maintenance/docker-compose.yml
     - modules/finance/docker-compose.yml
   ```
8. `docker compose up -d --build`

nginx picks up the new module's `nginx.conf` automatically (it's mounted
as a volume and globbed) - no edits to `nginx/nginx.conf` needed. The only
required edit to a shared file is the one `include:` line in the root
compose file, since Docker Compose doesn't support wildcard includes.

## Public access (Cloudflare)

The gateway doesn't hardcode any domain/server_name, so it's ready to sit
behind a Cloudflare Tunnel or DNS proxy in front of the `nginx` service's
published port. When you set that up, consider restricting
`set_real_ip_from` in `nginx/nginx.conf` to Cloudflare's IP ranges so
`X-Forwarded-For` can be trusted.

## Notes on the app-maintenance module

The backend currently uses an in-memory store (see
`modules/app-maintenance/backend/store.go`) seeded with an `admin` user, an
`Administrator` role, and the `app-maintenance` module itself, so the UI is
usable immediately after `docker compose up`. Data resets on container
restart - swap `store.go` for a real database once the schema stabilizes.

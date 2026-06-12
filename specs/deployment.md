# Deployment

Self-hosted production deployment of Nori via Docker Compose, designed for a
home lab or small-shop server. Pre-built images are published to GitHub
Container Registry on every push to `main`; the deployment host needs only
Docker, a compose file, and an env file — no source checkout.

## Architecture

One Caddy reverse proxy is the single entrypoint. The whole app lives on one
origin, so the browser uses same-origin relative API URLs (no CORS, no
hostnames baked into images).

```
                         ┌──────────────────────────────────────────┐
                         │ Docker host                              │
 browser ──:8080──▶ caddy├─ /api,/auth,/admin,/health ▶ nori-server │
                         │└─ everything else ─────────▶ nori-web    │
                         │   nori-server ──▶ database (postgres:16) │
                         │   nori-migrate (runs once, then exits)   │
                         └──────────────────────────────────────────┘
```

- **caddy** (`caddy:2-alpine`) — only service with a published host port
  (`NORI_HOST_PORT`, default 8080)
- **nori-server** (`ghcr.io/tylerjvollick/nori-server`) — Go API
- **nori-web** (`ghcr.io/tylerjvollick/nori-web`) — SvelteKit adapter-node SSR
- **nori-migrate** — same server image; runs `./nori migrate up` before the
  server starts (`service_completed_successfully`), so schema updates are
  automatic on every deploy
- **database** — PostgreSQL 16, named volume, not exposed on the host

Images are built for `linux/amd64` and `linux/arm64` by
`.github/workflows/build-images.yml`, tagged `latest` (tracks main) and
`sha-<commit>` (for pinning/rollback).

## Install (first time)

On the deployment host (Docker + Compose required):

```bash
mkdir -p ~/nori && cd ~/nori
# Copy these three files from the repo's docker/ directory:
#   docker-compose.yml  Caddyfile  .env.prod.example
cp .env.prod.example .env
# Edit .env: set POSTGRES_PASSWORD, NORI_JWT_SECRET, NORI_ADMIN_PASSWORD

docker compose pull
docker compose up -d
```

Then open `http://<host>:8080`, log in with `NORI_ADMIN_EMAIL` /
`NORI_ADMIN_PASSWORD`, and complete the forced password change.

If the GHCR packages are private, authenticate once first with a read-only
token: `docker login ghcr.io` (PAT with `read:packages`).

## Update

```bash
docker compose pull && docker compose up -d
```

New images are picked up, `nori-migrate` applies any new migrations, and the
server restarts. Data lives in named volumes (`postgres_data`,
`nori_uploads`) and survives updates.

## Rollback

Pin a previous image in `.env` and re-up:

```bash
# .env
NORI_IMAGE_TAG=sha-<commit>   # a known-good commit on main
docker compose pull && docker compose up -d
```

Note: migrations are not automatically rolled back. Rolling back past a
schema change requires `./nori migrate down` manually — prefer rolling
forward with a fix.

## Backups

Not yet automated (see architecture.md Future Components). Manual snapshot:

```bash
docker exec nori-db pg_dump -U postgres nori > nori-$(date +%F).sql
```

Uploads live in the `nori_uploads` volume.

## Building without the registry

From a source checkout, the same compose file builds images locally (the
services declare `build:` alongside `image:`):

```bash
docker compose -f docker/docker-compose.yml --env-file docker/.env up --build -d
```

## Not in scope (yet)

- HTTPS / external access — Caddy can terminate TLS later; today the stack
  is plain HTTP for LAN use
- Automated backups
- Auto-update (e.g. Watchtower) — updates are a manual two-command pull

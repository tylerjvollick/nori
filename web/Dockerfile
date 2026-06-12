# syntax=docker/dockerfile:1

# ── Dependencies ─────────────────────────────────────────────────────
FROM node:20-alpine AS deps

WORKDIR /app

COPY package*.json ./
RUN npm ci

# ── Dev server (used by docker-compose.dev.yml via target: dev) ──────
FROM deps AS dev

COPY . .

EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

# ── Production build ─────────────────────────────────────────────────
FROM deps AS build

COPY . .

# Empty VITE_API_URL → browser uses same-origin relative URLs; a reverse
# proxy in front routes /api, /auth, /admin, /health to the Go server.
ENV VITE_API_URL=""
RUN npm run build

# ── Production runtime (SvelteKit adapter-node) ──────────────────────
# Runtime env: PORT (default 3000), ORIGIN (public URL, required for
# correct origin checks), INTERNAL_API_URL (server-side fetches).
FROM node:20-alpine AS prod

WORKDIR /app
ENV NODE_ENV=production

COPY package*.json ./
RUN npm ci --omit=dev

COPY --from=build /app/build ./build

EXPOSE 3000
CMD ["node", "build"]

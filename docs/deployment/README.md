# Deployment — Neon + Render + Vercel

**Status:** Live on free tiers (no credit card).

```text
Frontend:       https://gitactionflow.vercel.app
Backend:        https://gitactionflow-backend.onrender.com
OAuth callback: https://gitactionflow.vercel.app/auth/github/callback  (proxied to Render)
GitHub webhook: https://gitactionflow-backend.onrender.com/webhooks/github
```

Browser calls go same-origin through Vercel (`/api`, `/auth`). GitHub webhooks hit Render directly.

---

## Architecture

```text
User
 │
 ▼
Vercel (SPA + /api /auth proxy)
 │
 ▼
Render — Go backend
 │
 ├── GitHub OAuth
 ├── GitHub Webhooks
 ├── GitHub API
 └── Slack Incoming Webhook
 │
 ▼
Neon — PostgreSQL (SSL)
```

Generic placeholders when recreating elsewhere:

```text
Frontend:       https://<vercel-domain>
Backend:        https://<render-domain>
OAuth callback: https://<vercel-domain>/auth/github/callback
GitHub webhook: https://<render-domain>/webhooks/github
```

---

## 1. Neon (PostgreSQL)

1. Create a free Neon project.
2. Copy the connection string (`DATABASE_URL`). Prefer the pooled URL Neon recommends for serverless/long-lived apps; include SSL (`sslmode=require` if not already present).
3. Do **not** commit this URL.

Migrations use the existing embedded golang-migrate files under `backend/migrations/`.

### Migration process

On Render, set:

```text
AUTO_MIGRATE=true
```

On backend startup the process runs `database.MigrateUp(DATABASE_URL)` (idempotent: already-applied migrations are skipped).

Alternatively, from a machine with network access to Neon:

```bash
cd backend
export DATABASE_URL='postgres://…neon…/neondb?sslmode=require'
# Temporary one-shot: run the server once with AUTO_MIGRATE=true, or use your usual migrate tooling against the same SQL files.
```

There is no separate migrate binary required — startup migrate is the supported production path for this assignment.

Never print `DATABASE_URL` in logs (config uses `Redacted()`).

---

## 2. Render (Go backend)

### Service settings

| Setting | Value |
| --- | --- |
| Runtime | Docker |
| Dockerfile path | `backend/Dockerfile` |
| Docker context | `backend` |
| Health check path | `/health` |
| Plan | Free |

Render injects `PORT`. The app prefers `PORT` over `APP_PORT`.

### Required environment variables

| Variable | Example / notes |
| --- | --- |
| `APP_ENV` | `production` |
| `DATABASE_URL` | Neon connection string (SSL) |
| `FRONTEND_URL` | `https://<your-vercel-domain>` (exact origin, no trailing slash) |
| `GITHUB_CLIENT_ID` | OAuth App client ID |
| `GITHUB_CLIENT_SECRET` | OAuth App secret |
| `GITHUB_OAUTH_REDIRECT_URL` | `https://<your-vercel-domain>/auth/github/callback` (proxied to Render) |
| `GITHUB_WEBHOOK_SECRET` | ≥ 16 chars; same as GitHub webhook secret |
| `SESSION_SECRET` | ≥ 32 chars (`openssl rand -hex 32`) |
| `SLACK_WEBHOOK_URL` | Incoming Webhook URL (server-only) |
| `AUTO_MIGRATE` | `true` (recommended for free-tier first deploy) |
| `COOKIE_SECURE` | `true` (default when `APP_ENV=production`) |
| `COOKIE_SAMESITE` | `None` (default when `APP_ENV=production`; required for Vercel↔Render cookies) |
| `EVENT_WORKER_ENABLED` | `true` |
| `LOG_LEVEL` | `info` |

Optional (defaults exist): `EVENT_MAX_RETRIES`, `EVENT_WORKER_POLL_INTERVAL`, `EVENT_PROCESSING_LEASE`, `ACTION_MAX_RETRIES`, `WEBHOOK_MAX_BODY_BYTES`, `SESSION_TTL`, `OAUTH_STATE_TTL`.

### Verify after deploy

```bash
curl -sS https://<your-render-domain>/health
# {"status":"ok"}

curl -sS https://<your-render-domain>/ready
# {"status":"ready"} when Neon is reachable
```

### Why SameSite=None (fallback)

If the SPA called Render **cross-origin**, browsers need `SameSite=None; Secure` for the session cookie. Prefer the **same-origin Vercel proxy** below so cookies are first-party (works in Incognito when third-party cookies are blocked).

---

## 3. Vercel (React frontend)

### Project settings

| Setting | Value |
| --- | --- |
| Root directory | `frontend` |
| Framework | Vite |
| Install | `npm install` |
| Build | `npm run build` |
| Output | `dist` |
| Routing | `frontend/vercel.json` proxies `/api/*` and `/auth/*` to Render, then SPA fallback to `index.html` |

### Same-origin API proxy (recommended)

`frontend/vercel.json` rewrites browser calls:

```text
https://<vercel>/api/*   → https://<render>/api/*
https://<vercel>/auth/*  → https://<render>/auth/*
```

OAuth callback and session cookie are then on the **Vercel** host. Webhooks stay on Render (GitHub → Render directly).

### Environment variables (Vercel)

| Variable | Value |
| --- | --- |
| `VITE_API_BASE_URL` | **Leave unset** in production (same-origin). Local dev defaults to `http://localhost:8080`. |

If `VITE_API_BASE_URL` still points at Render, remove it and **redeploy** so the build uses same-origin requests.

**Never** set backend secrets in Vercel.

---

## 4. GitHub OAuth App

Update (or create) the OAuth App:

| Field | Production value |
| --- | --- |
| Homepage URL | `https://<your-vercel-domain>` |
| Authorization callback URL | `https://<your-vercel-domain>/auth/github/callback` |

Must match Render `GITHUB_OAUTH_REDIRECT_URL` exactly (same Vercel callback URL).

You can keep a separate OAuth App for local development.

---

## 5. GitHub webhook (manual)

On the connected repository → Settings → Webhooks:

| Field | Value |
| --- | --- |
| Payload URL | `https://<your-render-domain>/webhooks/github` |
| Content type | `application/json` |
| Secret | same as `GITHUB_WEBHOOK_SECRET` |
| Events | **Issues** and **Pull requests** |

Do not put the secret in the frontend or README as a real value.

---

## 6. Slack

Create an Incoming Webhook in a free Slack workspace. Set `SLACK_WEBHOOK_URL` on **Render only**.

---

## 7. Deploy order (recommended)

1. Create Neon → copy `DATABASE_URL`
2. Deploy Render backend with env vars (`AUTO_MIGRATE=true`) → note public URL
3. Confirm `/health` and `/ready`
4. Deploy Vercel frontend (**unset** `VITE_API_BASE_URL` so same-origin proxy is used) → note public URL
5. Set Render `FRONTEND_URL` + `GITHUB_OAUTH_REDIRECT_URL` to the Vercel origin/callback; redeploy
6. Update GitHub OAuth homepage + **callback on Vercel** (`/auth/github/callback`)
7. Configure webhook on the demo repo
8. Set Slack URL on Render
9. Run [smoke-test.md](./smoke-test.md)

---

## Local development (unchanged)

Docker Compose Postgres + `go run` + Vite remain the local path. See [LOCAL.md](../setup/LOCAL.md).

Local cookies stay `COOKIE_SECURE=false`, `COOKIE_SAMESITE=Lax`.

---

## Related

- [smoke-test.md](./smoke-test.md)
- [production-checklist.md](./production-checklist.md)
- Root [README.md](../../README.md)

# Deployment Direction

**Status:** Planned — **nothing is deployed** yet. Phase 2 documents backend runtime expectations for when deployment happens.

GitHub OAuth callbacks and webhooks require a **public HTTPS URL**. Localhost alone is insufficient for the live assignment demo.

---

## Intended topology

```text
React frontend
      ↓
Public deployment
      ↓
Go backend
      ↓
PostgreSQL
```

| Piece | Role |
| --- | --- |
| React (Vite build) | Static assets + SPA (later) |
| Go backend | API, OAuth, webhooks, processing |
| PostgreSQL | Durable state |

---

## Expected free-tier services

Assignment constraint: **no credit card**. Prefer providers with genuine free tiers. Always re-check current free-tier terms before locking in a choice — limits and eligibility change.

| Concern | Candidate options (verify before use) |
| --- | --- |
| Frontend | Vercel, Netlify, or Render static hosting |
| Backend | Render (or similar free web service suitable for a long-running Go process) |
| PostgreSQL | Neon or Supabase free Postgres |
| Secrets | Host-provided environment variables |
| Slack | Free Slack workspace + Incoming Webhook |
| GitHub | Free OAuth App / webhooks / API |

If a service asks for a credit card for the tier you need, switch providers or tiers.

---

## Backend deployment considerations (Phase 2)

### Environment variables

At minimum the host must provide:

| Variable | Notes |
| --- | --- |
| `APP_ENV` | e.g. `production` |
| `APP_PORT` | Port the process listens on (platform may inject `PORT` — map as needed when deploying) |
| `DATABASE_URL` | Managed Postgres URL (TLS as required by provider) |
| `LOG_LEVEL` | e.g. `info` |
| `AUTO_MIGRATE` | Prefer explicit migrate control in production; default is `false` outside development/test |

Future secrets (`GITHUB_*`, `SLACK_WEBHOOK_URL`, `SESSION_SECRET`, AI keys) must come from the host secret store — never from the image or repo.

### PostgreSQL connection

- Use the provider connection string in `DATABASE_URL`.
- Prefer TLS (`sslmode=require` or provider default) in production.
- Do not expose Postgres on the public internet beyond the managed endpoint’s own controls.

### Health and readiness

| Endpoint | Use |
| --- | --- |
| `GET /health` | Liveness / process up |
| `GET /ready` | Readiness / Postgres reachable |

Configure the platform to:

- Restart on failed liveness if supported
- Route traffic only when readiness succeeds (when the platform supports it)

### Graceful shutdown

The server handles `SIGINT` / `SIGTERM`, stops accepting new requests, and closes the database pool. Prefer platforms that send SIGTERM on deploy/stop.

### Container image

`backend/Dockerfile` is a multi-stage build producing a minimal Alpine runtime image. Suitable for local Compose and future container hosts.

---

## Local vs production

| Environment | Notes |
| --- | --- |
| Local | `docker-compose up -d postgres` (+ optional `backend`); `go run ./cmd/server` |
| Production | Public HTTPS frontend + backend; managed Postgres; secrets only in host env |

---

## What is not done yet

- Create cloud accounts
- Provision remote databases
- Deploy frontend or backend
- Claim a live URL

Deployment steps and the live URL will be added to the root `README.md` when the deploy phase completes.

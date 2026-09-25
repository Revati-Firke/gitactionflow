# GitActionFlow

Event-driven automation for GitHub repositories.

Take-home for Abstrabit Software Engineer I: sign in with GitHub, connect one repo, react to signed webhooks, run simple rules, write back to GitHub and Slack, and show history on a dashboard—secure and durable enough for a real demo, without building a platform.

---

## Live demo

```text
Frontend:  https://gitactionflow.vercel.app
Backend:   https://gitactionflow-backend.onrender.com
Health:    https://gitactionflow-backend.onrender.com/health
```

Typical walkthrough: open the frontend → Login with GitHub → connect a repo you admin → create a rule → open an Issue → refresh the dashboard. Short script: [docs/assignment/demo-script.md](docs/assignment/demo-script.md).

Requirements ↔ evidence: [docs/assignment/requirements-matrix.md](docs/assignment/requirements-matrix.md) · [verification.md](docs/assignment/verification.md) · [final-acceptance.md](docs/assignment/final-acceptance.md)

---

## What ships

- GitHub OAuth, HttpOnly sessions, encrypted access tokens
- Connect / disconnect **one** repository (admin required)
- Signed webhooks (issues + pull requests), delivery-ID idempotency, background worker with retries
- Rules (AND conditions) → GitHub label / comment + Slack Incoming Webhook
- Dashboard: repo, rules, events, actions (including failures)
- Optional AI summaries / suggested labels (`AI_ENABLED`, off by default)

**Left out on purpose**

- Multi-repo product mode
- GitHub App installation auth (OAuth App is enough here)
- Automatic webhook create/delete on connect (configure the webhook once on the repo)

Why those cuts: see [AI_NOTES.md](AI_NOTES.md) and [docs/decisions/](docs/decisions/).

---

## Design choices (short)

| Choice | Rationale |
| --- | --- |
| Modular monolith (Go + React + Postgres) | One process to reason about; Postgres as queue and source of truth |
| Free stack: Neon + Render + Vercel | Assignment constraint; no paid APIs |
| Vercel proxy for `/api` and `/auth` | Session cookies stay first-party; Incognito-friendly |
| Persist before side effects | Ack only after durable event row; actions row before GitHub/Slack HTTP |
| Rule intents ≠ executor | Rules decide *what*; `internal/actions` does *how* (testable, idempotent) |

Architecture diagram and flow: [docs/architecture/HLA.md](docs/architecture/HLA.md).

```text
GitHub webhook → verify → dedupe → persist event
  → worker → rules → actions (GitHub / Slack) → persist results
  → dashboard reads Postgres
```

---

## Stack

| Layer | Choice |
| --- | --- |
| Backend | Go, Gin, pgx, golang-migrate |
| Frontend | React, TypeScript, Vite |
| Auth | GitHub OAuth App + server sessions |
| Data | PostgreSQL |
| Notify | Slack Incoming Webhook |
| Optional AI | Gemini or Groq behind a small interface |

---

## Production deploy (summary)

Full guide: [docs/deployment/README.md](docs/deployment/README.md) · checklist: [production-checklist.md](docs/deployment/production-checklist.md)

1. Neon Postgres → `DATABASE_URL`
2. Render Docker from `backend/` (`APP_ENV=production`, `AUTO_MIGRATE=true`)
3. Confirm `/health` and `/ready`
4. Vercel from `frontend/` — leave **`VITE_API_BASE_URL` unset** (same-origin proxy)
5. Render: `FRONTEND_URL` and `GITHUB_OAUTH_REDIRECT_URL` point at the Vercel origin / callback
6. GitHub OAuth App homepage + callback on the Vercel host
7. Repo webhook → `https://gitactionflow-backend.onrender.com/webhooks/github` (Issues + PRs)
8. Smoke: [docs/deployment/smoke-test.md](docs/deployment/smoke-test.md)

Cookie note: prefer first-party cookies via the Vercel proxy. Do not point the browser at Render’s API origin in production builds.

---

## Local development

Walkthrough: [docs/setup/LOCAL.md](docs/setup/LOCAL.md)

**Need:** Go 1.25+, Docker Compose, GitHub OAuth App.

```bash
# Postgres
docker-compose up -d postgres

# Backend (load .env from repo root — see .env.example)
cd backend && go run ./cmd/server

# Frontend
cd frontend && npm install && npm run dev
```

Open `http://localhost:5173`. Use **`localhost`**, not `127.0.0.1`, so cookies match OAuth.

| Variable | Role |
| --- | --- |
| `DATABASE_URL` | Postgres |
| `FRONTEND_URL` | CORS + post-login redirect |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | OAuth App |
| `GITHUB_OAUTH_REDIRECT_URL` | Must match the OAuth App callback |
| `SESSION_SECRET` | ≥32 chars (`openssl rand -hex 32`) |
| `GITHUB_WEBHOOK_SECRET` | Webhook HMAC |
| `SLACK_WEBHOOK_URL` | Optional Slack |

Health: `curl http://127.0.0.1:8080/health` and `/ready`.

Webhook locally: set the secret, tunnel `:8080` or use the signed curl in LOCAL.md, point the repo webhook at `/webhooks/github`.

Connect requires GitHub **admin** on the repo. Disconnect before switching repos. Webhook registration is still **manual**.

---

## Tests

```bash
cd backend && go test ./... && go vet ./...
cd frontend && npm run build
```

Unit coverage focuses on signature verification, idempotency, rule matching, and auth-sensitive paths. Manual E2E is on the live URLs above.

---

## Security

- HMAC-SHA256 webhook signatures (`hmac.Equal`), bounded body size
- Delivery ID uniqueness; action idempotency key before external calls
- OAuth `state` + Origin checks on mutating cookie APIs
- No secrets in the frontend bundle, logs, or git (see `.env.example`)

Details: [SECURITY.md](SECURITY.md). How AI was used: [AI_NOTES.md](AI_NOTES.md). Working rules for contributors/agents: [AGENTS.md](AGENTS.md).

---

## Docs map

| Doc | Purpose |
| --- | --- |
| [docs/setup/LOCAL.md](docs/setup/LOCAL.md) | Local OAuth + webhook setup |
| [docs/architecture/HLA.md](docs/architecture/HLA.md) | Architecture |
| [docs/api/README.md](docs/api/README.md) | HTTP API |
| [docs/database/README.md](docs/database/README.md) | Schema direction |
| [docs/deployment/README.md](docs/deployment/README.md) | Deploy |
| [docs/decisions/](docs/decisions/) | ADRs |
| [docs/assignment/](docs/assignment/) | Matrix, verification, demo, acceptance |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

---

## License

See [LICENSE](LICENSE).

# GitActionFlow

**Event-driven automation for Git repositories.**

GitActionFlow is a take-home engineering assessment for an Abstrabit Software Engineer I (SDE1) role. It demonstrates building a small but real product: a web app and bot that react to GitHub repository activity, apply configurable rules, write back to GitHub, notify Slack, and show results on an authenticated dashboard.

---

## Current status (Phase 2)

This repository is in **Phase 2: Backend Foundation**.

**What exists now**

- Project structure and Phase 1 documentation
- Runnable Go backend (`backend/cmd/server`) with Gin
- Environment-based configuration (`APP_ENV`, `APP_PORT`, `DATABASE_URL`, `LOG_LEVEL`, …)
- PostgreSQL connectivity via pgx pool
- SQL migrations (`golang-migrate`, embedded from `backend/migrations/`)
- `GET /health` (liveness) and `GET /ready` (Postgres check)
- Structured JSON logging (`log/slog`) and consistent HTTP error envelope
- Graceful shutdown (SIGINT/SIGTERM)
- Docker: backend `Dockerfile` + Compose services for Postgres and optional backend
- Foundation unit tests

**What is not implemented yet**

- GitHub OAuth, sessions, or login UI
- Webhook handling, event processing, or rule engine
- GitHub API write-back or Slack notifications
- Full application schema (users, repos, rules, events, actions)
- React dashboard pages
- Public deployment
- Optional AI triage

Do not assume features listed under “Core functionality (planned)” work until later phases land them.

---

## Assignment purpose

Deliver a polished, reliable, production-style implementation of an **Event-Driven GitHub Automation Bot**:

1. User signs in with GitHub and connects one repository they own.
2. The app receives signed webhooks (issues and pull requests at minimum).
3. The bot evaluates rules, acts on GitHub (label and/or comment), and notifies Slack.
4. A dashboard (behind login) shows event and action history and lets the user configure simple rules.
5. The system must be secure (signature verification, OAuth state, no secret leakage) and reliable (idempotent deliveries, durable persistence, visible failures).

Everything must use **free tiers only** (no credit card).

---

## Core functionality (planned)

| Capability | Phase |
| --- | --- |
| Backend foundation (config, DB, health/ready) | **Phase 2 (done)** |
| Publicly reachable web app | Later (deploy) |
| GitHub OAuth sign-in | Phase 3+ |
| Connect one owned repository | Later |
| Webhook endpoint (issues + pull requests) | Later |
| GitHub write-back (label / comment) | Later |
| Slack notifications | Later |
| Authenticated dashboard (repo, rules, events, actions) | Later |
| Configurable rules | Later |
| Webhook signature + delivery idempotency | Later |
| Durable processing without silent event loss | Later |
| Optional AI summary / label / priority (free provider) | Optional stretch |

---

## High-level architecture

Modular monolith. One Go backend serves auth, repository management, webhooks, processing, and dashboard APIs. React is the UI. PostgreSQL is the durable source of truth.

```text
                         USER
                           │
                           ▼
                  React Web Dashboard
                           │
                           ▼
                      Go Backend
                           │
                ┌──────────┼───────────┐
                ▼          ▼           ▼
              Auth     Repository    Dashboard
                │       Management     APIs
                │
                ▼
           GitHub OAuth

GitHub Repository
       │
       │ signed webhook
       ▼
 Webhook Handler
       │
       ├── verify signature
       ├── validate event
       ├── check delivery ID
       └── persist event
               │
               ▼
        Event Processor
               │
               ▼
          Rule Engine
               │
          ┌────┴─────┐
          ▼          ▼
       Optional      Actions
          AI          │
                      ├── GitHub API
                      └── Slack
               │
               ▼
        Action Results → PostgreSQL → React Dashboard
```

See [docs/architecture/HLA.md](docs/architecture/HLA.md) for the full description.

---

## Technology stack

| Layer | Choice |
| --- | --- |
| Backend | Go 1.25+ + Gin |
| Database | PostgreSQL via pgx; migrations via golang-migrate |
| Frontend | React + TypeScript + Vite (not scaffolded yet) |
| Auth | GitHub OAuth (not implemented yet) |
| Integrations | GitHub Webhooks, GitHub REST API, Slack Incoming Webhook (not implemented yet) |
| Containers | Docker / Docker Compose (Postgres + optional backend) |
| Optional AI | Gemini or Groq (stretch only; never required for core path) |

---

## Local development (backend)

### Requirements

- Go **1.25+**
- Docker + `docker-compose` (or Compose v2 plugin)
- PostgreSQL 16 (via Compose)

### PostgreSQL

```bash
# from repository root
docker-compose up -d postgres
```

Default local credentials (examples only — see `.env.example`):

```text
postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable
```

### Environment variables

Copy `.env.example` → `.env` and set at least:

| Variable | Purpose | Example (local) |
| --- | --- | --- |
| `APP_ENV` | Environment name | `development` |
| `APP_PORT` | HTTP listen port | `8080` |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `DATABASE_URL` | Postgres connection string | see above |
| `AUTO_MIGRATE` | Apply SQL migrations on startup | `true` (default in development) |

Future placeholders (`GITHUB_*`, `SLACK_WEBHOOK_URL`, `SESSION_SECRET`, AI keys) are listed in `.env.example` but unused in Phase 2.

### Start the backend

```bash
cd backend
export DATABASE_URL='postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable'
export APP_ENV=development APP_PORT=8080 LOG_LEVEL=info AUTO_MIGRATE=true
go run ./cmd/server
```

Or run everything in Compose:

```bash
docker-compose up -d --build
```

### Health and readiness

```bash
curl http://127.0.0.1:8080/health
# {"status":"ok"}

curl http://127.0.0.1:8080/ready
# {"status":"ready"}   # 503 if Postgres is down
```

### Migrations

With `AUTO_MIGRATE=true` (default in development), migrations under `backend/migrations/` apply on startup.

Current migration: `000001_foundation` creates a minimal `app_meta` table only. Full product schema comes later.

### Frontend

Not started yet. React + Vite will land in a later phase.

Webhook testing against a local machine will need a public tunnel once webhooks are implemented.

---

## Planned deployment

Free-tier oriented (verify current limits before choosing a provider):

| Piece | Likely host |
| --- | --- |
| React frontend | Vercel, Netlify, or Render static |
| Go backend | Render (or similar free web service) |
| PostgreSQL | Neon or Supabase |

OAuth callbacks and webhooks **must** use the public HTTPS URL — not localhost.

Backend hosts should configure `DATABASE_URL`, use `/health` for liveness and `/ready` for readiness, and allow graceful shutdown on SIGTERM. Details: [docs/deployment/README.md](docs/deployment/README.md).

**Status:** Not deployed yet.

---

## Testing

Backend foundation tests:

```bash
cd backend
go test ./...
gofmt -l .
go vet ./...
```

Later phases will add:

- Unit tests for signature verification, idempotency, and rule matching
- Handler tests with forged / replayed webhook fixtures
- Integration tests against local PostgreSQL
- Manual end-to-end checklist on the live URL

---

## Security considerations

Documented now; implemented in later phases:

- Verify `X-Hub-Signature-256` with HMAC-SHA256 and constant-time compare
- Treat GitHub delivery IDs as idempotency keys
- Validate OAuth `state` (CSRF protection)
- Keep all secrets server-side; never ship them to the frontend or commit them
- Do not log tokens, webhook secrets, Slack URLs, or database credentials

See [SECURITY.md](SECURITY.md).

---

## Documentation map

| Doc | Purpose |
| --- | --- |
| [AGENTS.md](AGENTS.md) | AI / developer working rules |
| [AI_NOTES.md](AI_NOTES.md) | Honest AI collaboration notes (fill during development) |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute; commit conventions |
| [SECURITY.md](SECURITY.md) | Security principles |
| [CHANGELOG.md](CHANGELOG.md) | Release history |
| [docs/architecture/HLA.md](docs/architecture/HLA.md) | High-level architecture |
| [docs/api/README.md](docs/api/README.md) | Planned API surface |
| [docs/database/README.md](docs/database/README.md) | Database direction |
| [docs/deployment/README.md](docs/deployment/README.md) | Deployment direction |
| [docs/decisions/README.md](docs/decisions/README.md) | ADR index |

---

## Future implementation phases (outline)

| Phase | Focus |
| --- | --- |
| **1** | Foundation, docs, conventions |
| **2 (current)** | Backend scaffold, Postgres access, health, config |
| **3** | Auth (GitHub OAuth + sessions) |
| **4** | Repository connect + webhook registration |
| **5** | Webhook ingest, persistence, idempotency |
| **6** | Rule engine + GitHub / Slack actions |
| **7** | React dashboard |
| **8** | Deploy + E2E hardening |
| **Optional** | Free-tier AI assist behind a clear abstraction |

---

## License

See [LICENSE](LICENSE).

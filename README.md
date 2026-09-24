# GitActionFlow

**Event-driven automation for Git repositories.**

GitActionFlow is a take-home engineering assessment for an Abstrabit Software Engineer I (SDE1) role. It demonstrates building a small but real product: a web app and bot that react to GitHub repository activity, apply configurable rules, write back to GitHub, notify Slack, and show results on an authenticated dashboard.

---

## Current status (Phase 4)

This repository is in **Phase 4: GitHub Repository Management**.

**What exists now**

- OAuth login + sessions (Phase 3)
- List GitHub repositories for the signed-in user
- Connect / disconnect **exactly one** repository (admin access required)
- Minimal React UI to sign in and manage that connection
- Backend foundation (health, Postgres, migrations)

**What is not implemented yet**

- Webhook registration or webhook endpoint
- Event processing, rules, Slack, AI
- Full event/action dashboard
- Public deployment

Webhook processing starts in Phase 5 — connecting a repository does **not** enable automation yet.

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
| GitHub OAuth sign-in + sessions | **Phase 3 (done)** |
| Connect one owned repository | **Phase 4 (done)** |
| Publicly reachable web app | Later (deploy) |
| Webhook endpoint (issues + pull requests) | Phase 5+ |
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

**Full walkthrough (OAuth App + test checklist):** [docs/setup/LOCAL.md](docs/setup/LOCAL.md)

### Requirements

- Go **1.25+**
- Docker + `docker-compose`
- PostgreSQL 16 (via Compose)
- GitHub OAuth App (see setup doc)

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
| `FRONTEND_URL` | Post-OAuth redirect + CORS origin | `http://localhost:5173` |
| `GITHUB_CLIENT_ID` | OAuth App client ID | from GitHub |
| `GITHUB_CLIENT_SECRET` | OAuth App client secret | from GitHub |
| `GITHUB_OAUTH_REDIRECT_URL` | Must match OAuth App callback | `http://localhost:8080/auth/github/callback` |
| `SESSION_SECRET` | ≥32 chars; cookies + token encryption | `openssl rand -hex 32` |
| `COOKIE_SECURE` | Set true behind HTTPS | `false` locally |
| `COOKIE_SAMESITE` | `Lax` (default), `Strict`, or `None` | `Lax` |

Future placeholders (`GITHUB_WEBHOOK_SECRET`, `SLACK_WEBHOOK_URL`, AI keys) are listed in `.env.example` but unused in Phase 3.

### GitHub OAuth App (local)

See **[docs/setup/LOCAL.md](docs/setup/LOCAL.md)** for exact form fields and troubleshooting.

Summary: create an **OAuth App** with callback `http://localhost:8080/auth/github/callback`, put Client ID/secret in `.env`, use **`localhost` only** (not `127.0.0.1`), restart the backend, open `http://localhost:5173`.

Scopes: `read:user repo`. Details: [ADR-005](docs/decisions/ADR-005-sessions-and-token-encryption.md).

### Start the backend

```bash
cd backend
# load .env somehow, or export vars manually
export DATABASE_URL='postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable'
export APP_ENV=development APP_PORT=8080 LOG_LEVEL=info AUTO_MIGRATE=true
export GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=...
export GITHUB_OAUTH_REDIRECT_URL=http://localhost:8080/auth/github/callback
export SESSION_SECRET="$(openssl rand -hex 32)"
export FRONTEND_URL=http://localhost:5173
go run ./cmd/server
```

### Repository connection (manual check)

After signing in (cookie jar or the React UI):

```bash
# List GitHub repos (server uses encrypted user token)
curl -sS -b /tmp/gaf.jar http://127.0.0.1:8080/api/github/repositories

# Connect by GitHub repository id only (metadata comes from GitHub)
curl -sS -b /tmp/gaf.jar -H 'Content-Type: application/json' \
  -d '{"github_repository_id":123456}' \
  http://127.0.0.1:8080/api/repository

# Show / disconnect
curl -sS -b /tmp/gaf.jar http://127.0.0.1:8080/api/repository
curl -sS -b /tmp/gaf.jar -X DELETE http://127.0.0.1:8080/api/repository
```

**Access rule:** connect requires GitHub `admin` permission on the repository (owners have admin).  
**One-repo rule:** disconnect the current repository before connecting another (`409` otherwise).  
**Scopes:** OAuth uses `read:user repo`.

Webhook registration / processing is **not** implemented yet.

### Health and readiness

```bash
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
```

### Migrations

With `AUTO_MIGRATE=true`, migrations apply on startup (`000001` foundation, `000002` auth, `000003` repositories).

### Frontend (minimal)

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:5173 — sign in, pick one repository, connect/disconnect. No rules or event logs yet.

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
| [docs/setup/LOCAL.md](docs/setup/LOCAL.md) | Local setup, OAuth App, test steps |
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
| **2** | Backend scaffold, Postgres access, health, config |
| **3** | Auth (GitHub OAuth + sessions) |
| **4 (current)** | Repository connect (one repo) |
| **5** | Webhook ingest, persistence, idempotency |
| **6** | Rule engine + GitHub / Slack actions |
| **7** | React dashboard |
| **8** | Deploy + E2E hardening |
| **Optional** | Free-tier AI assist behind a clear abstraction |

---

## License

See [LICENSE](LICENSE).

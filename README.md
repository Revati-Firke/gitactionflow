# GitActionFlow

**Event-driven automation for Git repositories.**

GitActionFlow is a take-home engineering assessment for an Abstrabit Software Engineer I (SDE1) role. It demonstrates building a small but real product: a web app and bot that react to GitHub repository activity, apply configurable rules, write back to GitHub, notify Slack, and show results on an authenticated dashboard.

---

## Current status (Phase 1)

This repository is in **Phase 1: Project Foundation & Documentation**.

**What exists now**

- Project structure (`backend/`, `frontend/`, `docs/`)
- Architecture and design documentation
- Security and reliability principles (documented, not yet implemented)
- Developer / AI context (`AGENTS.md`)
- Environment variable template (`.env.example`)
- Local PostgreSQL scaffolding via `docker-compose.yml`

**What is not implemented yet**

- GitHub OAuth, sessions, or login UI
- Webhook handling, event processing, or rule engine
- GitHub API write-back or Slack notifications
- Database schema / migrations
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
| Publicly reachable web app | Later (deploy) |
| GitHub OAuth sign-in | Phase 2+ |
| Connect one owned repository | Phase 2+ |
| Webhook endpoint (issues + pull requests) | Phase 2+ |
| GitHub write-back (label / comment) | Phase 2+ |
| Slack notifications | Phase 2+ |
| Authenticated dashboard (repo, rules, events, actions) | Phase 2+ |
| Configurable rules | Phase 2+ |
| Webhook signature + delivery idempotency | Phase 2+ |
| Durable processing without silent event loss | Phase 2+ |
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
| Backend | Go + Gin (or equivalent lightweight HTTP framework) |
| Database | PostgreSQL via pgx |
| Frontend | React + TypeScript + Vite |
| Auth | GitHub OAuth |
| Integrations | GitHub Webhooks, GitHub REST API, Slack Incoming Webhook |
| Containers | Docker / Docker Compose (local Postgres) |
| Optional AI | Gemini or Groq (stretch only; never required for core path) |

---

## Planned local development

> Exact commands will be finalized when backend and frontend scaffolds land. Outline below.

1. Clone the repository.
2. Copy `.env.example` → `.env` and fill values (never commit real secrets).
3. Start PostgreSQL: `docker compose up -d`.
4. Run the Go API from `backend/` (Phase 2+).
5. Run the Vite React app from `frontend/` (Phase 2+).
6. For webhook testing against a local machine, use a tunnel (e.g. Cloudflare Tunnel / similar free option) so GitHub can reach a public URL.

Required environment variables are listed in [`.env.example`](.env.example).

---

## Planned deployment

Free-tier oriented (verify current limits before choosing a provider):

| Piece | Likely host |
| --- | --- |
| React frontend | Vercel, Netlify, or Render static |
| Go backend | Render (or similar free web service) |
| PostgreSQL | Neon or Supabase |

OAuth callbacks and webhooks **must** use the public HTTPS URL — not localhost.

Details: [docs/deployment/README.md](docs/deployment/README.md).

**Status:** Not deployed in Phase 1.

---

## Testing (planned)

Phase 1 has no runtime code to test.

Later phases will cover:

- Unit tests for signature verification, idempotency, and rule matching
- Handler tests with forged / replayed webhook fixtures
- Integration tests against local PostgreSQL
- Manual end-to-end checklist on the live URL (OAuth → connect repo → open issue/PR → dashboard + Slack)

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
| **1 (current)** | Foundation, docs, conventions |
| **2** | Backend scaffold, Postgres access, health, config |
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

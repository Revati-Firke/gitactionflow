# High-Level Architecture (HLA)

**Status:** Phase 3 — auth implemented; repository/webhooks/rules still planned.

**Project:** GitActionFlow — event-driven automation for Git repositories.

This document is the architecture source of truth. Do not redesign without an ADR and explicit agreement.

---

## System overview

GitActionFlow is a **modular monolith**:

- A **React** dashboard for authenticated users
- A **Go** HTTP backend for OAuth, repository connection, webhooks, processing, and dashboard APIs
- **PostgreSQL** as the durable source of truth
- External systems: **GitHub** (OAuth, webhooks, REST API) and **Slack** (Incoming Webhook)
- Optional later: free-tier **AI** provider for summaries / suggestions (never required for core path)

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
        Action Results
               │
               ▼
           PostgreSQL
               │
               ▼
          React Dashboard
```

---

## Main components

| Component | Responsibility |
| --- | --- |
| React Web Dashboard | Login-gated UI: connected repo, rules, event/action history |
| Auth module | GitHub OAuth start/callback, session establishment, CSRF `state` (**implemented Phase 3**) |
| Repository management | Connect one owned repo; store webhook configuration metadata (**planned**) |
| Dashboard APIs | Read models for events, actions, rules, connection status |
| Webhook handler | Signature verify, validate, dedupe, durable persist |
| Event processor | Load pending events, apply rules, enqueue/execute actions |
| Rule engine | Match configured rules (e.g. title contains keyword) |
| Actions | GitHub label/comment; Slack notify; persist outcomes |
| Optional AI | Summarize / suggest label or priority; never block core path |
| PostgreSQL | Users, sessions, repos, rules, events, actions, failures |

---

## External systems

| System | Role |
| --- | --- |
| GitHub OAuth | User identity and authorization to act on their behalf |
| GitHub Webhooks | Push events (issues, pull_request, …) to our public endpoint |
| GitHub REST API | Labels, comments, repo metadata |
| Slack Incoming Webhook | Channel notifications |
| Optional LLM API | Stretch triage only |

---

## Authentication flow (Phase 3 — implemented)

```text
Browser
   ↓
GET /auth/github
   ↓
Generate OAuth state (random) → store hash in oauth_states (TTL, single-use)
   ↓
Redirect → GitHub authorize (scopes: read:user repo)
   ↓
GitHub callback → GET /auth/github/callback?code&state
   ↓
Validate + consume state
   ↓
Exchange code → access token (server-side only)
   ↓
GET GitHub /user → upsert users (token AES-GCM encrypted)
   ↓
Create sessions row (token hash) + Set-Cookie gaf_session (HttpOnly)
   ↓
Redirect → FRONTEND_URL
```

Protected APIs (e.g. `GET /api/me`) read the cookie, validate the session hash and expiry, and load the user. Logout deletes the session and clears the cookie.

Details: [ADR-005](../decisions/ADR-005-sessions-and-token-encryption.md).

---

## Data flow (happy path)

1. User authenticates via GitHub OAuth.
2. User connects one owned repository; webhook is registered (or configured) against the public backend URL.
3. Activity occurs on the repo → GitHub POSTs a signed webhook.
4. Backend verifies signature, validates payload, checks delivery ID, **persists** the event.
5. Processor evaluates rules and performs actions.
6. Action results are stored.
7. Dashboard reads history from PostgreSQL.

---

## Webhook flow (reliability-focused)

```text
GitHub webhook
      ↓
Validate (signature + payload)
      ↓
Deduplicate (delivery ID)
      ↓
Persist event
      ↓
Acknowledge (HTTP success after durable write)
      ↓
Process event
      ↓
Execute actions
      ↓
Persist action result / failure
```

Principles:

- Database is the durable source of truth.
- Do not rely on an in-memory queue for critical event state.
- Prefer acknowledging after durable persistence rather than waiting for all downstream side effects.
- Failures must be visible and retryable — never silently dropped.

---

## Dashboard flow

1. Browser loads React app (public static host).
2. Unauthenticated users are sent through GitHub OAuth.
3. Authenticated session cookie / token is used for dashboard API calls to the Go backend.
4. UI shows connected repository, rules CRUD (planned), and event/action logs with statuses.

---

## Security boundaries

| Boundary | Rule |
| --- | --- |
| Public webhook endpoint | Signature required; no session cookie trust |
| Dashboard APIs | Authenticated session required |
| Secrets | Server-only env; never in React bundle |
| OAuth | `state` validated; tokens stored server-side |
| Idempotency | Delivery ID uniqueness prevents duplicate side effects |

Details: root `SECURITY.md`.

---

## Reliability considerations

- Idempotent webhook handling via delivery IDs
- Persist-before-side-effects mindset
- Explicit action statuses (`pending`, `succeeded`, `failed`, etc. — exact enum later)
- Visible failure history for retries
- Structured logging without secrets (later phases)

---

## Explicit non-goals (architecture)

- Microservices split
- Kafka / Redis as primary event durability
- Multi-cloud complexity
- Non-GitHub VCS providers

---

## Related documents

- [API plan](../api/README.md)
- [Database direction](../database/README.md)
- [Deployment direction](../deployment/README.md)
- [ADR index](../decisions/README.md)

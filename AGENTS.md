# AGENTS.md — GitActionFlow

Context file for human developers and AI coding assistants working on this repository.

---

## Project purpose

**GitActionFlow** — event-driven automation for Git repositories.

Build a small, reliable product that:

- Authenticates users with GitHub OAuth
- Lets a user connect **one** repository they own
- Receives signed GitHub webhooks (issues and pull requests at minimum)
- Applies configurable rules
- Writes back to GitHub (label and/or comment)
- Sends Slack notifications
- Shows event and action history on an authenticated dashboard

This is an Abstrabit SDE1 take-home assignment. Prefer a **simple, secure, working** solution over speculative features.

---

## Scope boundaries

### In scope

- GitHub OAuth, webhooks, REST API
- Single repository per user (assignment core)
- Configurable rules (keywords, etc.)
- Slack Incoming Webhook notifications
- PostgreSQL durability, idempotency, failure visibility
- Free-tier deployment
- Optional AI (Gemini/Groq) behind an abstraction — never required for core path

### Out of scope (do not add)

- GitLab / Bitbucket support
- Multi-repository product features beyond assignment needs
- Kafka, Kubernetes, Redis, microservices
- Custom visual workflow builders
- Paid APIs or services that require a credit card
- Features not required by the assignment

If tempted to expand scope, document a decision or note it under “future work” — do not implement it silently.

---

## Architecture

**Modular monolith.** One Go process. React UI. PostgreSQL as source of truth.

Canonical flow:

```text
GitHub webhook → verify → validate → dedupe by delivery ID → persist event
  → process → rule engine → actions (GitHub API, Slack) → persist results
  → dashboard reads from PostgreSQL
```

Architecture source of truth: `docs/architecture/HLA.md`.

Do not redesign the architecture without an ADR and explicit human agreement.

---

## Technology stack

| Area | Stack |
| --- | --- |
| Backend | Go, Gin, pgx, PostgreSQL, golang-migrate |
| Frontend | React, TypeScript, Vite |
| Integrations | GitHub OAuth / Webhooks / REST, Slack Incoming Webhook |
| Local data | Docker Compose PostgreSQL (+ optional backend service) |
| Optional AI | Gemini or Groq (stretch only) |

---

## Backend conventions (Phase 2+)

- Module path: `github.com/Revati-Firke/gitactionflow/backend`
- Entrypoint: `cmd/server`
- Domain/infra code under `internal/`
- Prefer `pgx` + explicit SQL; no ORM without an ADR
- Config via environment variables (`internal/config`); fail fast on missing required values
- Structured logs via `log/slog` JSON; never log secrets (use `Config.Redacted()`)
- HTTP errors use `{ "error": { "code", "message" } }` (`internal/http/response`)
- Keep handlers thin; inject dependencies (e.g. `Pinger` for readiness)
- SQL migrations live in `backend/migrations/` (`*.up.sql` / `*.down.sql`)
- Do not expand scope into webhooks/Slack/AI/dashboard until that phase begins

---

## Authentication rules (Phase 3+)

- Validate OAuth `state` (CSRF); states expire and are single-use
- Use HttpOnly session cookies; never put GitHub tokens in localStorage/URL
- Store session token hashes and encrypted GitHub access tokens only
- Never log or return provider tokens, client secrets, or session secrets
- Do not trust client-supplied user IDs when a session is available
- Keep GitHub OAuth client code isolated (`internal/githuboauth`)
- Test security-sensitive auth behavior with mocks (no real GitHub account required)

---

## Repository management rules (Phase 4+)

- Never trust repository metadata from the client; re-fetch from GitHub by id
- Require GitHub `admin` permission before connecting
- Scope all repository queries to the authenticated user id from the session
- Enforce one connected repository per user (disconnect before switching)
- Never expose GitHub tokens to the frontend or logs
- Keep GitHub REST calls behind `internal/githubapi`; keep handlers thin
- Do not register or delete webhooks until the webhook phase

---

## Webhook rules (Phase 5+)

- Always verify `X-Hub-Signature-256` over the **raw** request body
- Use constant-time comparison (`hmac.Equal`)
- Treat `X-GitHub-Delivery` as the idempotency key (DB unique constraint)
- Never process/ack a delivery that was not durably persisted (DB errors → 5xx)
- Never run downstream actions inside the webhook request
- Never log webhook secrets or signature material
- Bound webhook body size

---

## Event processing rules (Phase 6+)

- PostgreSQL is the durable processing queue (no Redis/Kafka for this assignment)
- Claim events with transactions + `FOR UPDATE SKIP LOCKED`
- Lifecycle: `pending` → `processing` → `processed` | retry `pending` | `failed`
- Bound retries (`EVENT_MAX_RETRIES`); persist `last_error` / `failed_at`
- Recover stale `processing` via `locked_at` lease (`EVENT_PROCESSING_LEASE`)
- Worker starts/stops with the backend (context cancel on shutdown)
- Do not implement dashboard or AI until those phases
- Never log secrets or full webhook payloads

---

## Rule engine rules (Phase 7+)

- Conditions within a rule are **AND**; unspecified conditions do not restrict
- Supported conditions: event_type, keyword (title/body, case-insensitive), author (login, case-insensitive), required_labels (all must be present)
- Disabled rules never match; multiple enabled rules may match one event (deterministic order: created_at, id)
- Rules are scoped to the authenticated user's **connected** repository — never trust client `repository_id`
- Rule evaluation produces **action intents** — execution is a separate component (`internal/actions`)
- Valid action types: `github_label`, `github_comment`, `slack_notification`

---

## Action execution rules (Phase 8+)

- Persist an `actions` row **before** any GitHub or Slack HTTP call
- Idempotency key `event_id:rule_id:action_type` with a UNIQUE DB constraint (source of truth)
- Never perform external HTTP inside an open database transaction
- Classify errors: retry 429/5xx/network; do not endlessly retry 401/403/404/422/config errors
- Reuse `events.RetryBackoff` for action retries; bound with `ACTION_MAX_RETRIES`
- Slack webhook URL only from `SLACK_WEBHOOK_URL` env — never from rule config, client, logs, or API responses
- GitHub writes use the connected user's decrypted server-side token via `internal/githubapi`
- Operate only on the connected repository (stable GitHub repo id); issues and PRs share issue-number APIs
- Do not mark the parent event `processed` while required actions are pending or failed
- Exactly-once external delivery is not guaranteed; design minimizes duplicates (see ADR-008)
- Keep integrations behind small interfaces for tests; no real GitHub/Slack in unit tests
- Do not implement dashboard or AI until those phases

---

## Before modifying code

1. Inspect the existing implementation and understand the architecture.
2. Prefer small, focused changes over rewrites.
3. Do not overwrite useful files without reading them first.
4. Do not introduce unnecessary dependencies.
5. Do not start a later phase automatically when finishing an earlier one.
6. If a design decision is necessary, document it (ADR or docs) rather than silently changing architecture.

---

## Coding principles

- Prefer readable Go over clever Go.
- Keep functions focused.
- Keep interfaces small.
- Avoid premature abstraction.
- Avoid unnecessary dependencies.
- Validate external input.
- Handle errors explicitly.
- Never ignore errors without a clear reason.
- Do not log secrets.
- Write tests for important behavior.
- Keep configuration environment-based.
- Use `context.Context` appropriately.
- Keep business logic separate from HTTP handlers.
- Keep external integrations isolated behind clear boundaries.
- Update documentation when architecture or behavior changes.

### Frontend

- TypeScript strictness; clear separation of API client vs UI.
- Do not put secrets in client code or Vite env vars that are exposed to the browser.

---

## Security rules

- Verify GitHub webhooks with `X-Hub-Signature-256` (HMAC-SHA256, constant-time compare).
- Persist GitHub delivery IDs; duplicate deliveries must not re-execute side effects.
- Validate OAuth `state` against CSRF.
- Never expose to the frontend or commit:
  - GitHub OAuth client secret
  - GitHub access tokens
  - Webhook secrets
  - Slack webhook URLs
  - AI API keys
  - Database credentials
- Never log the above.
- Provide `.env.example` with placeholders only — no real secrets.

---

## Reliability rules

```text
Webhook → validate → dedupe → persist event → process → execute actions → persist action result
```

- Acknowledge receipt after **durable persistence**, not after every downstream side effect completes synchronously.
- Database is the durable source of truth — not an in-memory queue for critical event state.
- Failures must be visible and eventually retryable.
- Do not silently drop events when Slack or GitHub is briefly unavailable.

---

## Documentation rules

- Distinguish **implemented** vs **planned**.
- Do not claim features work when they do not.
- Keep HLA, API, database, and deployment docs aligned with reality.
- Update docs in the same change when behavior or architecture shifts.

---

## Testing expectations

- Unit-test signature verification, idempotency, and rule matching.
- Cover forged and replayed webhook cases with fixtures.
- Prefer focused tests over brittle end-to-end-only coverage.
- Manual E2E checklist required before submission (live URL).

---

## Git commit conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
feat:     new user-facing capability
fix:      bug fix
docs:     documentation only
test:     tests only
refactor: internal change without behavior change
chore:    tooling, deps, scaffolding
security: security-related change
```

Examples:

```text
docs: add initial project architecture
feat: verify GitHub webhook signatures
fix: skip duplicate webhook deliveries by delivery ID
```

Avoid meaningless messages (`update`, `changes`, `final`, `test`, `stuff`).

---

## Important constraints

- Free tiers only; no credit card.
- One connected repository is enough for the assignment core.
- AI is optional and must not block the core path.
- Phase discipline: complete the current phase; stop; do not auto-start the next.
- Keep the submission simple, reliable, secure, and well documented.

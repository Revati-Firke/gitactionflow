# Database Design Direction

**Status:** Direction only — **no schema, migrations, or tables** in Phase 1.

PostgreSQL is the durable source of truth for GitActionFlow (see ADR-002 placeholder in `docs/decisions/`). Access from Go will use **pgx**.

Do not over-design. Exact columns, indexes, and constraints will be defined when migrations are introduced.

---

## Goals

- Persist users and sessions from GitHub OAuth
- Track one connected repository (assignment core)
- Store configurable rules
- Record webhook deliveries idempotently
- Record actions and failures for dashboard visibility and retries
- Never store secrets in plaintext in ways that end up in client responses or logs (token storage strategy to be decided carefully later)

---

## Likely entities

### users

Purpose: identity linked to GitHub account.

Likely concepts: GitHub user id, login/name/avatar metadata, timestamps. Access tokens belong server-side (same table or related credentials table — decide at migration time).

### sessions

Purpose: authenticated dashboard access after OAuth.

Likely concepts: session id / token hash, user reference, expiry, created_at.

### repositories

Purpose: the GitHub repository connected for automation.

Likely concepts: GitHub repo id, owner/name, webhook id (if registered via API), user reference, active flag. Webhook **secret** must be handled carefully (encrypted or host secret reference — decide later; never expose to frontend).

### rules

Purpose: user-configured match → action definitions.

Likely concepts: repository reference, enabled flag, match criteria (e.g. event type, title keyword), action instructions (add label name, comment template, notify Slack). Keep the model simple — not a general workflow engine.

### webhook_events

Purpose: durable record of each GitHub delivery.

Likely concepts: delivery id (**unique** idempotency key), event type, action subtype, repository reference, payload (or summarized payload), received_at, processing status.

Duplicate deliveries with the same delivery id must not create duplicate processing side effects.

### actions

Purpose: record of side effects the bot attempted or completed.

Likely concepts: related event, type (`github_label`, `github_comment`, `slack_notify`, …), status, attempt count, response summary / error, timestamps.

### failures / retries

Purpose: make unhappy paths visible and recoverable.

May be modeled as:

- Status + error fields on `actions` / `webhook_events`, and/or
- A dedicated failure / retry table

Exact shape deferred. Requirement: failures must not be silent; retries must be possible later.

---

## Reliability mapping

```text
Persist webhook_events (unique delivery id)
  → process
  → persist actions (success or failure)
```

Critical event state lives in PostgreSQL — not only in memory.

---

## What Phase 1 does not include

- SQL migration files
- Final ERD
- Indexes, enums, or constraint DDL
- ORM choice beyond “use pgx”

When schema work begins, add migrations under the backend and update this document with the concrete model.

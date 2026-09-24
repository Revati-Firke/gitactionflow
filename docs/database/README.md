# Database

**Status:** Phase 2 — connectivity, migrations, and a minimal foundation table. Full product schema is **not** created yet.

PostgreSQL is the durable source of truth for GitActionFlow (ADR-002). Access from Go uses **pgx** (`pgxpool`). No ORM.

---

## Connection approach

- Configuration: `DATABASE_URL` (required).
- Pool: `pgxpool` with modest defaults (max 10 connections, connect timeout 5s).
- Operations are context-aware; the pool is closed on process shutdown.
- `/ready` pings the pool to verify dependency health.

Never expose `DATABASE_URL` or passwords in HTTP responses or logs (URLs are redacted in startup logs).

---

## Migration approach

- Tool: [golang-migrate](https://github.com/golang-migrate/migrate)
- Files: `backend/migrations/*.up.sql` / `*.down.sql`
- Embedded into the binary via `backend/migrations` package (`embed`)
- Applied on startup when `AUTO_MIGRATE=true` (default in `development` / `test`)

Version history is tracked in PostgreSQL table `schema_migrations` (managed by golang-migrate).

### Current migrations

| Version | Name | Purpose |
| --- | --- | --- |
| 000001 | foundation | Creates `app_meta` key/value table to prove migrate up/down works |

`app_meta` is **not** the application domain model — only a Phase 2 smoke-test table.

---

## Future schema direction

Likely entities (unchanged from Phase 1 planning; still not implemented):

| Entity | Purpose |
| --- | --- |
| users | GitHub identity |
| sessions | Dashboard auth |
| repositories | One connected repo per user (core) |
| rules | Match → action configuration |
| webhook_events | Durable deliveries; unique delivery ID |
| actions | GitHub / Slack side-effect results |
| failures/retries | Visible unhappy paths |

Exact columns and constraints will be added in the phase that needs them.

---

## Reliability mapping (planned)

```text
Persist webhook_events (unique delivery id)
  → process
  → persist actions (success or failure)
```

Critical event state will live in PostgreSQL — not only in memory.

# Database

**Status:** Phase 3 — foundation + authentication tables. Product tables (repos, rules, events, actions) are **not** created yet.

PostgreSQL via **pgx** pool. Migrations: **golang-migrate** embedded from `backend/migrations/`.

---

## Connection

- `DATABASE_URL` (required)
- Pool closed on shutdown
- `/ready` pings the pool
- Credentials never returned in HTTP responses; URLs redacted in logs

---

## Migrations

| Version | Name | Purpose |
| --- | --- | --- |
| 000001 | foundation | `app_meta` smoke-test table |
| 000002 | auth | `users`, `sessions`, `oauth_states` |

Applied on startup when `AUTO_MIGRATE=true`.

---

## Implemented tables

### `users`

| Column | Notes |
| --- | --- |
| `id` | UUID PK |
| `github_user_id` | Unique GitHub numeric ID |
| `github_username` | Login |
| `display_name` | Name or login fallback |
| `avatar_url` | Avatar |
| `github_access_token_encrypted` | AES-GCM ciphertext (never plaintext) |
| `created_at` / `updated_at` | Timestamps |

### `sessions`

| Column | Notes |
| --- | --- |
| `id` | UUID PK |
| `user_id` | FK → users |
| `token_hash` | SHA-256 of cookie token (unique) |
| `expires_at` | Session expiry |
| `created_at` | Created |

### `oauth_states`

| Column | Notes |
| --- | --- |
| `state_hash` | SHA-256 of OAuth state (PK) |
| `expires_at` | Short TTL |
| `consumed_at` | Set on single-use consume |
| `created_at` | Created |

### `app_meta`

Phase 2 foundation table only.

---

## Future schema (not implemented)

`repositories`, `rules`, `webhook_events`, `actions` — later phases.

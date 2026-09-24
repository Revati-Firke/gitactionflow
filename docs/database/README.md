# Database

**Status:** Phase 4 — foundation, auth, and `repositories`. Webhook/event tables are **not** created yet.

PostgreSQL via **pgx**. Migrations via **golang-migrate** (`backend/migrations/`).

---

## Relationship

```text
users
  │ 1
  │
  └── repositories   (at most one row per user)
```

---

## Implemented tables

### `users` / `sessions` / `oauth_states`

See Phase 3. GitHub access tokens stay encrypted on `users`.

### `repositories`

| Column | Notes |
| --- | --- |
| `id` | UUID PK |
| `user_id` | FK → users, **UNIQUE** (one connected repo per user) |
| `github_repository_id` | GitHub numeric id |
| `name`, `full_name`, `owner_login` | From GitHub at connect time |
| `default_branch`, `html_url`, `private` | From GitHub |
| `created_at`, `updated_at` | Timestamps |

Constraints:

- `UNIQUE (user_id)` — one connected repository per user
- `UNIQUE (user_id, github_repository_id)` — no duplicate pair

### `app_meta`

Phase 2 smoke-test table.

---

## Migrations

| Version | Name |
| --- | --- |
| 000001 | foundation |
| 000002 | auth |
| 000003 | repositories |

---

## Future (not implemented)

`rules`, `webhook_events`, `actions`

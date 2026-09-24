# Database

**Status:** Phase 6 — foundation, auth, repositories, webhook ingest, and durable event processing.

PostgreSQL via **pgx**. Migrations via **golang-migrate** (`backend/migrations/`).

---

## Relationship

```text
users
  │ 1
  │
  └── repositories          (at most one row per user)
          │ 1
          │
          └── webhook_events   (many deliveries; unique delivery_id)
```

---

## Implemented tables

### `users` / `sessions` / `oauth_states`

Phase 3. GitHub access tokens encrypted on `users`.

### `repositories`

Phase 4. `UNIQUE (user_id)` — one connected repo per user.

### `webhook_events`

| Column | Notes |
| --- | --- |
| `id` | UUID PK |
| `repository_id` | FK → repositories |
| `delivery_id` | GitHub `X-GitHub-Delivery`, **UNIQUE** (ingest idempotency) |
| `event_type` | e.g. `issues`, `pull_request` |
| `action` | e.g. `opened` |
| `payload` | JSONB raw body |
| `status` | `pending` \| `processing` \| `processed` \| `failed` |
| `retry_count` | Failures so far (Phase 6) |
| `max_retries` | Bound from `EVENT_MAX_RETRIES` at insert |
| `next_retry_at` | When a `pending` retry becomes eligible |
| `last_error` | Last processing error message |
| `locked_at` | Lease timestamp while `processing` |
| `received_at` | Ingest time |
| `processed_at` | Set when `processed` |
| `failed_at` | Set when `failed` |

#### Status meanings

| Status | Meaning |
| --- | --- |
| `pending` | Accepted; waiting for worker (or scheduled retry) |
| `processing` | Claimed by a worker under a lease |
| `processed` | Pipeline validation completed (no rules/actions yet) |
| `failed` | Retries exhausted or permanent validation error |

#### Lifecycle

```text
pending → processing → processed
                  ↘ pending (retry) → …
                  ↘ failed
```

Stale `processing` (`locked_at` older than lease) can be reclaimed; `retry_count` increments.

---

## Migrations

| Version | Name |
| --- | --- |
| 000001 | foundation |
| 000002 | auth |
| 000003 | repositories |
| 000004 | webhook_events |
| 000005 | event_processing |

---

## Future (not implemented)

`rules`, `actions` (side-effect results)

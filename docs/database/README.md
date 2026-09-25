# Database

**Status:** Current — users, repositories, webhook events, rules, and durable actions.

PostgreSQL via **pgx**. Migrations via **golang-migrate** (`backend/migrations/`).

---

## Relationship

```text
users
  │ 1
  │
  └── repositories          (at most one row per user)
          │ 1
          ├── webhook_events   (unique delivery_id)
          │         │
          │         └── actions   (unique idempotency_key)
          └── rules            (many per repository)
```

---

## `actions`

Durable side-effect records created **before** GitHub/Slack calls.

| Column | Notes |
| --- | --- |
| `id` | UUID PK |
| `event_id` | FK → webhook_events (CASCADE) |
| `rule_id` | FK → rules (SET NULL on delete) |
| `action_type` | `github_label` \| `github_comment` \| `slack_notification` |
| `action_config` | JSONB from the rule (no secrets) |
| `idempotency_key` | UNIQUE; typically `event_id:rule_id:action_type` |
| `status` | `pending` → `processing` → `completed` \| `failed` |
| `attempt_count` / `max_attempts` | Bounded retries |
| `next_retry_at` | Backoff schedule |
| `last_error` | Sanitized error text |
| `started_at` / `completed_at` / `failed_at` | Lifecycle timestamps |
| `created_at` / `updated_at` | Timestamps |

Lifecycle: create pending → mark processing → external call → completed, or retry/failed. Rows are never deleted on failure.

---

## `rules`

| Column | Notes |
| --- | --- |
| `id` | UUID PK |
| `repository_id` | FK → repositories (CASCADE) |
| `name` | Display name |
| `enabled` | Disabled rules never match |
| `event_type` | `issues` \| `pull_request` |
| `keyword` | Optional; substring in title/body (case-insensitive) |
| `author` | Optional; GitHub login (case-insensitive) |
| `required_labels` | `TEXT[]`; all must be present on the event |
| `action_type` | `github_label` \| `github_comment` \| `slack_notification` |
| `action_config` | JSONB (e.g. `{"label":"automation"}`) |
| `created_at` / `updated_at` | Timestamps |

Evaluation loads `enabled = true` for the event's repository, ordered by `created_at ASC, id ASC`.

---

## `webhook_events`

Ingest + processing fields from Phases 5–6 (`pending` / `processing` / `processed` / `failed`, retries, lease). An event is marked `processed` only after all related actions complete successfully.

---

## Migrations

| Version | Name |
| --- | --- |
| 000001 | foundation |
| 000002 | auth |
| 000003 | repositories |
| 000004 | webhook_events |
| 000005 | event_processing |
| 000006 | rules |
| 000007 | actions |

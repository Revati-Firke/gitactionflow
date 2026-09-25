# API

**Status:** Current — health, OAuth, repository, webhooks, rules, actions, dashboard history.

Base URL: Go server root (default `http://localhost:8080`).

---

## Implemented

### Health

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/health` | no |
| GET | `/ready` | no |

### Authentication

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/auth/github` | no |
| GET | `/auth/github/callback` | no |
| POST | `/auth/logout` | cookie optional |
| GET | `/api/me` | session required |

`/api/me` never returns tokens. OAuth details: [ADR-005](../decisions/ADR-005-sessions-and-token-encryption.md).

### Repository management

All routes require a valid session cookie. User identity comes from the session — never from the request body.

#### `GET /api/github/repositories`

Lists repositories from GitHub for the signed-in user (not persisted).

**200**

```json
{
  "repositories": [
    {
      "id": 123456,
      "name": "example-project",
      "full_name": "username/example-project",
      "private": false,
      "default_branch": "main",
      "html_url": "https://github.com/username/example-project",
      "owner_login": "username"
    }
  ]
}
```

#### `GET /api/repository`

Returns the connected repository for the current user.

**200** `{ "repository": { ... } }`  
**404** `REPOSITORY_NOT_CONNECTED`

#### `POST /api/repository`

Connect one repository. Body:

```json
{ "github_repository_id": 123456 }
```

Backend re-fetches the repo from GitHub and requires **admin** permission. Client-supplied name/owner fields are ignored (not accepted).

**201** `{ "repository": { ... } }`  
**400** `INVALID_REPOSITORY_ID`  
**403** `REPOSITORY_ACCESS_DENIED`  
**404** `REPOSITORY_NOT_FOUND`  
**409** `REPOSITORY_ALREADY_CONNECTED` — disconnect first  
**401** `GITHUB_UNAUTHORIZED` — re-login  
**429** `GITHUB_RATE_LIMITED`

#### `DELETE /api/repository`

Disconnects the local connection (idempotent). Does **not** delete GitHub webhooks automatically.

### Webhooks

#### `POST /webhooks/github`

Public endpoint. Authenticated by **HMAC signature**, not session cookies.

**Required headers**

| Header | Purpose |
| --- | --- |
| `X-Hub-Signature-256` | `sha256=<hex>` HMAC of **raw** body with `GITHUB_WEBHOOK_SECRET` |
| `X-GitHub-Event` | Event name |
| `X-GitHub-Delivery` | Unique delivery id (idempotency key) |

**Supported events (persisted):** `issues`, `pull_request` (all actions).  
**Other signed events:** `200` `{ "status":"ignored", "reason":"unsupported_event" }` (not persisted).

**Body limit:** `WEBHOOK_MAX_BODY_BYTES` (default 1 MiB) → `413` if exceeded.

**Responses**

| Case | HTTP | Body |
| --- | --- | --- |
| Persisted | 200 | `{ "status":"accepted" }` |
| Duplicate delivery id | 200 | `{ "status":"already_received" }` |
| Unknown / not connected repo | 200 | `{ "status":"ignored", "reason":"unknown_repository" }` |
| Invalid / missing signature | **401** | `{ "error": { "code":"UNAUTHORIZED", ... } }` |
| Missing event/delivery headers | 400 | structured error |
| Invalid JSON / missing repo id | 400 | structured error |
| DB failure | **500** | so GitHub can retry |

New rows are stored with status **`pending`**. A background worker claims them, evaluates rules, executes matched actions (GitHub/Slack), then marks `processed` or retries/`failed`.

---

### Rules

All routes require a session cookie. Repository is derived from the authenticated user's connected repo (never from the client).

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/api/rules` | List rules for connected repository |
| POST | `/api/rules` | Create rule |
| GET | `/api/rules/:id` | Get one rule |
| PUT | `/api/rules/:id` | Update rule |
| DELETE | `/api/rules/:id` | Delete rule |

**Create/update body (example)**

```json
{
  "name": "Bug Issues",
  "enabled": true,
  "event_type": "issues",
  "keyword": "bug",
  "author": "",
  "required_labels": [],
  "action_type": "github_label",
  "action_config": { "label": "automation" }
}
```

Conditions within a rule are **AND**. Empty keyword/author are treated as unspecified.  
`action_type`: `github_label` | `github_comment` | `slack_notification`.

| Case | HTTP |
| --- | --- |
| Success list/create/update/get | 200 / 201 |
| Invalid input | 400 `INVALID_RULE` |
| Unauthenticated | 401 |
| No connected repository / rule not found | 404 |

---

### Dashboard activity

Session required. Results are scoped to the authenticated user’s **connected** repository. Raw webhook payloads are never returned.

#### `GET /api/events`

Query: `page` (default 1), `limit` (default 20, max 100).

**200**

```json
{
  "events": [
    {
      "id": "…",
      "event_type": "issues",
      "action": "opened",
      "status": "processed",
      "delivery_id": "…",
      "retry_count": 0,
      "last_error": null,
      "received_at": "…",
      "processed_at": "…",
      "failed_at": null
    }
  ],
  "page": { "limit": 20, "offset": 0 }
}
```

**404** `REPOSITORY_NOT_CONNECTED`

#### `GET /api/actions`

Same pagination. Includes optional `rule_name` when the rule still exists.

**200** `{ "actions": [ { "id", "event_id", "rule_id", "rule_name", "action_type", "status", "attempt_count", "max_attempts", "last_error", "created_at", "completed_at", "failed_at" } ], "page": { … } }`

**404** `REPOSITORY_NOT_CONNECTED`

---

## Error envelope

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "internal server error"
  }
}
```

---

## Planned (not implemented)

Optional AI stretch; automatic webhook registration on connect.
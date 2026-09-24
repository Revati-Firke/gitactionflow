# API

**Status:** Phase 6 — health, OAuth, repository connection, webhook ingestion, and background event processing. Rules/actions remain planned.

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

See Phase 3 docs in git history for OAuth details. `/api/me` never returns tokens.

### Repository management (Phase 4)

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

### Webhooks (Phase 5)

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

New rows are stored with status **`pending`**. A background worker then claims them → `processed` (validation only) or retries/`failed`. HTTP webhook behavior is unchanged. No GitHub/Slack actions run yet.

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

Rules, GitHub/Slack actions, dashboard aggregations.
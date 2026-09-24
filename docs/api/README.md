# API

**Status:** Phase 2 — health/readiness implemented; all other categories remain planned.

Base URL: the Go server root (default `http://localhost:8080`). Dashboard routes will later live under `/api` (exact prefix TBD).

---

## Implemented

### `GET /health`

Liveness. Confirms the process is running. No authentication. Does not check PostgreSQL.

**200**

```json
{ "status": "ok" }
```

### `GET /ready`

Readiness. Pings PostgreSQL via the connection pool.

**200** when the database is reachable:

```json
{ "status": "ready" }
```

**503** when the database is unavailable (no credentials or internal DB errors are exposed):

```json
{
  "error": {
    "code": "NOT_READY",
    "message": "database unavailable"
  }
}
```

### Error envelope (shared)

Client-facing errors use:

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "internal server error"
  }
}
```

Internal details stay in server logs only.

---

## Planned categories (not implemented)

```text
Authentication
Repositories
Rules
Webhooks
Events
Actions
Dashboard
```

### Authentication (planned)

| Concern | Direction |
| --- | --- |
| Start OAuth | Redirect user to GitHub authorize URL with `state` |
| OAuth callback | Exchange code, validate `state`, create session |
| Logout | Invalidate session |
| Current user | Return authenticated profile (no secrets) |

### Repositories (planned)

| Concern | Direction |
| --- | --- |
| List owned repos | Via GitHub API using stored user token (server-side) |
| Connect repository | Persist one connected repo; configure webhook |
| Connected status | Return current connection metadata (no secrets) |
| Disconnect | Remove connection / disable webhook (later) |

Assignment core: **one** connected repository per user is sufficient.

### Rules (planned)

| Concern | Direction |
| --- | --- |
| List rules | Rules for the connected repository |
| Create / update / delete | Simple matchers (e.g. title contains keyword → label + Slack) |
| Enable / disable | Soft control without deleting history |

### Webhooks (planned)

| Concern | Direction |
| --- | --- |
| Ingest endpoint | Public `POST` for GitHub deliveries |
| Auth model | `X-Hub-Signature-256` — not session cookies |
| Events | At least `issues` and `pull_request` |
| Response | Success after durable persistence of the delivery |

### Events / Actions / Dashboard (planned)

Read models for webhook history, action outcomes, and a thin summary aggregation — all behind authentication once Phase 3+ lands.

---

## Cross-cutting

- JSON request/response
- Consistent error shape (implemented)
- No secret fields in responses
- CSRF protection via OAuth `state` (planned)

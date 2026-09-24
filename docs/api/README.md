# API

**Status:** Phase 4 — health, OAuth auth, and repository connection implemented. Webhooks/rules/actions remain planned.

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

Disconnects the local connection (idempotent). Does **not** delete GitHub webhooks (none registered yet).

**200** `{ "status": "ok" }`

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

Webhooks, events, actions, rules, dashboard aggregations.

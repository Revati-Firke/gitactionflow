# API

**Status:** Phase 3 — health/readiness + GitHub OAuth auth implemented. Repository/webhooks/rules remain planned.

Base URL: Go server root (default `http://localhost:8080`).

---

## Implemented

### `GET /health`

Liveness. No authentication.

**200** `{"status":"ok"}`

### `GET /ready`

Readiness (PostgreSQL ping). No authentication.

**200** `{"status":"ready"}`  
**503** structured `NOT_READY` if database unavailable

### `GET /auth/github`

Starts GitHub OAuth. No session required.

- Generates cryptographically random `state`, stores hash with TTL
- Redirects (`302`) to GitHub authorize URL

### `GET /auth/github/callback`

OAuth callback from GitHub. No session required.

Query params: `code`, `state` (or `error` on denial).

- Validates/consumes `state` (reject missing, invalid, expired, reused)
- Exchanges code, fetches GitHub user, upserts local user
- Creates server-side session + `HttpOnly` cookie `gaf_session`
- Redirects to `FRONTEND_URL` (on failure, redirects with `?error=...`)

Never returns GitHub access tokens in the response body or redirect URL.

### `POST /auth/logout`

Invalidates the current session if present and clears the cookie. **Idempotent.**

**200** `{"status":"ok"}`

### `GET /api/me`

Requires valid session cookie.

**200**

```json
{
  "user": {
    "id": "...",
    "github_username": "...",
    "display_name": "...",
    "avatar_url": "..."
  }
}
```

**401**

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "authentication required"
  }
}
```

Does not include access tokens or secrets.

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

```text
Authentication extras beyond above — none required
Repositories
Rules
Webhooks
Events
Actions
Dashboard
```

See earlier Phase 1 plans for direction. Do not treat those routes as available.

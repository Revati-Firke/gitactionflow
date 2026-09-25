# ADR-005 — Server-side sessions and encrypted GitHub tokens

**Status:** Accepted  
**Date:** 2026-09-24

## Context

GitActionFlow needs GitHub OAuth for dashboard access and a durable GitHub access token for repository API calls (labels, comments). Browser storage of tokens is unsafe. Plaintext token storage in PostgreSQL is below the quality bar for this assignment.

Frontend (Vite, `:5173`) and backend (`:8080`) are different origins during local development.

## Decision

1. **Server-side sessions** in PostgreSQL. The browser only receives an opaque session cookie (`gaf_session`). The DB stores a **SHA-256 hash** of the session token, never the raw value.
2. **Cookie flags:** `HttpOnly`; `Secure` in production (`COOKIE_SECURE`); default `SameSite=Lax` locally. Production prefers a **same-origin Vercel proxy** so the cookie is first-party; `SameSite=None; Secure` remains available if calling the API cross-origin.
3. **GitHub access tokens** live in `users.github_access_token_encrypted` using **AES-256-GCM**, key derived as `SHA-256(SESSION_SECRET)`. Tokens are never returned by `/api/me` or logged.
4. **OAuth CSRF `state`** is cryptographically random; only its hash is stored in `oauth_states`, with short TTL and single-use consume.
5. **OAuth scopes:** `read:user repo` — identify the user and automate owned repos (including private). No admin/org scopes.

## Consequences

- `SESSION_SECRET` is required and sensitive (session integrity + token encryption).
- Rotating `SESSION_SECRET` invalidates encrypted tokens and requires re-login (acceptable here).
- Cross-origin cookie pitfalls drove the Vercel proxy decision documented in `AI_NOTES.md`.

# ADR-005 — Server-side sessions and encrypted GitHub tokens

**Status:** Accepted  
**Date:** 2026-09-24  
**Phase:** 3

## Context

GitActionFlow needs GitHub OAuth for dashboard access. Later phases need the user's GitHub access token for repository API calls (webhooks, labels, comments). Browser storage of tokens is unsafe. Plaintext token storage in PostgreSQL is unacceptable for the assignment quality bar.

Frontend (Vite, `:5173`) and backend (`:8080`) are different origins during local development.

## Decision

1. **Server-side sessions** in PostgreSQL. The browser only receives an opaque session cookie (`gaf_session`). The DB stores a **SHA-256 hash** of the session token, never the raw value.
2. **Cookie flags:** `HttpOnly`; `Secure` in production (`COOKIE_SECURE`); default `SameSite=Lax` so the top-level OAuth redirect callback can set the cookie. Document that a cross-origin SPA should use a same-origin reverse proxy locally, or `SameSite=None; Secure` behind HTTPS in production.
3. **GitHub access tokens** are stored in `users.github_access_token_encrypted` using **AES-256-GCM**, with the key derived as `SHA-256(SESSION_SECRET)`. Tokens are never returned by `/api/me` or logged.
4. **OAuth CSRF `state`** is a cryptographically random value; only its hash is stored in `oauth_states`, with short TTL and single-use consume.
5. **OAuth scopes:** `read:user repo` — identify the user and allow later repository automation on owned repos (including private). No admin/org scopes.

## Consequences

- `SESSION_SECRET` is required and sensitive (session integrity + token encryption).
- Rotating `SESSION_SECRET` invalidates encrypted tokens and requires re-login (acceptable for this assignment).
- Cross-origin cookie pitfalls are documented; Phase 3 verification can use cookie-jar / redirect flows without a full SPA.

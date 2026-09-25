# Security

Security model for the deployed GitActionFlow application (OAuth App + modular monolith).

This document describes **implemented** controls. It is not a formal security certification.

## Reporting

If you find a vulnerability or a committed secret, rotate credentials immediately and notify the repository owner. Do not open a public issue that includes secret values.

## Authentication

- GitHub OAuth App login with cryptographically random, single-use, TTL-bound `state` (CSRF for login).
- Server-side sessions: raw session token in an **HttpOnly** cookie (`gaf_session`); only a hash is stored in PostgreSQL.
- Production defaults: `Secure` + `SameSite=None` (cross-site capable). Preferred production hosting uses a **Vercel same-origin proxy** for `/api` and `/auth` so the cookie is first-party on the frontend host.
- Logout deletes the session row and clears the cookie.
- GitHub access tokens are encrypted at rest (`SESSION_SECRET`-derived key) and never returned to the browser or logged.

## Authorization / isolation

- Authenticated APIs derive the user from the session cookie — never from a client-supplied `user_id`.
- Connected repository, rules, events, and actions are scoped to the current user’s connected repository.
- One connected repository per user (DB unique constraint).
- GitHub write actions use the connected repo’s owner/name from server-side state, not client-supplied repo identity.

## CSRF (cookie APIs)

- Mutating `/api/*` and `POST /auth/logout` reject `Origin` values that do not match configured `FRONTEND_URL` (normalized).
- `POST /webhooks/github` is **not** Origin-gated; authenticity is HMAC signature based.
- OAuth `GET` start/callback rely on OAuth `state`, not Origin CSRF middleware.

## Webhooks

- `X-Hub-Signature-256` HMAC-SHA256 over the **raw** body; compared with `hmac.Equal`.
- Request body size bounded (`WEBHOOK_MAX_BODY_BYTES`).
- `X-GitHub-Delivery` is the idempotency key (`UNIQUE` in PostgreSQL).
- Unknown / unconnected repositories are ignored without side effects.
- Webhook secret is never logged.

## Idempotency

- Events: unique delivery ID.
- Actions: unique `(event_id, rule_id, action_type)` / idempotency key before external HTTP.
- Duplicate deliveries must not create duplicate logical actions.

## Secrets

Never expose to frontend, bundles, API responses, or logs:

- `GITHUB_CLIENT_SECRET`, GitHub access tokens, `GITHUB_WEBHOOK_SECRET`
- `SESSION_SECRET`, `DATABASE_URL` passwords, `SLACK_WEBHOOK_URL`
- `GEMINI_API_KEY` / `GROQ_API_KEY` / `AI_API_KEY`

Config startup logging uses `Config.Redacted()`.

## Browser security headers

API responses set conservative headers including `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, and a restrictive API `Content-Security-Policy`.

## Rate limiting

- Lightweight **per-process** in-memory limits on OAuth and `/api` routes.
- GitHub webhooks are **not** rate-limited in-app (avoid dropping legitimate deliveries).
- Not a distributed control on multi-instance hosts.

## Optional AI

- Disabled by default (`AI_ENABLED=false`).
- Model output is schema-validated and allowlisted; it cannot authorize or call arbitrary URLs.
- AI failure skips enrichment; core label/comment/Slack config still applies when present.

## Known limitations

- In-memory rate limits are per instance.
- Exactly-once external delivery is not guaranteed; design minimizes duplicates (see ADR-008).
- GitHub App installation auth is **not** used; OAuth App user tokens power API writes.
- Automatic webhook registration on connect is **not** implemented (manual repo webhook).

## CORS

- Reflects only the configured `FRONTEND_URL` origin with credentials — never `*`.

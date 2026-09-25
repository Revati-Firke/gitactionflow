# Security

Controls that are actually in the running app (OAuth App + modular monolith). Not a formal audit.

## Reporting

If you find a vulnerability or a committed secret, rotate credentials and tell the repo owner. Do not file a public issue that includes secret values.

## Authentication

- GitHub OAuth with random, single-use, TTL-bound `state`
- Session token in an HttpOnly cookie (`gaf_session`); only a hash in Postgres
- Production prefers a **Vercel same-origin proxy** for `/api` and `/auth` so the cookie is first-party on the frontend host (avoids third-party cookie blocks)
- Logout deletes the session row and clears the cookie
- GitHub access tokens encrypted at rest; never returned to the browser or logged

## Authorization

- User identity comes from the session cookie, never a client `user_id`
- Repo, rules, events, and actions are scoped to that user’s connected repository
- One connected repo per user (unique constraint)
- GitHub writes use server-side owner/name for that connection, not client-supplied identity

## CSRF

- Mutating `/api/*` and `POST /auth/logout` require `Origin` matching `FRONTEND_URL`
- `POST /webhooks/github` is not Origin-gated—HMAC proves authenticity
- OAuth start/callback rely on OAuth `state`

## Webhooks

- `X-Hub-Signature-256` over the raw body; `hmac.Equal`
- Body size capped (`WEBHOOK_MAX_BODY_BYTES`)
- `X-GitHub-Delivery` unique in Postgres
- Unknown / unconnected repos ignored; webhook secret never logged

## Idempotency

- Events: unique delivery ID
- Actions: unique `(event_id, rule_id, action_type)` before any external HTTP

## Secrets

Keep out of frontend, bundles, API responses, and logs:

- OAuth client secret, access tokens, webhook secret
- `SESSION_SECRET`, DB passwords, `SLACK_WEBHOOK_URL`
- AI provider keys

Startup logs use `Config.Redacted()`.

## Browser headers & rate limits

- Conservative headers on API responses (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, restrictive API CSP)
- Light in-memory limits on OAuth and `/api` (per process); webhooks are not rate-limited in-app so legitimate deliveries are not dropped

## Optional AI

Off by default. Model output is schema-validated; it cannot authorize or call arbitrary URLs. Provider failure skips enrichment only.

## Known limits

- Rate limits are per instance
- Exactly-once external delivery is not guaranteed (ADR-008)
- OAuth App user tokens, not GitHub App installation tokens
- Repo webhook is configured manually

## CORS

Reflects only `FRONTEND_URL`, with credentials—never `*`.

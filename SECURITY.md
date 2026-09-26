# Security

What the live app actually enforces. Not a formal audit.

## Quality bar (assignment)

| Requirement | How |
| --- | --- |
| Not fooled by forged webhooks | `X-Hub-Signature-256` on raw body, `hmac.Equal` → 401 if bad |
| Not double-acting on redelivery | Unique delivery ID; action key `event_id:rule_id:action_type` |
| Not silently losing events | Persist event before ack; retries; failures visible on dashboard |
| Never expose secrets | Not in repo, frontend bundle, API responses, or logs |

## Auth & sessions

- GitHub OAuth; single-use TTL `state`
- HttpOnly cookie `gaf_session` (hash in DB)
- Access tokens AES-GCM encrypted at rest
- Production: Vercel proxies `/api` + `/auth` so the cookie is first-party

## Authorization

Identity from session only. Repo / rules / events / actions scoped to the connected repo. Connect requires GitHub `admin`. One repo per user.

## Other

- CSRF Origin check on mutating cookie APIs (not on webhooks — those use HMAC)
- CORS reflects only `FRONTEND_URL`
- Optional AI off by default; output schema-validated; fail-open

If you find a leaked secret, rotate it and tell me — don’t open a public issue with the value.

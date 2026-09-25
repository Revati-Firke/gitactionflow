# Changelog

Notable changes to GitActionFlow. Format inspired by [Keep a Changelog](https://keepachangelog.com/).

## [1.0.0] — Release Candidate — 2026-09-26

### Added

- GitHub OAuth login (CSRF `state`, HttpOnly sessions, encrypted access tokens)
- Connect / disconnect one owned GitHub repository
- Signed webhooks (issues + pull requests) with delivery-ID idempotency
- Durable event worker (claim, retries, stale lease recovery, failure states)
- Configurable rules and action intents
- GitHub label/comment actions and Slack Incoming Webhook notifications
- Authenticated React dashboard (repo, rules, events, actions)
- Optional AI (`AI_ENABLED`) with schema-validated suggestions (fail-open)
- Production hardening: CSRF Origin checks, security headers, request IDs, rate limits, timeouts
- Free-tier deploy: Neon, Render (Go Docker), Vercel (SPA + `/api` `/auth` proxy)
- Assignment docs under `docs/assignment/`

### Reliability

- Webhook HMAC-SHA256 (constant-time compare)
- Persist events and actions before external side effects
- Action idempotency key `event_id:rule_id:action_type`
- Retry/backoff for transient failures; permanent errors fail visibly
- Dashboard shows `last_error` and action statuses

### Deployment

- Neon PostgreSQL, Render backend, Vercel frontend with same-origin proxy for cookies

### Intentionally omitted

- Multi-repository product mode
- GitHub App installation authentication
- Automatic webhook registration on connect

## Earlier work

Built incrementally on `dev` (foundation → OAuth → repo → webhooks → worker → rules → actions → dashboard → deploy). See `git log` for conventional commits.

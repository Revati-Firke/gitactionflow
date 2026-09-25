# AGENTS.md — GitActionFlow

Notes I leave for anyone touching this repo (including future me). This is not a feature wishlist—it records how I chose to build the Abstrabit SDE1 take-home.

---

## What this project is

A small event-driven bot: GitHub OAuth → connect one repo → signed webhooks → rules → GitHub label/comment + Slack → history on a login-gated dashboard.

I optimized for **working end-to-end on free tiers**, not for looking like a platform. If a change makes the demo less reliable, I skip it.

---

## Decisions I locked in

| Choice | Why I made it |
| --- | --- |
| Modular monolith (one Go process + React + Postgres) | One thing to deploy and debug on Render; microservices were overkill for the brief |
| Neon + Render + Vercel | Free tiers only, no credit card; enough for a public reviewer URL |
| GitHub **OAuth App** (not GitHub App) | Enough for “sign in + act as the user on one repo”; App install flow would burn time without winning the core ask |
| **One connected repo per user** | Matches the assignment; multi-repo is product work I deliberately deferred |
| Postgres as queue (`FOR UPDATE SKIP LOCKED`) | Durable without Redis/Kafka; survives Render cold starts |
| Rules emit **intents**; executor does side effects | So retries don’t double-fire Slack/GitHub; easier to unit-test matching |
| Persist action row **before** external HTTP | Crash mid-call shouldn’t lose the intent; unique `event_id:rule_id:action_type` |
| Manual webhook on the repo | Auto register/delete is polish; signature verify + idempotent ingest mattered more for the deadline |
| Vercel **same-origin proxy** for `/api` and `/auth` | Incognito blocked third-party cookies to Render; first-party cookies on Vercel fixed login |
| Optional AI behind `AI_ENABLED` (default off) | Stretch only—label/comment/Slack must work with AI offline |

Longer write-ups: `docs/decisions/`. How I used Cursor vs what I decided myself: `AI_NOTES.md`.

---

## Architecture (keep this shape)

```text
GitHub webhook → verify HMAC → dedupe delivery ID → persist event
  → worker → rule engine → actions (GitHub / Slack) → persist results
  → dashboard reads PostgreSQL
```

Details live in `docs/architecture/HLA.md`. Don’t silently redesign—open an ADR if something genuinely needs to change.

| Layer | Stack I used |
| --- | --- |
| Backend | Go, Gin, pgx, golang-migrate (`cmd/server`, code under `internal/`) |
| Frontend | React, TypeScript, Vite |
| Integrations | GitHub OAuth / Webhooks / REST, Slack Incoming Webhook |
| Optional AI | Gemini or Groq behind a small interface |

I preferred **pgx + SQL** over an ORM so the schema and idempotency constraints stay obvious in migrations.

---

## Hard rules (security / reliability)

These are non-negotiable for how I built the core path:

- OAuth `state` is single-use + TTL; sessions are HttpOnly cookies; tokens encrypted at rest; never in localStorage/URL/logs
- Identity always comes from the session—never from a client `user_id`
- Re-fetch repo metadata from GitHub by id; require `admin`; one repo per user
- Webhooks: `X-Hub-Signature-256` on the **raw** body (`hmac.Equal`); `X-GitHub-Delivery` unique; persist before ack; no GitHub/Slack calls inside the webhook request
- Rule conditions are **AND**; disabled rules never match; scope to the connected repo only
- Slack URL only from `SLACK_WEBHOOK_URL`; never from rule config or the browser
- Retry transient 429/5xx/network; fail permanently on 401/403/404/422/config mistakes
- Don’t mark an event `processed` while required actions are still pending/failed
- Exactly-once external delivery is **not** claimed (see ADR-008)—minimize duplicates, show failures on the dashboard

Frontend: API under `frontend/src/services/api/`, `credentials: 'include'`, no provider tokens in the browser, refresh buttons instead of websockets.

Deploy: prefer platform `PORT`; `FRONTEND_URL` exact origin; CORS only that origin; leave `VITE_API_BASE_URL` unset in production so the Vercel proxy stays first-party.

---

## Scope I refuse by default

GitLab/Bitbucket, multi-repo product features, Redis/Kafka/K8s, visual workflow builders, paid APIs. If something tempting shows up, write it under “future work” or an ADR—don’t ship it into the submission unless it clearly improves the demo.

---

## How I prefer to change the code

Read the existing package before rewriting it. Small diffs. Fail fast on missing config. Structured `slog` JSON with `Config.Redacted()`—never log secrets. Thin handlers; business logic in `internal/`. Tests for signature, idempotency, and rule matching matter more than a giant E2E suite.

Commits: Conventional Commits (`feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `security`). No `update` / `final` messages.

Docs should match what actually runs. If I cut a feature on purpose, say so—don’t paper over it.

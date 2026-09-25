# Assignment verification

Evidence log for submission. Separates what is coded, unit-tested, and checked in production.

**Do not put secrets in this file.**

## Environment

| Layer | Detail |
| --- | --- |
| Frontend | https://gitactionflow.vercel.app |
| Backend | https://gitactionflow-backend.onrender.com |
| Database | Neon Postgres |
| OAuth | GitHub OAuth App → callback on Vercel `/auth/github/callback` (proxied) |
| Webhook | `https://gitactionflow-backend.onrender.com/webhooks/github` |
| Slack | Render `SLACK_WEBHOOK_URL` → `#gitactionflow-demo` |

Local setup: `docs/setup/LOCAL.md`.

---

## Core

| Requirement | Status | How verified | Notes |
| --- | --- | --- | --- |
| Public app | PRODUCTION | Open Vercel URL | Live SPA |
| OAuth login | PRODUCTION | Authorize → dashboard | Fixed via same-origin proxy |
| Connect repository | PRODUCTION | Dashboard | Owner confirmed |
| Health/ready | PRODUCTION | curl | ok / ready |
| Webhook ingest | PRODUCTION + tests | Live deliveries + unit tests | HMAC + delivery ID |
| Issue automation | PRODUCTION | Issue → event/Slack | Observed |
| PR automation | CODE READY | Same pipeline as issues | Open one live PR before demo |
| Slack | PRODUCTION | Channel message | Verified |
| Dashboard history | PRODUCTION | UI after login | Events/actions |
| Rules CRUD | PRODUCTION | UI | — |

## Quality / security

| Item | Status | Notes |
| --- | --- | --- |
| Signature verify | LOCAL + LIVE | `go test`; forged live → 401 |
| Duplicate delivery | LOCAL | Unique constraint + handler tests; live Redeliver once preferred |
| Action idempotency | LOCAL | Executor / store tests |
| CSRF Origin | LOCAL | Middleware tests |
| Authz scoping | CODE | Session → connected repo only |
| Race detector | LOCAL | `go test -race ./...` PASS |

## Optional

| Item | Status |
| --- | --- |
| AI assistance | Implemented, disabled by default |
| Multi-repo | Not implemented |
| GitHub App | Not implemented |

## Production smoke checklist

```text
[x] Login E2E
[x] Repo connected
[x] Webhook delivery (issues appear)
[x] Matching issue → Slack completed
[ ] Matching issue → GitHub label (existing label only)
[ ] Non-match → no action
[ ] PR event
[ ] Redeliver → no duplicate action
[x] Slack message in channel
[ ] Logout
```

## Duplicate delivery (live)

```text
Date: 2026-09-26
Result: NOT YET RUN live — unit coverage PASS; confirm via GitHub Redeliver once
```

## Automated tests (last RC run)

```text
cd backend && go test ./...       → PASS
cd backend && go test -race ./... → PASS
cd backend && go vet ./...        → PASS
cd frontend && npm run build      → PASS
Forged webhook (live)             → 401 invalid signature
Frontend live bundle              → no hardcoded Render API base
```

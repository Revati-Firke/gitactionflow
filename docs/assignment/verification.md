# Assignment verification

Evidence log for GitActionFlow submission. Distinguishes what was coded, tested locally, and verified in production.

**Do not put secrets in this file.**

## Environment

| Layer | Detail |
| --- | --- |
| Frontend | https://gitactionflow.vercel.app |
| Backend | https://gitactionflow-backend.onrender.com |
| Database | Neon Postgres (`gitactionflow` project) |
| OAuth | GitHub OAuth App → callback on Vercel `/auth/github/callback` (proxied to Render) |
| Webhook | `https://gitactionflow-backend.onrender.com/webhooks/github` |
| Slack | Optional via Render `SLACK_WEBHOOK_URL` |

Local: Docker Compose Postgres + `go run` + Vite (see `docs/setup/LOCAL.md`).

---

## Core requirements

| Requirement | Status | How verified | Evidence |
| --- | --- | --- | --- |
| Public app | PRODUCTION TESTED | Open Vercel URL | Live SPA |
| OAuth login | PRODUCTION TESTED | Authorize → dashboard | Session after proxy fix |
| Connect repository | PRODUCTION TESTED | Dashboard connect | Owner confirmed |
| Health/ready | PRODUCTION TESTED | curl | `{"status":"ok"}` / `ready` |
| Webhook ingest | IMPLEMENTED | Unit tests; live when webhook configured | See smoke checklist |
| Issue automation | IMPLEMENTED | Code + local tests; live smoke per owner | Label/comment rules |
| PR automation | IMPLEMENTED | Same pipeline as issues | Event type `pull_request` |
| Slack | IMPLEMENTED | Unit mocks; live needs env | — |
| Dashboard history | PRODUCTION TESTED | UI after login | Events/actions panels |
| Rules CRUD | PRODUCTION TESTED | UI | — |

## Quality / security

| Item | Status | Notes |
| --- | --- | --- |
| Signature verify | TESTED LOCALLY | `go test` webhook packages |
| Duplicate delivery | TESTED LOCALLY | Unique constraint + handler tests |
| Action idempotency | TESTED LOCALLY | executor / store tests |
| CSRF Origin | TESTED LOCALLY | middleware tests |
| Authz scoping | IMPLEMENTED | Session → connected repo only |
| Race detector | See Phase 10 test run | `go test -race ./...` |

## Optional

| Item | Status |
| --- | --- |
| AI assistance | IMPLEMENTED — disabled by default; enable with `AI_ENABLED` + provider key |
| Multi-repo | NOT IMPLEMENTED |
| GitHub App | NOT IMPLEMENTED |

## Production smoke (owner checklist)

```text
[x] Login E2E
[x] Repo connected
[x] Webhook delivery (issues appear)
[x] Matching issue → Slack action completed
[ ] Matching issue → GitHub label (use existing label; avoid 422)
[ ] Non-match → no action
[ ] PR event
[ ] Redeliver → no duplicate action
[x] Slack message in channel
[ ] Logout
```

## Duplicate delivery production test

```text
Date: 2026-09-26
Result: NOT YET RUN (owner) — unit coverage PASS; confirm via GitHub Redeliver once
```

## Phase 11 automated tests

```text
cd backend && go test ./...       → PASS
cd backend && go test -race ./... → PASS
cd backend && go vet ./...        → PASS
cd frontend && npm run build      → PASS
Forged webhook (live)             → 401 invalid signature
Frontend live bundle              → no localhost / no onrender hardcoded API base
```

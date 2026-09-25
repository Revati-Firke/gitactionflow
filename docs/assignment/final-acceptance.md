# Final acceptance — Release Candidate

Date: 2026-09-26

```text
READY FOR SUBMISSION — with demo hygiene notes
```

Core Abstrabit requirements are implemented and verified on the live stack (OAuth, one repo, issue webhooks, Slack, rules, dashboard). Two demo-hygiene items remain on me before a reviewer session—not missing product features:

1. Demo rules should use **labels that already exist** on the repo (or Slack/comment only), so a GitHub 422 does not mark the event Failed while Slack still succeeds.
2. Optionally run one live **PR** webhook and one webhook **Redeliver** once before the demo.

---

## Core requirements

| # | Requirement | Result | Evidence |
| --- | --- | --- | --- |
| 1 | Public deployed app | **PASS** | Frontend/backend HTTPS; `/health` `/ready` ok |
| 2 | GitHub OAuth | **PASS** | Live login → dashboard (same-origin proxy) |
| 3 | Connect one repository | **PASS** | `Daily-Impression-AI-Model` connected |
| 4 | Webhook issues | **PASS** | Events + Slack/actions for issues |
| 5 | Webhook pull requests | **NOT TESTED live** | Code supports `pull_request`; open one PR once |
| 6 | Bot GitHub label | **PARTIAL** | Works when the label exists; 422 if missing |
| 7 | Bot GitHub comment | **NOT TESTED live** | Implemented + unit tests |
| 8 | Slack notification | **PASS** | `#gitactionflow-demo` |
| 9 | Configurable rules | **PASS** | Dashboard CRUD |
| 10 | Event/action history | **PASS** | Lists + failure visibility |
| 11 | Signature verification | **PASS** | Unit tests; forged live POST rejected |
| 12 | Idempotency | **PASS** (code/tests) | Delivery UNIQUE + action key; live redeliver once preferred |
| 13 | Retries / failure persistence | **PASS** | `last_error` visible; unit tests |
| 14 | README / .env.example / deploy docs | **PASS** | Live URLs in README |
| 15 | AGENTS.md / AI_NOTES / SECURITY | **PASS** | Present and current |

## Optional

| Feature | Status |
| --- | --- |
| AI assistance | Optional, `AI_ENABLED=false` by default — not required live |
| Observability | JSON logs + `request_id` |
| Multi-repository | Not built (by design) |
| GitHub App auth | Not built (OAuth App kept) |

## Automated checks

```text
cd backend && go test ./...          → PASS
cd backend && go vet ./...           → PASS
cd backend && go test -race ./...    → PASS
cd frontend && npm run build         → PASS
Frontend bundle secret scan          → no client/session/Slack/DB secret patterns
```

## Demo pitfall

Several rules can match keyword `bug`. If a label rule names a label that does not exist, GitHub returns **422**, Slack may still **Complete**, and the **event** shows **Failed**. For demos: disable that rule or create the label first.

## Design stance (for reviewers)

I chose a modular monolith, free Neon/Render/Vercel hosting, a Vercel proxy for cookies, and a GitHub OAuth App with one repo—documented in `AI_NOTES.md` and `docs/decisions/`. Stretch AI is behind a flag and does not own the critical path.

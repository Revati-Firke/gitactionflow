# Submission checklist

## Repository

- [x] Public GitHub repository (`Revati-Firke/gitactionflow`)
- [x] Clean README with live URLs
- [x] Modular monolith structure (`backend/`, `frontend/`, `docs/`)
- [x] No secrets in git (`.env` gitignored; `.env.example` placeholders)
- [ ] Useful commit history includes Phase 10 (commit/push pending if still local)
- [x] `AGENTS.md`
- [x] `AI_NOTES.md`
- [x] `SECURITY.md`
- [x] `.env.example`

## Application

- [x] Public frontend — https://gitactionflow.vercel.app
- [x] Public backend — https://gitactionflow-backend.onrender.com
- [x] GitHub OAuth
- [x] Repository connection (one repo)
- [x] Webhooks (manual registration on repo)
- [x] Issues path verified live
- [ ] Pull requests — run one live PR smoke before submit if not done
- [x] Configurable rules
- [ ] GitHub label — use existing label for clean demo (422 if missing)
- [ ] GitHub comment — optional live confirm
- [x] Slack verified live
- [x] Event history
- [x] Action history

## Reliability

- [x] Signature verification (tests + forged request rejected)
- [x] Idempotency (DB constraints + tests; confirm live redeliver once)
- [x] Retries (implemented + tested)
- [x] Failure persistence / dashboard visibility
- [x] Worker recovery (lease / SKIP LOCKED)
- [x] Authorization isolation (session-scoped)

## Optional

- [x] AI — implemented, disabled by default
- [x] Observability — structured logs + request_id (Phase 10)
- [ ] Multi-repository — **not implemented**
- [ ] GitHub App — **not implemented**

## Docs for reviewers

- [x] `docs/assignment/requirements-matrix.md`
- [x] `docs/assignment/verification.md`
- [x] `docs/assignment/final-acceptance.md`
- [x] `docs/assignment/demo-script.md`
- [x] `docs/deployment/`

## Pre-submit actions

1. Commit + push Phase 10/11 docs and hardening; redeploy Render + Vercel.  
2. Fix demo rules (existing labels only).  
3. One PR smoke + one webhook redeliver.  
4. Fill any remaining checkboxes above with real results.

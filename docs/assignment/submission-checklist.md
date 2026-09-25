# Submission checklist — Release Candidate

Date: 2026-09-26

## Repository

- [x] Public GitHub repository
- [x] README with live demo URLs
- [x] Clear project structure
- [x] No secrets committed (`.env` / `*.local.md` gitignored)
- [x] Conventional commit history on `dev`
- [x] `AGENTS.md`
- [x] `AI_NOTES.md` (tools used + my decisions + corrections)
- [x] `SECURITY.md`
- [x] `.env.example`
- [x] `CHANGELOG.md` RC entry

## Application

- [x] Frontend — https://gitactionflow.vercel.app
- [x] Backend — https://gitactionflow-backend.onrender.com
- [x] GitHub OAuth
- [x] Repository connection
- [x] Webhooks (manual on repo)
- [x] Issues path verified live
- [ ] Pull request — code ready; one live PR before demo if not done
- [x] Configurable rules
- [x] GitHub label (use an **existing** label for a clean demo)
- [x] GitHub comment (implemented; confirm live if demoing)
- [x] Slack verified live
- [x] Event history
- [x] Action history

## Reliability

- [x] Signature verification (tests + live forged → 401)
- [x] Idempotency (DB + tests; live redeliver once if possible)
- [x] Retries
- [x] Failure visibility on the dashboard
- [x] Worker recovery
- [x] Authorization isolation

## Optional

- [x] AI — implemented, off by default
- [x] Observability — structured logs + request_id
- [ ] Multi-repository — not built
- [ ] GitHub App — not built

## Docs

- [x] Deployment docs
- [x] Requirements matrix
- [x] Verification
- [x] Demo script
- [x] Final acceptance
- [x] This checklist

## Pre-demo

Use labels that exist on the demo repo (or Slack-only rules) so the event status shows **Processed**, not Failed from a GitHub 422.

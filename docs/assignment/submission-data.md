# Submission data — Abstrabit SDE1 (GitActionFlow)

Copy-paste fields for the submission form / email. **No secrets.**  
Date prepared: 2026-09-26

> **Before you submit:** commit and push the latest doc polish (`AGENTS.md`, `AI_NOTES.md`, README, assignment docs) to `origin/dev`, then redeploy Render/Vercel if needed so the public repo matches what you claim.

Live check (just now): frontend **200**, `/health` **ok**, `/ready` **ready**.

---

## Form fields (copy as-is)

### Candidate

```text
Name:     Revati Firke
GitHub:   https://github.com/Revati-Firke
Email:    revatifirke02@gmail.com
```

### Project

```text
Project name:     GitActionFlow
Repository:       https://github.com/Revati-Firke/gitactionflow
Default branch:   dev
```

### Live demo URLs

```text
Frontend:         https://gitactionflow.vercel.app
Backend:          https://gitactionflow-backend.onrender.com
Health:           https://gitactionflow-backend.onrender.com/health
Ready:            https://gitactionflow-backend.onrender.com/ready
Login:            https://gitactionflow.vercel.app/login
Dashboard:        https://gitactionflow.vercel.app/dashboard
Webhook (GitHub): https://gitactionflow-backend.onrender.com/webhooks/github
```

### Stack (one line)

```text
Go (Gin/pgx) + React/Vite + PostgreSQL (Neon) · Render + Vercel · GitHub OAuth App · Slack Incoming Webhook
```

### Docs reviewers should open

```text
README.md
AI_NOTES.md
AGENTS.md
SECURITY.md
docs/assignment/requirements-matrix.md
docs/assignment/verification.md
docs/assignment/demo-script.md
docs/assignment/final-acceptance.md
docs/architecture/HLA.md
docs/decisions/
```

---

## Short summary (paste into “About your submission”)

```text
GitActionFlow is a deployed event-driven GitHub automation bot: OAuth login, one connected repository, signed webhooks (issues + PRs), configurable AND rules, GitHub label/comment + Slack actions, and an authenticated dashboard for events/actions.

Stack: Go modular monolith + React, Postgres as durable queue (Neon), free-tier Render + Vercel. Session cookies use a Vercel same-origin proxy for /api and /auth so login works when third-party cookies are blocked.

Security: HMAC webhook verification, delivery-ID idempotency, HttpOnly sessions, encrypted GitHub tokens, CSRF Origin on cookie APIs. Optional AI is behind AI_ENABLED (default off) and does not own the core path.

How I used AI: Cursor for coding assistance; ChatGPT for research. Decisions and the hardest AI wrong turn (cross-site cookies vs same-origin proxy) are in AI_NOTES.md.
```

---

## Demo credentials / access notes

```text
No shared demo user password — reviewers use “Login with GitHub” on their own account.
Connect a repo they admin, or ask me to leave Daily-Impression-AI-Model connected for a guided demo.
Demo script: docs/assignment/demo-script.md
Slack (my workspace): #gitactionflow-demo — reviewers will not see my Slack; they can set SLACK_WEBHOOK_URL on their own deploy, or watch GitHub actions + dashboard history.
```

---

## What works live (honest)

| Item | Status |
| --- | --- |
| Public app + health/ready | Verified |
| GitHub OAuth → dashboard | Verified |
| Connect one repo | Verified (`Daily-Impression-AI-Model`) |
| Issue webhook → rules → Slack | Verified |
| Signature reject (forged) | Verified (401) |
| GitHub label | Works if label exists on repo |
| GitHub comment | Implemented; confirm live if demoing |
| Pull request webhook | Code ready; one live PR smoke recommended |
| Webhook Redeliver | Unit-tested; one live redeliver recommended |
| Optional AI | Implemented, default off |

---

## One-message email / Slack (optional)

```text
Subject: Abstrabit SDE1 submission — GitActionFlow (Revati Firke)

Hi,

Submitting my SDE1 take-home: GitActionFlow.

Repo:     https://github.com/Revati-Firke/gitactionflow  (branch: dev)
Frontend: https://gitactionflow.vercel.app
Backend:  https://gitactionflow-backend.onrender.com/health

Please start at the README, then AI_NOTES.md for how I used Cursor + ChatGPT and the cookie/proxy fix. Demo walkthrough: docs/assignment/demo-script.md.

Happy to walk through a live issue → Slack/dashboard flow if useful.

Thanks,
Revati Firke
```

---

## Pre-submit checklist (you)

1. [ ] `git status` clean — doc polish committed & pushed to `origin/dev`
2. [ ] Repo is **public**
3. [ ] Wake backend once (`/health`) before the reviewer opens the app (Render free tier)
4. [ ] Demo rules use **existing** labels (or Slack-only) — avoid GitHub 422 → event Failed
5. [ ] Optional: one PR + one webhook Redeliver once
6. [ ] No secrets in git (`.env`, `DEPLOYMENT_URLS.local.md` stay local)

# GitActionFlow

Abstrabit SDE1 take-home — **event-driven GitHub automation bot**.

Sign in with GitHub → connect **one** repo → receive signed webhooks → match simple rules → label/comment on GitHub and notify Slack → see it all on a login-gated dashboard.

Built and deployed on **free tiers only** (no credit card): Neon + Render + Vercel.

---

## Live URLs

```text
App:     https://gitactionflow.vercel.app
API:     https://gitactionflow-backend.onrender.com
Health:  https://gitactionflow-backend.onrender.com/health
Source:  https://github.com/Revati-Firke/gitactionflow   (branch: dev)
```

**Try it:** open the app → Login with GitHub (your account) → connect a repo you admin → add a rule → open an issue.  
Webhook on *your* repo is manual (URL below). Or watch the demo video in the submission for the full live path on my wired demo repo.

```text
Webhook URL:  https://gitactionflow-backend.onrender.com/webhooks/github
Events:       Issues + Pull requests
Secret:       ask me privately if you need to test on your own repo
```

---

## Assignment coverage

| Brief requirement | Status |
| --- | --- |
| Public deployed app | Live (links above) |
| GitHub sign-in + connect one owned repo | Done (OAuth App, admin required) |
| Webhooks for ≥2 event types, recorded | `issues` + `pull_request` |
| Bot writes back to GitHub (label or comment) | Both implemented |
| Slack notification | Incoming Webhook |
| Dashboard behind login (events + actions + rules) | React dashboard |
| README + `.env.example` + deploy notes | This file + docs below |
| Quality: forged webhooks, idempotency, no silent loss, no leaked secrets | HMAC + delivery/action keys + persist-before-ack + redacted logs |
| Stretch: configurable rules | UI + AND conditions |
| Stretch: optional free AI | `AI_ENABLED` default **off** — core path works without it |
| Stretch: GitHub App / multi-repo | **Skipped on purpose** (OAuth App + one repo) |

How I used AI tools while building: **[AI_NOTES.md](AI_NOTES.md)**.  
Working constraints I set for myself: **[AGENTS.md](AGENTS.md)**.

---

## What I chose (and why)

| Decision | Why |
| --- | --- |
| One Go process + React + Postgres | Small product; easy to deploy/debug on free hosts |
| Postgres as the queue (`FOR UPDATE SKIP LOCKED`) | Durable without Redis/Kafka; survives Render sleep |
| GitHub **OAuth App**, one repo per user | Matches the brief; GitHub App / multi-repo were extra scope |
| Rules emit intents; separate executor | Retries don’t blindly double-fire Slack/GitHub |
| Persist event/action **before** external HTTP | Don’t silently lose work if Slack/GitHub blips |
| Vercel proxy for `/api` + `/auth` | Incognito blocked third-party cookies to Render |

More detail: [docs/decisions/](docs/decisions/) · [docs/architecture/HLA.md](docs/architecture/HLA.md)

```text
webhook → verify HMAC → dedupe delivery ID → persist
  → worker → rules → GitHub / Slack → dashboard
```

---

## Stack

Go (Gin, pgx) · React + Vite · PostgreSQL · GitHub OAuth/Webhooks/REST · Slack Incoming Webhook · optional Gemini/Groq

---

## How to test (reviewers)

1. Open https://gitactionflow.vercel.app (wake API via `/health` if cold).
2. Login with GitHub → dashboard.
3. Confirm a connected repo (or connect one you admin).
4. Create/enable a rule: keyword `bug` → Slack and/or GitHub comment (use an **existing** label name if you choose label).
5. Open an issue titled e.g. `demo bug` on the connected repo (webhook must point at the URL above).
6. Refresh **Events** and **Actions** on the dashboard.
7. Check GitHub for comment/label; Slack only if you share my workspace or set your own `SLACK_WEBHOOK_URL` on a fork/deploy.

Forged webhook (expect non-2xx):

```bash
curl -sS -o /dev/null -w "%{http_code}\n" -X POST \
  https://gitactionflow-backend.onrender.com/webhooks/github \
  -H 'Content-Type: application/json' \
  -H 'X-GitHub-Event: issues' \
  -H 'X-GitHub-Delivery: review-forge-1' \
  -H 'X-Hub-Signature-256: sha256=deadbeef' \
  -d '{"action":"opened"}'
```

---

## Local run

Details: [docs/setup/LOCAL.md](docs/setup/LOCAL.md). Summary:

```bash
cp .env.example .env   # fill OAuth + SESSION_SECRET + webhook secret
docker-compose up -d postgres
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

Open `http://localhost:5173` (use **localhost**, not `127.0.0.1`).

Required env (see `.env.example`): `DATABASE_URL`, `FRONTEND_URL`, `GITHUB_CLIENT_*`, `GITHUB_OAUTH_REDIRECT_URL`, `SESSION_SECRET`, `GITHUB_WEBHOOK_SECRET`. Optional: `SLACK_WEBHOOK_URL`, `AI_ENABLED`.

```bash
cd backend && go test ./...
cd frontend && npm run build
```

---

## Deploy (how I shipped it)

Full steps: [docs/deployment/README.md](docs/deployment/README.md).

1. Neon → `DATABASE_URL`
2. Render Docker from `backend/` (`AUTO_MIGRATE=true`)
3. Vercel from `frontend/` — leave `VITE_API_BASE_URL` **unset** (same-origin proxy in `vercel.json`)
4. OAuth App homepage + callback on the **Vercel** host
5. Repo webhook → Render `/webhooks/github`

---

## Security (quality bar)

- Webhook HMAC on raw body; unique `X-GitHub-Delivery`
- Action idempotency key before GitHub/Slack HTTP
- HttpOnly sessions; GitHub tokens encrypted at rest; never in the browser
- No secrets in git, frontend, or logs (`.env.example` placeholders only)

See [SECURITY.md](SECURITY.md).

---

## License

See [LICENSE](LICENSE).

# Deployment (Neon + Render + Vercel)

How this app is hosted — free tiers, no credit card.

```text
App:      https://gitactionflow.vercel.app
API:      https://gitactionflow-backend.onrender.com
Callback: https://gitactionflow.vercel.app/auth/github/callback   (Vercel → Render proxy)
Webhook:  https://gitactionflow-backend.onrender.com/webhooks/github
```

Browser uses same-origin `/api` and `/auth` on Vercel. GitHub webhooks hit Render directly.

---

## Steps I used

1. **Neon** — create Postgres; set `DATABASE_URL` (SSL).
2. **Render** — Docker from `backend/` (`APP_ENV=production`, `AUTO_MIGRATE=true`, health `/health`).
3. Confirm `/health` and `/ready`.
4. **Vercel** — root `frontend/`; leave **`VITE_API_BASE_URL` unset** so `vercel.json` proxies `/api` + `/auth`, then SPA fallback.
5. Set Render `FRONTEND_URL` and `GITHUB_OAUTH_REDIRECT_URL` to the Vercel origin/callback.
6. GitHub OAuth App: homepage = Vercel; callback = Vercel `/auth/github/callback`.
7. On the demo repo: webhook → Render URL, JSON, Issues + PRs, same secret as `GITHUB_WEBHOOK_SECRET`.
8. Optional: `SLACK_WEBHOOK_URL` on Render only.

### Why the Vercel proxy

Cross-site cookies to Render failed in Incognito. Proxying `/api` and `/auth` makes the session cookie first-party on the app host.

### Render env (required)

`DATABASE_URL`, `FRONTEND_URL`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `GITHUB_OAUTH_REDIRECT_URL`, `GITHUB_WEBHOOK_SECRET`, `SESSION_SECRET`, `AUTO_MIGRATE=true`.  
Optional: `SLACK_WEBHOOK_URL`, `AI_ENABLED=false`.

Never put secrets in Vercel.

Local still uses Docker Compose + `.env` — see [LOCAL.md](../setup/LOCAL.md).

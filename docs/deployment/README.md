# Deployment Direction

**Status:** Planned — **nothing is deployed** in Phase 1.

GitHub OAuth callbacks and webhooks require a **public HTTPS URL**. Localhost alone is insufficient for the live assignment demo.

---

## Intended topology

```text
React frontend
      ↓
Public deployment
      ↓
Go backend
      ↓
PostgreSQL
```

| Piece | Role |
| --- | --- |
| React (Vite build) | Static assets + SPA |
| Go backend | API, OAuth, webhooks, processing |
| PostgreSQL | Durable state |

---

## Expected free-tier services

Assignment constraint: **no credit card**. Prefer providers with genuine free tiers. Always re-check current free-tier terms before locking in a choice — limits and eligibility change.

| Concern | Candidate options (verify before use) |
| --- | --- |
| Frontend | Vercel, Netlify, or Render static hosting |
| Backend | Render (or similar free web service suitable for a long-running Go process) |
| PostgreSQL | Neon or Supabase free Postgres |
| Secrets | Host-provided environment variables |
| Slack | Free Slack workspace + Incoming Webhook |
| GitHub | Free OAuth App / webhooks / API |

If a service asks for a credit card for the tier you need, switch providers or tiers.

---

## Configuration on the host

Environment variables will mirror `.env.example` (public URL, OAuth client id/secret, webhook secret, database URL, Slack webhook URL, etc.).

Critical public URLs to configure consistently:

- Frontend origin
- Backend public base URL
- GitHub OAuth callback URL
- GitHub webhook payload URL

---

## Local vs production

| Environment | Notes |
| --- | --- |
| Local | `docker compose` for Postgres; app processes on localhost; tunnel required to receive real GitHub webhooks |
| Production | Public HTTPS frontend + backend; managed Postgres; secrets only in host env |

---

## What Phase 1 does not do

- Create cloud accounts
- Provision databases remotely
- Deploy frontend or backend
- Claim a live URL

Deployment steps and the live URL will be added to the root `README.md` when Phase 8 (or equivalent) completes.

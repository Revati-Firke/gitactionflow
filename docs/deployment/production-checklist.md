# Production configuration checklist

Use when (re)deploying. Items below reflect the current live demo unless noted.

## Infrastructure

- [x] Neon database created
- [x] Render backend deployed (Docker, free tier)
- [x] Vercel frontend deployed
- [x] HTTPS active on frontend and backend

## Backend

- [x] `DATABASE_URL` configured (Neon, SSL)
- [x] `PORT` handled by platform
- [x] `GET /health` works
- [x] `GET /ready` works
- [x] Migrations applied (`AUTO_MIGRATE=true`)

## Authentication

- [x] `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` set on Render
- [x] `GITHUB_OAUTH_REDIRECT_URL` = Vercel `/auth/github/callback` (proxied)
- [x] GitHub OAuth App homepage/callback match production Vercel URLs
- [x] `SESSION_SECRET` ≥ 32 chars
- [x] Production cookie defaults / Secure as needed
- [x] `FRONTEND_URL` exact Vercel origin
- [x] Login and logout work from Vercel

## Frontend

- [x] `VITE_API_BASE_URL` **unset** (same-origin proxy)
- [x] Production build succeeds
- [x] SPA routes (`/login`, `/dashboard`) work on refresh
- [x] Authenticated API calls send cookies (`credentials: 'include'`)
- [x] Vercel Deployment Protection / Require Log In OFF for Production

## GitHub

- [x] One repository connected in the app
- [x] Webhook URL → Render `/webhooks/github`
- [x] Content type `application/json`
- [x] Issues + Pull requests subscribed
- [x] Webhook secret matches `GITHUB_WEBHOOK_SECRET`
- [x] Issue delivery shows HTTP 200

## Slack

- [x] `SLACK_WEBHOOK_URL` set on Render only
- [x] Notification tested end-to-end

## Security

- [x] No secrets committed / no real values in `.env.example`
- [x] No secrets in Vercel / frontend bundle
- [x] CORS allows only `FRONTEND_URL` (not `*`)
- [x] Logs do not print secrets (redacted config)

## End-to-end

- [x] Login
- [x] Connect repository
- [x] Create rule
- [x] Trigger issue
- [ ] Trigger pull request (confirm once before demo)
- [ ] GitHub label (use existing label) and/or comment
- [x] Slack notification
- [x] Event history
- [x] Action history
- [ ] Duplicate delivery (Redeliver once)

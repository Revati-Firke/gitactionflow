# Production configuration checklist

## Infrastructure

- [ ] Neon database created
- [ ] Render backend deployed (Docker, free tier)
- [ ] Vercel frontend deployed
- [ ] HTTPS active on frontend and backend

## Backend

- [ ] `DATABASE_URL` configured (Neon, SSL)
- [ ] `PORT` handled by platform (app prefers `PORT` over `APP_PORT`)
- [ ] `GET /health` works
- [ ] `GET /ready` works
- [ ] Migrations applied (`AUTO_MIGRATE=true` or equivalent)

## Authentication

- [ ] `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` set on Render
- [ ] `GITHUB_OAUTH_REDIRECT_URL` = `https://gitactionflow.vercel.app/auth/github/callback` (Vercel proxy)
- [ ] GitHub OAuth App homepage/callback match production Vercel URLs
- [ ] `SESSION_SECRET` ≥ 32 chars
- [ ] `COOKIE_SECURE=true`, `COOKIE_SAMESITE=None` (or production defaults)
- [ ] `FRONTEND_URL` exact Vercel origin (no trailing slash)
- [ ] Login and logout work from Vercel

## Frontend

- [ ] `VITE_API_BASE_URL` **unset** (same-origin `/api` + `/auth` proxy in `vercel.json`)
- [ ] Production build succeeds
- [ ] SPA routes (`/login`, `/dashboard`) work on refresh
- [ ] Authenticated API calls send cookies (`credentials: 'include'`)
- [ ] Vercel Deployment Protection / Require Log In is OFF for Production

## GitHub

- [ ] One repository connected in the app
- [ ] Webhook URL = `https://<render>/webhooks/github`
- [ ] Content type `application/json`
- [ ] Issues + Pull requests subscribed
- [ ] Webhook secret matches `GITHUB_WEBHOOK_SECRET`
- [ ] Delivery shows HTTP 200

## Slack

- [ ] `SLACK_WEBHOOK_URL` set on Render only
- [ ] Notification tested end-to-end

## Security

- [ ] No secrets committed / no real values in `.env.example`
- [ ] No secrets in Vercel / frontend bundle
- [ ] CORS allows only `FRONTEND_URL` (not `*`)
- [ ] Logs do not print secrets (use redacted config)

## End-to-end

- [ ] Login
- [ ] Connect repository
- [ ] Create rule
- [ ] Trigger issue
- [ ] Trigger pull request (optional confirmation)
- [ ] GitHub label and/or comment
- [ ] Slack notification
- [ ] Event history
- [ ] Action history
- [ ] Duplicate delivery handling

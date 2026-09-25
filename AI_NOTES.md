# AI Notes

Honest record of how AI tools were used on GitActionFlow.

## AI Tools Used

- Cursor agent (Composer) for Phases 7–10, deployment (Neon/Render/Vercel), and production hardening

## How AI Was Used

- Phases 7–9: rules, actions, dashboard UI, history APIs
- Deployment: Render/Vercel/Neon wiring, cookie/CORS debugging, same-origin Vercel proxy
- Phase 10: security middleware, optional AI abstraction, assignment matrix/verification docs

## Engineering Decisions Made by Me

- Free stack: Neon + Render + Vercel
- Prefer Vercel **same-origin proxy** for `/api` and `/auth` so session cookies work when third-party cookies are blocked (Incognito)
- Keep GitHub **OAuth App** (no rushed GitHub App migration)
- One repository per user (skip multi-repo stretch)
- Optional AI behind `AI_ENABLED` with schema allowlisting; core path never depends on AI
- CSRF Origin checks on mutating cookie APIs; do not Origin-gate GitHub webhooks

## Incorrect AI Suggestion / Hardest AI Mistake

1. Assuming cross-site `SameSite=None` alone was enough for Vercel↔Render sessions in Incognito — browsers blocked third-party cookies (“Failed to fetch” / login loop).
2. Removing the SPA fallback from `vercel.json` while adding API proxies caused `/login` and `/dashboard` **404**.
3. Leaving `VITE_API_BASE_URL` set to the Render URL after the proxy shipped kept the login button on `onrender.com`, defeating first-party cookies.

## How I Corrected It

- Proxied `/api` + `/auth` through Vercel; OAuth callback on the Vercel host; unset `VITE_API_BASE_URL` in production builds
- Restored SPA `/(.*)` → `index.html` **after** API/auth rewrites
- Documented the runbook in local `DEPLOYMENT_URLS.local.md` and updated deployment docs

## Optional product AI (runtime)

When `AI_ENABLED=true` and a provider key is set, rules may set `use_ai` / `append_ai_summary` in action config. Failures skip enrichment; static config still runs.

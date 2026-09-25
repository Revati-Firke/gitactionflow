# AI Notes

Honest record of how AI tools were used on GitActionFlow.

## AI Tools Used

- Cursor agent (Composer) for Phases 7–9 and deployment preparation (Neon/Render/Vercel)

## How AI Was Used

- Phases 7–9: rules, actions, dashboard UI, history APIs.
- Deployment prep: `PORT` preference for Render, production cookie defaults for cross-site SPA, `vercel.json`, Render/Neon docs, smoke checklist — **no claim of a live deploy without account access**.

## Engineering Decisions Made by Me

- Free stack: Neon Postgres + Render Go Docker + Vercel static SPA.
- Prefer platform `PORT` over `APP_PORT` without breaking local defaults.
- Production default `COOKIE_SAMESITE=None` + `COOKIE_SECURE=true` so Vercel can call Render with session cookies.
- Migrations via existing `AUTO_MIGRATE` on startup (idempotent) rather than a new migrate binary.

## Incorrect AI Suggestion / Hardest AI Mistake

- Assuming Vite proxy alone works with OAuth cookies (cookie host mismatch). Production split hosting needs explicit API base + `SameSite=None`.
- Claiming “deployed successfully” without dashboard access would be false — docs distinguish preparation vs verification.

## How I Corrected It

- Documented cookie/CORS model; kept placeholders for live URLs; smoke tests marked NOT YET RUN until manually verified.

## What I Would Improve

- Run the live smoke test on real Neon/Render/Vercel accounts
- Optional AI stretch and automatic webhook registration if time allows

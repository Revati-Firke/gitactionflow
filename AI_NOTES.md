# AI_NOTES.md

How I used AI on GitActionFlow for the Abstrabit take-home.

## Tools and how I split the work

- **Cursor** (Composer / agent) for most of the build: Go packages, React dashboard, tests, deploy wiring, and first drafts of docs.
- **ChatGPT** for research—comparing free-tier hosts, OAuth App vs GitHub App trade-offs, cookie/`SameSite` behavior, Slack Incoming Webhooks, and clarifying GitHub webhook signature/idempotency docs. I used it to think through options, not to paste large generated patches into the repo.
- Cursor was the daily coding driver; ChatGPT stayed in the “read / compare / decide” lane.

**Rough split**

| Me | AI |
| --- | --- |
| Product scope cuts (one repo, OAuth App, free stack, what to skip) | Cursor: scaffolding handlers/stores once the design was clear |
| Architecture shape (modular monolith, Postgres as queue, rule intents vs executor) | Cursor: boilerplate SQL/migrations, unit tests, middleware drafts |
| Deploy provider choice + cookie/proxy diagnosis after Incognito broke | ChatGPT: research on hosts/cookies/OAuth; Cursor: first-pass `vercel.json` / env wiring |
| Reviewing every security-sensitive change (HMAC, sessions, secrets) | Speeding up repetitive edits; doc drafts I then rewrote |

I treated AI as a fast junior pair: good for volume, not trusted for judgment until I read the diff.

## Decisions I made myself

1. **Modular monolith + PostgreSQL as the queue**  
   One Go process on Render, React on Vercel, Neon for durable state. I rejected Redis/Kafka/microservices—they don’t fit free-tier ops or the brief. Worker claims rows with `FOR UPDATE SKIP LOCKED` so cold starts don’t lose events.

2. **GitHub OAuth App + one connected repo (not a GitHub App, not multi-repo)**  
   The assignment needs sign-in, connect, and write-back. An OAuth App with `read:user repo` was enough. GitHub App install flow and multi-repo product features would have burned time without improving the demo.

3. **Vercel same-origin proxy for `/api` and `/auth`**  
   After production login failed in Incognito, I chose to proxy browser traffic through Vercel so the session cookie is first-party, instead of relying on cross-site cookies to Render. Leave `VITE_API_BASE_URL` unset in production so the build uses relative URLs.

## Hardest wrong turn the AI led me into

**What it got wrong:** Treating cross-origin `SameSite=None; Secure` cookies (Vercel SPA → Render API) as “done” for production sessions.

**How it showed up:** Fresh Incognito login looked successful, then the dashboard couldn’t call the API (“Failed to fetch” / login loop). Third-party cookies to `*.onrender.com` were blocked.

**How I noticed:** Network tab showed credentialed requests to the Render origin failing while the Vercel page loaded fine; health on Render was OK, so it wasn’t a dead backend.

**How I fixed it:**

1. Rewrote `vercel.json` to proxy `/api/*` and `/auth/*` to Render, with SPA `/(.*)` → `index.html` **after** those rewrites (an earlier edit dropped the SPA fallback and 404’d `/login` / `/dashboard`).
2. Pointed the GitHub OAuth callback at the **Vercel** host (`/auth/github/callback`).
3. Removed production `VITE_API_BASE_URL` pointing at Render so the login button no longer jumped cross-origin.

That sequence was the clearest case of “looks correct in theory, fails in a real browser”—I had to override the AI’s cookie-only approach with hosting topology.

### Optional excerpt (cookie / proxy fix)

Rough prompt I used when stuck:

> Login works on Vercel but Incognito loses the session calling Render. SameSite=None is set. Is this third-party cookie blocking? Prefer a same-origin fix on Vercel rather than asking users to allow third-party cookies.

I kept the proxy approach and discarded “just document SameSite=None” as the production answer.

## What I’d improve with more time

- Auto-create/delete the GitHub webhook when connecting/disconnecting a repo
- Create missing labels via the GitHub API before applying them (avoid demo 422s)
- Always smoke a live PR + webhook Redeliver before calling a release done
- Shared rate limiting across instances (limits today are per process)

## Runtime AI (product, optional)

Separate from coding assistance: `AI_ENABLED` (default false) can enrich actions with a short summary / suggested labels via Gemini or Groq. Failures skip enrichment; static label/comment/Slack still run. Core path never depends on it.

# AI_NOTES.md

How AI tools were used while building GitActionFlow — useful context for anyone reviewing or extending the project.

## Tools and how work was split

- **Cursor** (Composer / agent) for most of the build: Go packages, React dashboard, tests, deploy wiring, and first drafts of docs.
- **ChatGPT** for research—comparing free-tier hosts, OAuth App vs GitHub App trade-offs, cookie/`SameSite` behavior, Slack Incoming Webhooks, and clarifying GitHub webhook signature/idempotency docs. Used to think through options, not to paste large generated patches into the repo.
- Cursor was the daily coding driver; ChatGPT stayed in the “read / compare / decide” lane.

**Rough split**

| Human | AI |
| --- | --- |
| Product scope cuts (one repo, OAuth App, free stack, what to skip) | Cursor: scaffolding handlers/stores once the design was clear |
| Architecture shape (modular monolith, Postgres as queue, rule intents vs executor) | Cursor: boilerplate SQL/migrations, unit tests, middleware drafts |
| Deploy provider choice + cookie/proxy diagnosis after Incognito broke | ChatGPT: research on hosts/cookies/OAuth; Cursor: first-pass `vercel.json` / env wiring |
| Reviewing every security-sensitive change (HMAC, sessions, secrets) | Speeding up repetitive edits; doc drafts that were then rewritten |

AI was treated as a fast junior pair: good for volume, not trusted for judgment until the diff was read.

## Decisions made by the author

1. **Modular monolith + PostgreSQL as the queue**  
   One Go process on Render, React on Vercel, Neon for durable state. Redis/Kafka/microservices were rejected—they don’t fit free-tier ops or this product’s size. Worker claims rows with `FOR UPDATE SKIP LOCKED` so cold starts don’t lose events.

2. **GitHub OAuth App + one connected repo (not a GitHub App, not multi-repo)**  
   Sign-in, connect, and write-back needed an OAuth App with `read:user repo`. GitHub App install flow and multi-repo product features would have added surface area without improving the core loop.

3. **Vercel same-origin proxy for `/api` and `/auth`**  
   After production login failed in Incognito, browser traffic is proxied through Vercel so the session cookie is first-party, instead of relying on cross-site cookies to Render. Leave `VITE_API_BASE_URL` unset in production so the build uses relative URLs.

## Hardest wrong turn AI led into

**What it got wrong:** Treating cross-origin `SameSite=None; Secure` cookies (Vercel SPA → Render API) as “done” for production sessions.

**How it showed up:** Fresh Incognito login looked successful, then the dashboard couldn’t call the API (“Failed to fetch” / login loop). Third-party cookies to `*.onrender.com` were blocked.

**How it was noticed:** Network tab showed credentialed requests to the Render origin failing while the Vercel page loaded fine; health on Render was OK, so it wasn’t a dead backend.

**How it was fixed:**

1. Rewrote `vercel.json` to proxy `/api/*` and `/auth/*` to Render, with SPA `/(.*)` → `index.html` **after** those rewrites (an earlier edit dropped the SPA fallback and 404’d `/login` / `/dashboard`).
2. Pointed the GitHub OAuth callback at the **Vercel** host (`/auth/github/callback`).
3. Removed production `VITE_API_BASE_URL` pointing at Render so the login button no longer jumped cross-origin.

That sequence was the clearest case of “looks correct in theory, fails in a real browser”—the AI’s cookie-only approach had to be overridden with hosting topology.

### Optional excerpt (cookie / proxy fix)

Rough prompt used when stuck:

> Login works on Vercel but Incognito loses the session calling Render. SameSite=None is set. Is this third-party cookie blocking? Prefer a same-origin fix on Vercel rather than asking users to allow third-party cookies.

The proxy approach was kept; “just document SameSite=None” was discarded as the production answer.

## What I’d improve with more time

- Auto-create/delete the GitHub webhook when connecting/disconnecting a repo
- Create missing labels via the GitHub API before applying them (avoid 422s on unknown labels)
- Always smoke a live PR + webhook Redeliver before calling a release done
- Shared rate limiting across instances (limits today are per process)

## Runtime AI (product, optional)

Separate from coding assistance: `AI_ENABLED` (default false) can enrich actions with a short summary / suggested labels via Gemini or Groq. Failures skip enrichment; static label/comment/Slack still run. Core path never depends on it.

# Frontend

React + TypeScript + Vite UI for GitActionFlow.

**Phase 4:** Sign in and connect one GitHub repository.

## Run

See **[docs/setup/LOCAL.md](../docs/setup/LOCAL.md)** for OAuth + full local steps.

```bash
# backend must already be on :8080 with .env configured
npm install
npm run dev
```

Open http://localhost:5173 — always use `localhost` (not `127.0.0.1`) so the session cookie matches.

## Not included yet

Rules UI, event/action logs, webhook setup, Slack, AI.

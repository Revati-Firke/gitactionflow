# Frontend

React + TypeScript + Vite dashboard: repository connect/disconnect, rules CRUD, event and action history.

## Run

See **[docs/setup/LOCAL.md](../docs/setup/LOCAL.md)** for OAuth + full local steps.

```bash
# backend must already be on :8080 with .env configured
npm install
npm run dev
```

Open http://localhost:5173 — always use `localhost` (not `127.0.0.1`) so the session cookie matches.

Default API base: `http://localhost:8080` (`VITE_API_BASE_URL` to override).

## Vercel

| Setting | Value |
| --- | --- |
| Root | `frontend` |
| Build | `npm run build` |
| Output | `dist` |
| Env | `VITE_API_BASE_URL=https://<render-backend>` |
| SPA | `vercel.json` rewrites |

See [docs/deployment/README.md](../docs/deployment/README.md).

## Build

```bash
npm run build
```

## Structure

- `src/pages/` — Login, Dashboard
- `src/components/` — Layout, repository, rules, events, actions, badges
- `src/services/api/` — cookie-authenticated fetch client
- `src/types/` — API response types

## Not included

AI UI, multi-repository management, workflow builder, WebSockets.

# Local setup

```bash
cp .env.example .env    # OAuth App + SESSION_SECRET + webhook secret
docker-compose up -d postgres
cd backend && set -a && source ../.env && set +a && go run ./cmd/server
cd frontend && npm install && npm run dev
```

Open http://localhost:5173 — use **`localhost`**, not `127.0.0.1`.

OAuth App (local): homepage `http://localhost:5173`, callback `http://localhost:8080/auth/github/callback`.

1. Login → connect one admin repo  
2. Rule e.g. keyword `bug` → Slack or comment (label only if it exists on the repo)  
3. Point repo webhook at your tunnel `/webhooks/github` with `GITHUB_WEBHOOK_SECRET`  
4. Open a matching issue → check dashboard Events/Actions  

Webhook create-on-connect is manual by design. Optional AI stays off unless you set `AI_ENABLED=true`.

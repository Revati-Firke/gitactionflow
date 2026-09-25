# Local setup & test (through Phase 9)

OAuth, one connected repo, webhooks, rule/action execution, and the React dashboard.

---

## Start

```bash
docker-compose up -d postgres
cd backend && set -a && source ../.env && set +a && go run ./cmd/server
# other terminal
cd frontend && npm run dev
```

Open http://localhost:5173 — **Continue with GitHub** → dashboard.

Use `localhost` (not `127.0.0.1`) for both UI and API hosts in `.env`.

Optional: set `SLACK_WEBHOOK_URL` for Slack actions.

---

## Demo flow

1. Sign in with GitHub.
2. Connect one repository (admin required).
3. Create a rule (e.g. issues + keyword `bug` → GitHub label `automation`).
4. Ensure the repo webhook points at your public tunnel `/webhooks/github` with the shared secret.
5. Open/update a matching issue → event and action appear on the dashboard; GitHub/Slack side effects run.

---

## Useful APIs

| Method | Path |
| --- | --- |
| GET | `/api/me` |
| GET/POST/DELETE | `/api/repository` |
| CRUD | `/api/rules` |
| GET | `/api/events` |
| GET | `/api/actions` |

---

## Not included yet

AI, automatic webhook registration on connect.

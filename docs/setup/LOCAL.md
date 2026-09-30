# Local setup

Canonical short version: root `README.md`. Env template: `.env.example`.

---

## 1. One-time setup

```bash
cp .env.example .env
# Edit .env: GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, SESSION_SECRET (>=32),
#            GITHUB_WEBHOOK_SECRET (>=16), DATABASE_URL (compose default OK)

openssl rand -hex 32   # SESSION_SECRET
openssl rand -hex 32   # GITHUB_WEBHOOK_SECRET
```

**GitHub OAuth App (local):**

| Field | Value |
| --- | --- |
| Homepage | `http://localhost:5173` |
| Callback | `http://localhost:8080/auth/github/callback` |

Keep the production OAuth App separate (Vercel homepage + callback).

---

## 2. Start everything (3 terminals)

### Terminal A — Postgres

```bash
docker compose up -d postgres
docker compose ps
```

Default DB URL (also in `.env.example`):

```text
postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable
```

### Terminal B — Backend (Go)

```bash
cd backend
set -a && source ../.env && set +a
go run ./cmd/server
```

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready
```

Expect `{"status":"ok"}` and `{"status":"ready"}`.

### Terminal C — Frontend (Vite)

```bash
cd frontend
npm install
npm run dev
```

Open **`http://localhost:5173`** — use **`localhost`**, not `127.0.0.1` (session cookie).

API base in dev defaults to `http://localhost:8080`.

---

## 3. Tests and rebuilds

```bash
cd backend && go test ./... && go vet ./...
cd frontend && npm run build
```

Reset DB volume (destroys local data):

```bash
docker compose down -v
docker compose up -d postgres
```

With `AUTO_MIGRATE=true`, the backend re-applies migrations on start.

---

## 4. Optional: webhook from GitHub → local

GitHub cannot hit `localhost` directly. Tunnel `:8080`:

```bash
ngrok http 8080
# Payload URL: https://<tunnel>/webhooks/github
# Content-Type: application/json
# Secret: same as GITHUB_WEBHOOK_SECRET in .env
# Events: Issues + Pull requests
```

Without a tunnel you can still exercise login, rules CRUD, and the dashboard — not live webhook ingest.

---

## 5. Optional: AI / Slack locally

```bash
AI_ENABLED=true
AI_PROVIDER=gemini
GEMINI_API_KEY=...

SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
```

Restart the backend after env changes.

---

## 6. How events are processed → UI status

```text
GitHub POST /webhooks/github
  → HMAC on raw body (fail → 401, never persisted)
  → UNIQUE delivery_id (dup → 200 already_received)
  → INSERT webhook_events status = pending
  → HTTP 200 accepted   ← no GitHub/Slack here

Worker (same Go process, EVENT_WORKER_ENABLED=true)
  → ClaimNext: FOR UPDATE SKIP LOCKED
  → Validate + ExtractEventContext
  → rules.Evaluate → ActionIntent[]   (no side effects)
  → Executor.EnsureAndExecute
  → MarkProcessed / ScheduleRetry / MarkFailed
```

| Outcome | Event `status` |
| --- | --- |
| All actions done (or no intents) | `processed` |
| Permanent error / retries exhausted | `failed` |
| Transient error / actions still pending retry | `pending` (+ backoff) |
| Mid-work | `processing` |

Action rows: `pending` → `processing` → `completed` | `failed`.

The dashboard reads Postgres for the connected repo only — no WebSockets; use **Refresh** after a webhook.

**Failed** usually means a permanent GitHub/Slack error, validation failure, or retries exhausted. Slack can succeed while a label/comment fails → parent event still **Failed**.

**Redeliver** with the same `X-GitHub-Delivery` returns `already_received` — no new event row (idempotent by design).

---

## 7. Quick smoke

1. Open http://localhost:5173 → Continue with GitHub  
2. Connect one admin repo  
3. Create a rule (e.g. keyword `bug` → comment or Slack)  
4. With a tunnel: open a matching issue  
5. Dashboard → Refresh → Events **Processed** / Actions **Completed**

---

## 8. Useful paths

| Area | Location |
| --- | --- |
| Routes | `backend/internal/http/router/` |
| Webhook | `backend/internal/webhook/`, `handlers/webhook.go` |
| Worker | `backend/internal/events/` |
| Rules | `backend/internal/rules/` |
| Actions | `backend/internal/actions/` |
| Activity API | `backend/internal/http/handlers/activity.go` |
| Dashboard UI | `frontend/src/pages/`, `frontend/src/components/` |

---

## 9. Optional full Docker stack

```bash
docker compose up -d --build
# Postgres + backend on :8080; still run frontend with npm run dev
```

Typical local path: **compose postgres only** + `go run` + `npm run dev`.

# Local setup & test (through Phase 4)

Short guide to run GitActionFlow locally: OAuth, one connected repo, minimal UI.

**Not included yet:** webhooks, rules, Slack, labels/comments, event logs.

---

## 1. Prerequisites

- Go 1.25+
- Node.js 20+ (for frontend)
- Docker + `docker-compose`
- A GitHub account

---

## 2. Create a GitHub OAuth App

1. Open [GitHub → Settings → Developer settings → OAuth Apps](https://github.com/settings/developers) → **New OAuth App**.
2. Set:

| Field | Value |
| --- | --- |
| Application name | `GitActionFlow Local` |
| Homepage URL | `http://localhost:5173` |
| Authorization callback URL | `http://localhost:8080/auth/github/callback` |

3. Register → copy **Client ID** → **Generate a new client secret** → copy secret.

Use **OAuth App**, not GitHub App. Callback must match exactly (no trailing slash).

Scopes requested by the app at login: `read:user repo`.

---

## 3. Configure `.env`

```bash
cd "/path/to/GitActionFlow"
cp .env.example .env
```

Set at least:

```bash
APP_ENV=development
APP_PORT=8080
AUTO_MIGRATE=true
DATABASE_URL=postgres://gitactionflow:gitactionflow@localhost:5432/gitactionflow?sslmode=disable

FRONTEND_URL=http://localhost:5173
GITHUB_CLIENT_ID=your_real_client_id
GITHUB_CLIENT_SECRET=your_real_client_secret
GITHUB_OAUTH_REDIRECT_URL=http://localhost:8080/auth/github/callback
SESSION_SECRET=$(openssl rand -hex 32)   # must be >= 32 chars

COOKIE_SECURE=false
COOKIE_SAMESITE=Lax
```

Never commit `.env`.

**Host rule:** use `localhost` everywhere (OAuth callback, frontend, API). Do **not** mix `localhost` and `127.0.0.1` — session cookies will not match and you will stay on the login screen after Authorize.

---

## 4. Start services

```bash
# Postgres
docker-compose up -d postgres

# Backend (terminal 1)
cd backend
set -a && source ../.env && set +a
go run ./cmd/server

# Frontend (terminal 2)
cd frontend
npm install
npm run dev
```

Open **http://localhost:5173**

Quick checks:

```bash
curl -sS http://localhost:8080/health   # {"status":"ok"}
curl -sS http://localhost:8080/ready    # {"status":"ready"}
```

---

## 5. Manual test flow

1. Click **Sign in with GitHub** → authorize on GitHub.
2. You return to the app **signed in** (username shown).
3. Repo list appears → **Connect** one repo you admin/own.
4. Connected panel shows name, owner, visibility, branch, **Open on GitHub**.
5. Connecting a second repo without disconnect → blocked (`409`).
6. **Disconnect** → list returns; **Log out** → login screen.

Automated checks (no GitHub needed):

```bash
cd backend && go test ./... && go vet ./...
cd ../frontend && npx tsc --noEmit
```

---

## 6. Common problems

| Symptom | Fix |
| --- | --- |
| GitHub **404** on authorize | `.env` still has placeholder `your_github_client_id` — paste real Client ID/secret and **restart backend** |
| `redirect_uri_mismatch` | OAuth App callback ≠ `GITHUB_OAUTH_REDIRECT_URL` |
| Authorize works, then back to **Sign in** | You mixed `localhost` / `127.0.0.1` — use `localhost` only; hard-refresh frontend |
| Backend won’t start (`SESSION_SECRET`) | Secret must be at least 32 characters |
| `/ready` is 503 | Start Postgres: `docker-compose up -d postgres` |

---

## 7. What “connected” means

Connected = repo is saved in Postgres for your user.  
It does **not** register webhooks or run automation yet (Phase 5+).

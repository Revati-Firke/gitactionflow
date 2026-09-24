# Local setup & test (through Phase 5)

Short guide: OAuth, connect one repo, ingest signed GitHub webhooks, ingest signed GitHub webhooks.

**Not included yet:** event processing, rules, Slack, labels/comments, event dashboard UI.

---

## 1. Prerequisites

- Go 1.25+
- Node.js 20+ (frontend)
- Docker + `docker-compose`
- GitHub account
- For live webhooks: a public HTTPS tunnel to `:8080` (Cloudflare Tunnel, ngrok, etc.)

---

## 2. Create a GitHub OAuth App

1. [OAuth Apps → New OAuth App](https://github.com/settings/developers)
2. Homepage: `http://localhost:5173`
3. Callback: `http://localhost:8080/auth/github/callback`
4. Copy Client ID + Client secret

Scopes at login: `read:user repo`.

---

## 3. Configure `.env`

```bash
cp .env.example .env
```

Required highlights:

```bash
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
GITHUB_OAUTH_REDIRECT_URL=http://localhost:8080/auth/github/callback
SESSION_SECRET=$(openssl rand -hex 32)          # >= 32 chars
GITHUB_WEBHOOK_SECRET=$(openssl rand -hex 32)   # >= 16 chars; same value on GitHub webhook
FRONTEND_URL=http://localhost:5173
WEBHOOK_MAX_BODY_BYTES=1048576                  # 1 MiB
```

Use **`localhost` only** (not `127.0.0.1`) for OAuth cookie matching. Never commit `.env`.

---

## 4. Start services

```bash
docker-compose up -d postgres

cd backend && set -a && source ../.env && set +a && go run ./cmd/server
# other terminal:
cd frontend && npm install && npm run dev
```

Open http://localhost:5173

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready
```

---

## 5. Test OAuth + connect repo

1. Sign in with GitHub → authorize.
2. Connect one admin/owned repository.
3. Second connect without disconnect → blocked.

---

## 6. Configure GitHub webhook (manual)

On the **connected** repository: **Settings → Webhooks → Add webhook**

| Field | Value |
| --- | --- |
| Payload URL | `https://<public-host>/webhooks/github` (tunnel to local `:8080`) |
| Content type | `application/json` |
| Secret | same as `GITHUB_WEBHOOK_SECRET` |
| Events | **Issues** and **Pull requests** |

Ping → **200** `ignored` / `unsupported_event` (expected). Issue open → `accepted` and DB row `pending`.

### Local signed request

```bash
SECRET='your_webhook_secret'
BODY='{"action":"opened","repository":{"id":YOUR_GITHUB_REPO_NUMERIC_ID}}'
SIG="sha256=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')"

curl -sS -X POST http://localhost:8080/webhooks/github \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: issues" \
  -H "X-GitHub-Delivery: $(uuidgen)" \
  -H "X-Hub-Signature-256: $SIG" \
  -d "$BODY"
```

Check DB (wait a few seconds for the worker):

```bash
docker-compose exec -T postgres psql -U gitactionflow -d gitactionflow \
  -c "SELECT delivery_id, event_type, status, retry_count, last_error FROM webhook_events ORDER BY received_at DESC LIMIT 5;"
```

Expect `status = pending` (processing is Phase 6).

---

## 7. Automated tests

```bash
cd backend && go test ./... && go vet ./...
```

---

## 8. Common problems

| Symptom | Fix |
| --- | --- |
| GitHub authorize **404** | Placeholder Client ID — use real values, restart backend |
| Authorize then still **Sign in** | Mixed `localhost` / `127.0.0.1` |
| Backend won’t start (`GITHUB_WEBHOOK_SECRET`) | Set secret ≥ 16 chars |
| Webhook **401** | Wrong secret or body altered before HMAC |
| Webhook ignored / unknown_repository | Repo not connected, or `repository.id` mismatch |
| Duplicate delivery | Same `X-GitHub-Delivery` → `already_received` |

---

## 9. Scope reminder

| Done | Not yet |
| --- | --- |
| Secure ingest + persist `pending` | Rules, labels, comments, Slack, AI, dashboard logs |

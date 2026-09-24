# Local setup & test (through Phase 7)

Short guide: OAuth, connect one repo, webhooks, background processing, and configurable rules (intents only — no GitHub/Slack writes yet).

**Not included yet:** GitHub/Slack action execution, AI, event dashboard UI.

---

## Start

```bash
docker-compose up -d postgres
cd backend && set -a && source ../.env && set +a && go run ./cmd/server
# other terminal
cd frontend && npm run dev
```

Open http://localhost:5173 — sign in and connect one admin repo.

---

## Create a rule

With a session cookie after browser login (or `-b` jar from curl login flow):

```bash
curl -sS -b /tmp/gaf.jar -X POST http://localhost:8080/api/rules \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Bug Issues",
    "enabled": true,
    "event_type": "issues",
    "keyword": "bug",
    "required_labels": [],
    "action_type": "github_label",
    "action_config": { "label": "automation" }
  }'
```

`GET /api/rules` lists rules for the connected repository.

---

## Webhook → rules → processed

Open an issue containing `bug` in the title/body (webhook must point at your tunnel).

Expect: ingest `accepted` → worker `rule matched` → DB `processed`. **No** GitHub label yet.

```bash
docker-compose exec -T postgres psql -U gitactionflow -d gitactionflow \
  -c "SELECT delivery_id, status FROM webhook_events ORDER BY received_at DESC LIMIT 5;"
```

---

## Tests

```bash
cd backend && go test ./... && go vet ./...
```

# Backend

Go HTTP API for GitActionFlow.

**Phase 6 status:** Foundation + OAuth + one-repo connection + webhook ingestion + durable event worker.

## Requirements

- Go **1.25+**
- PostgreSQL 16
- GitHub OAuth App (`read:user repo`)
- `GITHUB_WEBHOOK_SECRET` (≥ 16 chars)
- Event worker env (defaults OK): `EVENT_WORKER_ENABLED`, `EVENT_WORKER_POLL_INTERVAL`, `EVENT_MAX_RETRIES`, `EVENT_PROCESSING_LEASE`

## Quick start

Full local steps: [docs/setup/LOCAL.md](../docs/setup/LOCAL.md)

```bash
docker-compose up -d postgres
cd backend
set -a && source ../.env && set +a
go run ./cmd/server
```

## Endpoints

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/health` | no |
| GET | `/ready` | no |
| GET | `/auth/github` | no |
| GET | `/auth/github/callback` | no |
| POST | `/auth/logout` | cookie optional |
| GET | `/api/me` | session |
| GET | `/api/github/repositories` | session |
| GET | `/api/repository` | session |
| POST | `/api/repository` | session |
| DELETE | `/api/repository` | session |
| POST | `/webhooks/github` | HMAC signature |

## Tests

```bash
go test ./...
gofmt -l .
go vet ./...
```

## Not implemented yet

Event processor, rules, Slack, AI, event dashboard.

# Backend

Go HTTP API for GitActionFlow: OAuth, one-repo connection, webhooks, event worker, rules, GitHub/Slack actions, and dashboard history APIs.

## Requirements

- Go **1.25+**
- PostgreSQL 16
- GitHub OAuth App (`read:user repo`)
- `GITHUB_WEBHOOK_SECRET` (≥ 16 chars)
- Event worker + action env (defaults OK): `EVENT_*`, `ACTION_MAX_RETRIES`, optional `SLACK_WEBHOOK_URL`

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
| GET/POST/DELETE | `/api/repository` | session |
| GET/POST/PUT/DELETE | `/api/rules` | session |
| GET | `/api/events` | session |
| GET | `/api/actions` | session |
| POST | `/webhooks/github` | HMAC signature |

## Tests

```bash
go test ./...
go vet ./...
```

## Not implemented yet

AI, automatic webhook registration on connect.

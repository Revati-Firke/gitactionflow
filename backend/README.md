# Backend

Go HTTP API for GitActionFlow.

**Phase 4 status:** Foundation + OAuth + one-repository connection.

## Requirements

- Go **1.25+**
- PostgreSQL 16
- GitHub OAuth App (`read:user repo`)

## Quick start

Full local OAuth + test steps: [docs/setup/LOCAL.md](../docs/setup/LOCAL.md)

```bash
docker-compose up -d postgres
# configure ../.env then:
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

Connect requires GitHub **admin** on the repo. One connected repo per user.

## Tests

```bash
go test ./...
gofmt -l .
go vet ./...
```

## Not implemented yet

Webhooks, rules, Slack, AI, event dashboard.

# Backend

Go HTTP API for GitActionFlow.

**Phase 3 status:** Foundation + GitHub OAuth authentication.

## Requirements

- Go **1.25+**
- PostgreSQL 16 (local via Docker Compose)
- GitHub OAuth App credentials

## Layout

```text
backend/
├── cmd/server/
├── internal/
│   ├── app/
│   ├── auth/           # cookies, crypto, context
│   ├── config/
│   ├── database/
│   ├── githuboauth/    # GitHub OAuth HTTP client
│   ├── http/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── response/
│   │   └── router/
│   ├── logging/
│   └── store/          # users, sessions, oauth_states
├── migrations/
├── Dockerfile
├── go.mod
└── go.sum
```

## Quick start

```bash
docker-compose up -d postgres
cp ../.env.example ../.env   # fill GitHub + SESSION_SECRET
cd backend
set -a && source ../.env && set +a
go run ./cmd/server
```

## Auth endpoints

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/auth/github` | no |
| GET | `/auth/github/callback` | no |
| POST | `/auth/logout` | cookie optional (idempotent) |
| GET | `/api/me` | session cookie required |

## Tests

```bash
go test ./...
gofmt -l .
go vet ./...
```

## Not implemented yet

Repository connect, webhooks, rules, Slack, AI, dashboard APIs.

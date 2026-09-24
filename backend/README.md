# Backend

Go HTTP API and event processor for GitActionFlow.

**Phase 2 status:** Backend foundation is implemented (config, PostgreSQL, migrations, `/health`, `/ready`, graceful shutdown).

## Requirements

- Go **1.25+** (developed with Go 1.25.6)
- PostgreSQL 16 (local via Docker Compose)

## Layout

```text
backend/
├── cmd/server/          # process entrypoint
├── internal/
│   ├── app/             # startup / shutdown wiring
│   ├── config/          # environment-based configuration
│   ├── database/        # pgx pool + migrate helpers
│   ├── http/
│   │   ├── handlers/    # health / ready
│   │   ├── response/    # consistent JSON errors
│   │   └── router/      # Gin router
│   └── logging/         # structured slog logger
├── migrations/          # versioned SQL (up/down)
├── Dockerfile
├── go.mod
└── go.sum
```

## Quick start

From the repository root:

```bash
# 1. Start Postgres
docker-compose up -d postgres

# 2. Configure env (or export variables)
cp .env.example .env
# ensure DATABASE_URL points at localhost:5432

# 3. Run the API (auto-migrates when AUTO_MIGRATE=true)
cd backend
export $(grep -v '^#' ../.env | xargs)   # or set vars manually
go run ./cmd/server
```

Verify:

```bash
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
```

## Tests

```bash
cd backend
go test ./...
gofmt -l .
go vet ./...
```

## Not implemented yet

GitHub OAuth, webhooks, rules, Slack, AI, dashboard APIs, and React UI belong to later phases.

See root `README.md` and `docs/architecture/HLA.md`.

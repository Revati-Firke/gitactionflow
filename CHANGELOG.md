# Changelog

All notable changes to GitActionFlow will be documented here.

Format inspired by [Keep a Changelog](https://keepachangelog.com/). Versions follow Semantic Versioning when releases begin.

## [Unreleased]

### Added

- Phase 6 durable event processing: worker, retries, stale lease recovery, processed/failed states
- Phase 5 webhook ingestion: signature verification, delivery idempotency, pending event persistence
- Phase 4 repository management: list/connect/disconnect one GitHub repo, minimal React UI
- Phase 3 GitHub OAuth authentication: state CSRF, sessions, `/api/me`, logout, encrypted token storage
- Phase 2 backend foundation: Go/Gin server, config, pgx pool, migrations, `/health`, `/ready`, logging, graceful shutdown, Dockerfile, foundation tests
- Phase 1 project foundation and documentation
- Repository structure (`backend/`, `frontend/`, `docs/`)
- Architecture, API plan, database direction, deployment direction, ADR index
- `AGENTS.md`, `AI_NOTES.md`, `SECURITY.md`, `CONTRIBUTING.md`
- `.env.example`, `.gitignore`, `docker-compose.yml` (PostgreSQL + optional backend)
- MIT License

### Not yet available

- Rules, Slack, AI, full dashboard, production deploy, auto webhook registration

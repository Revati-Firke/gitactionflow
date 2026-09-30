# Contributing to GitActionFlow

Thanks for taking an interest. Prefer small, focused changes that respect the product scope in `README.md` and `AGENTS.md`.

## Before you start

1. Read `README.md`, `AGENTS.md`, and the ADRs under `docs/decisions/`.
2. Prefer incremental improvements over rewrites.
3. Do not add out-of-scope products or infrastructure without an explicit decision (and ideally a new ADR).

## Local workflow

1. Copy `.env.example` → `.env` (placeholders only; never commit real secrets).
2. `docker compose up -d` for Postgres.
3. Run backend (`backend/`) and frontend (`frontend/`) as in `docs/setup/LOCAL.md`.
4. Add or update tests for behavior you change.
5. Update docs when architecture or behavior changes.

## Commits

Conventional Commits: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `security`.

Avoid vague messages (`update`, `changes`, `final`, `stuff`).

## Review focus

- Changes stay inside product scope (see `AGENTS.md`)
- Secrets not committed or logged
- Docs match what actually runs
- Errors handled explicitly
- Security/reliability paths have tests where it matters

## Security

See `SECURITY.md`. Rotate credentials immediately if anything sensitive leaks.

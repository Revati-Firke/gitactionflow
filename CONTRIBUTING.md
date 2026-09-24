# Contributing to GitActionFlow

Thank you for working on this project. GitActionFlow is a take-home assignment codebase; keep changes aligned with the assignment scope described in `README.md` and `AGENTS.md`.

## Before you start

1. Read `README.md` and `AGENTS.md`.
2. Confirm which phase you are implementing.
3. Prefer small, focused changes.
4. Do not add out-of-scope products or infrastructure.

## Development workflow (planned)

Exact tooling land in later phases. Expected flow:

1. Copy `.env.example` to `.env` and configure locally.
2. Start PostgreSQL with `docker compose up -d`.
3. Run backend and frontend from their directories once scaffolds exist.
4. Add or update tests for behavior you change.
5. Update docs if architecture or behavior changes.

## Commit messages

Use Conventional Commits:

```text
feat:
fix:
docs:
test:
refactor:
chore:
security:
```

Example:

```text
docs: add initial project architecture
```

Do not use vague messages such as `update`, `changes`, `final`, or `stuff`.

## Pull requests / review focus

When reviewing or preparing changes, check:

- Assignment scope not expanded without reason
- Secrets not committed or logged
- Docs distinguish planned vs implemented
- Errors handled explicitly
- Tests cover important security/reliability behavior when applicable

## Security

See `SECURITY.md`. Never commit real credentials. Report suspected secret leaks immediately and rotate credentials.

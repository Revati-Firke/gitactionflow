# High-Level Architecture (HLA)

**Status:** Current — core product path including authenticated dashboard.

**Project:** GitActionFlow — event-driven automation for Git repositories.

Architecture source of truth. Do not redesign without an ADR and explicit agreement.

---

## System overview

```text
                React Dashboard
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
   Repository        Rules         Activity
        │              │              │
        └──────────────┼──────────────┘
                       ▼
                   Go Backend
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
      Auth          Webhooks       Worker
                       │              │
                       ▼              ▼
                  PostgreSQL ←── Actions / Events
                       │
          ┌────────────┴────────────┐
          ▼                         ▼
     GitHub API                   Slack
```

Canonical flow:

```text
GitHub webhook → verify → dedupe → persist event
 → worker → rules → action intents → executor (GitHub / Slack)
 → action results → dashboard reads PostgreSQL via API
```

---

## Main components

| Component | Responsibility |
| --- | --- |
| React dashboard | Login, repo, rules, event/action history |
| Auth | GitHub OAuth + sessions |
| Repository management | One connected repo |
| Webhook handler | Signature, dedupe, persist |
| Event worker | Claim, validate, rules, actions |
| Rule engine | Match → intents |
| Action executor | Persist + execute GitHub/Slack |

---

## Dashboard APIs

Authenticated, scoped to the user’s connected repository:

- `GET /api/events?page=&limit=` — summarized events (no raw payloads)
- `GET /api/actions?page=&limit=` — action history with rule name when available

Also: `/api/me`, repository routes, `/api/rules` CRUD.

---

## Why this shape

I kept a **modular monolith** so deploy and debugging stay simple on free tiers. Postgres is both store and queue—no Redis/Kafka for the assignment. Rules emit intents; a separate executor owns side effects and idempotency. Details: `docs/decisions/`.

---

## Related documents

- [API](../api/README.md) · [Database](../database/README.md) · [ADRs](../decisions/README.md) · [Local setup](../setup/LOCAL.md)

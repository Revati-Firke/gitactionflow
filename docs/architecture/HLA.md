# High-Level Architecture (HLA)

**Status:** Phase 9 — full core path including authenticated React dashboard.

**Project:** GitActionFlow — event-driven automation for Git repositories.

This document is the architecture source of truth. Do not redesign without an ADR and explicit agreement.

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

Canonical processing flow:

```text
GitHub webhook → verify → dedupe → persist event
 → worker → rules → action intents → executor (GitHub / Slack)
 → action results → dashboard reads PostgreSQL via API
```

---

## Main components

| Component | Responsibility |
| --- | --- |
| React dashboard | Login, repo, rules, event/action history (**Phase 9**) |
| Auth | GitHub OAuth + sessions (**done**) |
| Repository management | One connected repo (**done**) |
| Webhook handler | Signature, dedupe, persist (**done**) |
| Event worker / processor | Claim, validate, rules, actions (**done**) |
| Rule engine | Match → intents (**done**) |
| Action executor | Persist + execute GitHub/Slack (**done**) |

---

## Dashboard APIs (Phase 9)

Authenticated, scoped to the user’s connected repository:

- `GET /api/events?page=&limit=` — summarized events (no raw payloads)
- `GET /api/actions?page=&limit=` — action history with rule name when available

Existing: `/api/me`, repository routes, `/api/rules` CRUD.

---

## Related documents

- [API](../api/README.md) · [Database](../database/README.md) · [ADRs](../decisions/README.md) · [Local setup](../setup/LOCAL.md)

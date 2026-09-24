# High-Level Architecture (HLA)

**Status:** Phase 8 — auth, repo connection, webhook ingest, durable processing, rule evaluation, and GitHub/Slack action execution.

**Project:** GitActionFlow — event-driven automation for Git repositories.

This document is the architecture source of truth. Do not redesign without an ADR and explicit agreement.

---

## System overview

```text
GitHub
  ↓
Webhook Handler
  ↓
PostgreSQL (pending)
  ↓
Event Worker
  ↓
Event Processor
  ↓
Rule Engine
  ↓
Action Intents
  ↓
Action Executor
  ├── GitHub (label / comment)
  └── Slack (Incoming Webhook)
  ↓
Action Results (completed / retry / failed)
  ↓
Event processed | retry | failed
```

---

## Main components

| Component | Responsibility |
| --- | --- |
| Auth | GitHub OAuth + sessions (**done**) |
| Repository management | One connected repo (**done**) |
| Webhook handler | Signature, dedupe, persist (**done**) |
| Event worker | Claim / retry / fail (**done**) |
| Event processor | Validate + extract context + rules + actions (**done**) |
| Rule engine | Match rules → action intents (**done**) |
| Action executor | Persist actions, call GitHub/Slack (**Phase 8**) |
| Dashboard | Rules/events UI — **not yet** |

---

## Action execution (Phase 8)

- Intents from the rule engine become durable `actions` rows **before** any external call
- Idempotency key: `event_id:rule_id:action_type` (UNIQUE in PostgreSQL)
- External HTTP is **outside** DB transactions
- Event is marked `processed` only when all actions for that event are `completed`
- Permanent action failures → event `failed`; retryable → event stays in retry lifecycle
- See [ADR-008](../decisions/ADR-008-action-idempotency-and-failure-handling.md)

---

## Rule matching (Phase 7)

- Conditions: `event_type`, `keyword` (title/body), `author`, `required_labels`
- Semantics: **AND** within a rule; unspecified fields ignored
- Disabled rules skipped; multiple matches allowed (order: `created_at`, `id`)
- CRUD: `GET/POST/PUT/DELETE /api/rules` (session auth, scoped to connected repo)
- See [ADR-007](../decisions/ADR-007-rule-action-intent-separation.md)

---

## Event processing (Phase 6)

`pending` → `processing` → `processed` | retry | `failed`  
Claim: `FOR UPDATE SKIP LOCKED`. Lease recovery. [ADR-006](../decisions/ADR-006-durable-event-processing.md).

---

## Related documents

- [API](../api/README.md) · [Database](../database/README.md) · [ADRs](../decisions/README.md)

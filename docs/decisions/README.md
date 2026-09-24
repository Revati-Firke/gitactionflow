# Architecture Decision Records (ADR) Index

**Status:** Placeholders only in Phase 1. Full ADR write-ups will be added when decisions are locked during implementation.

ADRs capture **why** we chose an approach, not only what we built.

---

## Index

| ID | Title | Status |
| --- | --- | --- |
| ADR-001 | Modular Monolith | Accepted (placeholder) |
| ADR-002 | PostgreSQL as Source of Truth | Accepted (placeholder) |
| ADR-003 | Webhook Idempotency | Accepted (placeholder) |
| ADR-004 | AI Provider Abstraction | Proposed (placeholder; optional stretch) |
| [ADR-005](./ADR-005-sessions-and-token-encryption.md) | Server-side sessions and encrypted GitHub tokens | Accepted |
| [ADR-006](./ADR-006-durable-event-processing.md) | PostgreSQL-backed event worker, retries, stale lease recovery | Accepted |
| [ADR-007](./ADR-007-rule-action-intent-separation.md) | Rule matching produces action intents; no external side effects | Accepted |
| [ADR-008](./ADR-008-action-idempotency-and-failure-handling.md) | Persist actions, idempotency keys, retries; HTTP outside DB txns | Accepted |

---

## ADR-001 — Modular Monolith

**Context:** Assignment needs one coherent product (auth, webhooks, processing, dashboard APIs) with free-tier ops simplicity.

**Decision (direction):** Single Go deployable with clear internal packages; React SPA separate. No microservices.

**Consequences:** Simpler deploy and debugging; package boundaries must still isolate GitHub/Slack/DB.

_Full ADR text deferred._

---

## ADR-002 — PostgreSQL as Source of Truth

**Context:** Events and actions must survive process restarts; duplicates and failures must be queryable.

**Decision (direction):** PostgreSQL (Neon/Supabase in prod; Docker Compose locally) holds durable state. pgx for access.

**Consequences:** No in-memory queue as sole critical store; schema/migrations required in later phases.

_Full ADR text deferred._

---

## ADR-003 — Webhook Idempotency

**Context:** GitHub may deliver the same event more than once.

**Decision (direction):** Persist GitHub delivery IDs uniquely; skip re-execution of side effects for duplicates.

**Consequences:** Requires unique constraint / upsert strategy; actions tied to deliveries.

_Full ADR text deferred._

---

## ADR-004 — AI Provider Abstraction

**Context:** Optional stretch uses a free LLM (Gemini/Groq). Core bot must work without AI.

**Decision (direction):** If AI is added, isolate behind a small interface; feature-flag / no-op when unset.

**Consequences:** Core path never hard-depends on an AI key; provider can be swapped.

_Full ADR text deferred._

---

## Adding new ADRs

1. Add a row to the index.
2. Keep status honest (`Proposed`, `Accepted`, `Superseded`).
3. Prefer short context → decision → consequences.
4. Do not invent decisions that were not actually made.

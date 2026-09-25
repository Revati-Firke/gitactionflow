# Architecture Decision Records (ADR) Index

Short records of **why** a choice was made—not only what shipped.

| ID | Title | Status |
| --- | --- | --- |
| ADR-001 | Modular Monolith | Accepted |
| ADR-002 | PostgreSQL as Source of Truth | Accepted |
| ADR-003 | Webhook Idempotency | Accepted |
| ADR-004 | AI Provider Abstraction | Accepted (optional stretch) |
| [ADR-005](./ADR-005-sessions-and-token-encryption.md) | Server-side sessions and encrypted GitHub tokens | Accepted |
| [ADR-006](./ADR-006-durable-event-processing.md) | PostgreSQL-backed event worker, retries, stale lease recovery | Accepted |
| [ADR-007](./ADR-007-rule-action-intent-separation.md) | Rule matching produces action intents; no external side effects | Accepted |
| [ADR-008](./ADR-008-action-idempotency-and-failure-handling.md) | Persist actions, idempotency keys, retries; HTTP outside DB txns | Accepted |

---

## ADR-001 — Modular Monolith

**Context:** One coherent product (auth, webhooks, processing, dashboard) on free-tier hosts.

**Decision:** Single Go deployable with clear internal packages; React SPA separate. No microservices, Kafka, or Redis.

**Consequences:** Simpler deploy and debugging; package boundaries must still isolate GitHub/Slack/DB.

---

## ADR-002 — PostgreSQL as Source of Truth

**Context:** Events and actions must survive restarts; duplicates and failures must be queryable.

**Decision:** PostgreSQL (Neon in prod; Docker Compose locally) holds durable state. Access via pgx + migrations.

**Consequences:** No in-memory queue as the only critical store; schema discipline required.

---

## ADR-003 — Webhook Idempotency

**Context:** GitHub may deliver the same event more than once.

**Decision:** Persist GitHub delivery IDs with a unique constraint; skip re-execution of side effects for duplicates.

**Consequences:** Duplicate POSTs return success without a second logical event/action chain.

---

## ADR-004 — AI Provider Abstraction

**Context:** Optional stretch may use a free LLM (Gemini/Groq). Core bot must work without AI.

**Decision:** Isolate behind a small interface; `AI_ENABLED` defaults false; schema-validate model output; fail open (skip enrichment).

**Consequences:** Core label/comment/Slack never hard-depends on an AI key; provider can be swapped.

---

## Adding new ADRs

1. Add a row to the index.
2. Keep status honest (`Proposed`, `Accepted`, `Superseded`).
3. Prefer short context → decision → consequences.
4. Do not invent decisions that were not actually made.

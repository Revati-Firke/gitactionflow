# ADR-006 — Durable Event Processing (PostgreSQL Worker)

**Status:** Accepted  
**Date:** 2026-09-24

## Context

Webhook handlers persist deliveries as `pending` and return quickly. Events must be processed reliably without silent loss, without Redis, Kafka, or another broker (assignment / free-tier constraint).

## Decision

1. **PostgreSQL is the processing queue.** A background worker polls `webhook_events`, claims rows with `SELECT … FOR UPDATE SKIP LOCKED`, and updates status in the same database.
2. **Explicit lifecycle:** `pending` → `processing` → `processed` | (`pending` with backoff) | `failed`.
3. **Bounded retries** with simple backoff (1m / 5m / 15m). Exhausted events remain `failed` with `last_error` and `failed_at`.
4. **Stale lease recovery:** `processing` rows whose `locked_at` is older than `EVENT_PROCESSING_LEASE` can be reclaimed. Crash mid-process does not lose the event.
5. After validation, the worker runs the rule engine and action executor (see ADR-007 / ADR-008).

## Consequences

- Multiple workers are safe via row locks; no in-memory locks for durability.
- Webhook HTTP path stays unchanged (ack after durable insert).
- Operators inspect failures via SQL or the dashboard on `webhook_events` / `actions`.

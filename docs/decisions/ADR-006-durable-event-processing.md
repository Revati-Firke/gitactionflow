# ADR-006 — Durable Event Processing (PostgreSQL Worker)

**Status:** Accepted  
**Date:** 2026-09-24  
**Phase:** 6

## Context

Phase 5 persists GitHub webhook deliveries as `pending` and returns quickly. Events must be processed reliably without silent loss, without introducing Redis, Kafka, or another broker (assignment / free-tier constraint).

## Decision

1. **PostgreSQL is the processing queue.** A background worker polls `webhook_events`, claims rows with `SELECT … FOR UPDATE SKIP LOCKED`, and updates status in the same database.
2. **Explicit lifecycle:** `pending` → `processing` → `processed` | (`pending` with backoff) | `failed`.
3. **Bounded retries** with simple backoff (1m / 5m / 15m). Exhausted events remain `failed` with `last_error` and `failed_at`.
4. **Stale lease recovery:** `processing` rows whose `locked_at` is older than `EVENT_PROCESSING_LEASE` can be reclaimed (retry_count incremented). Crash mid-process does not lose the event.
5. **No rules/actions yet.** Successful processing means validated pipeline completion only.

## Consequences

- Multiple workers are safe via row locks; no in-memory locks for durability.
- Webhook HTTP path stays unchanged (ack after durable insert).
- Future rule engine plugs into `events.Processor` after validation.
- Operators inspect failures via SQL on `webhook_events`.

# ADR-008 — Action Idempotency and Failure Handling

**Status:** Accepted  
**Date:** 2026-09-24  
**Phase:** 8

## Context

Matched rules produce action intents (GitHub label, GitHub comment, Slack notification). External side effects must:

1. Survive process crashes (not lost in memory)
2. Avoid duplicate labels/comments/notifications when events are reprocessed
3. Retry transient failures without endlessly retrying permanent errors
4. Never hold a DB transaction open while waiting on GitHub or Slack
5. Propagate required action failures to the parent event lifecycle

Exactly-once delivery to external APIs cannot be guaranteed with a database alone (e.g. GitHub succeeds, then the process crashes before `completed` is written). The design **minimizes** duplicate side effects; it does not claim mathematical exactly-once external delivery.

## Decision

1. **Persist first.** Insert an `actions` row (`pending`) for each intent before any HTTP call. Unique `idempotency_key` = `event_id + ":" + rule_id + ":" + action_type`.
2. **Reuse on conflict.** Concurrent or replayed creates hit the UNIQUE constraint and reuse the existing row.
3. **Execute outside transactions.** Claim → HTTP → update status in separate short DB operations.
4. **Classify errors.** Transient (network, 429, 5xx) → schedule retry with shared exponential backoff. Permanent (401/403/404/422, bad config, Slack not configured) → `failed` immediately.
5. **Parent event coupling.** Processor returns incomplete/failed when actions are not all `completed`. Worker marks the event processed only on full success; otherwise retry or permanent fail per existing Phase 6 rules.
6. **Secrets.** Slack webhook URL and GitHub tokens come only from server config / encrypted user storage — never from rule config, logs, or API responses.

## Consequences

- Duplicate webhook deliveries still create one event (Phase 5) and at most one action per intent key.
- Re-applying the same GitHub label is generally safe; comments/Slack may duplicate only in the crash-after-success window.
- Failed action rows are retained for troubleshooting (dashboard later).
- `ACTION_MAX_RETRIES` bounds action attempts independently of `EVENT_MAX_RETRIES`.

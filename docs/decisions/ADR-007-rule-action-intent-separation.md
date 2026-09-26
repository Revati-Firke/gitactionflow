# ADR-007 — Rule Evaluation vs Action Execution

**Status:** Accepted  
**Date:** 2026-09-24

## Context

Configurable rules must decide what to do on a GitHub event, then GitHub/Slack must be called. Mixing “match” and “execute” in one step makes worker retries dangerous (duplicate labels/comments/Slack messages) and hard to test.

## Decision

1. The **rule engine** only evaluates conditions and returns **action intents** (rule id, action type, config, event id).
2. **No GitHub, Slack, or AI calls** happen during rule evaluation.
3. The **action executor** (`internal/actions`) consumes intents and performs side effects with its own idempotency (ADR-008).

## Consequences

- Rule matching is deterministic and unit-testable without network.
- Retries can re-run matching safely; duplicate side effects are constrained by action idempotency keys.
- Clear package boundary: `internal/rules` vs `internal/actions`.

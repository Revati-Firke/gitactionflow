# ADR-007 — Rule Evaluation vs Action Execution

**Status:** Accepted  
**Date:** 2026-09-24  
**Phase:** 7

## Context

GitActionFlow must apply configurable rules to GitHub events, then later call GitHub/Slack. Mixing “match” and “execute” in one step makes retries from Phase 6 dangerous (duplicate labels/comments/Slack messages) and hard to test.

## Decision

1. The **rule engine** only evaluates conditions and returns **action intents** (rule id, action type, config, event id).
2. **No GitHub, Slack, or AI calls** happen during rule evaluation.
3. Phase 8 (and later) will consume intents and perform side effects with their own idempotency.

## Consequences

- Rule matching is deterministic and unit-testable without network.
- Retries re-run matching safely in Phase 7 (no external effects yet).
- Clear package boundary: `internal/rules` vs future action executors.

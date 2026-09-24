# AI Notes

Honest record of how AI tools were used on GitActionFlow.

## AI Tools Used

- Cursor agent (Composer) for Phase 6 implementation assistance

## How AI Was Used

- Scaffolded migration `000005`, store claim/retry APIs, `internal/events` worker + processor, config knobs, tests, and docs updates aligned to the Phase 6 prompt.

## Engineering Decisions Made by Me

- ADR number **006** (ADR-005 already used for sessions)
- Permanent validation errors fail immediately (no useless retries); transient errors use backoff
- Keep SQL claim logic in `store`; `events` package owns processor + worker
- Successful Phase 6 processing = validated pipeline only (no rules/actions)

## Incorrect AI Suggestion / Hardest AI Mistake

- Early claim tests failed because leftover `pending` rows from live Phase 5 testing were claimed instead of the newly inserted row; concurrent claim also saw multiple pending rows.

## How I Corrected It

- Integration tests clear claimable rows before asserting; claim SQL remains oldest-first for production fairness.

## What I Would Improve

- Optional metrics/admin API for failed events (later dashboard phase)
- Action-level idempotency when GitHub/Slack side effects land

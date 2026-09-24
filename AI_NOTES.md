# AI Notes

Honest record of how AI tools were used on GitActionFlow.

## AI Tools Used

- Cursor agent (Composer) for Phase 7 rule engine and Phase 8 action execution

## How AI Was Used

- Phase 7: `rules` migration/store, matching engine, CRUD APIs, processor integration (intents), tests, ADR-007.
- Phase 8: `actions` migration/store, GitHub write methods, Slack Incoming Webhook client, action executor (persist → execute → results), processor/worker wiring, unit tests with mocks, ADR-008 and docs.

## Engineering Decisions Made by Me

- ADR **007** for rule/intent separation; ADR **008** for action idempotency and failure handling
- Idempotency key format: `event_id:rule_id:action_type`
- Action retries reuse `events.RetryBackoff`; `ACTION_MAX_RETRIES` defaults to 3
- Parent event incomplete while actions pending; permanent action failure fails the event
- Empty keyword/author after trim → unspecified (NULL)
- Slack URL only from env; rule config holds message text only

## Incorrect AI Suggestion / Hardest AI Mistake

- Prompt referenced “ADR-006” for rules; repository already had ADR-006 for event processing — used ADR-007 instead.
- Mid Phase 8, putting `actions.ErrFailed` in `events.IsPermanent` would create an import cycle (`actions` already imports `events`) — errors live in `events` (`ErrActionsFailed` / `ErrActionsIncomplete`).

## How I Corrected It

- Documented ADR-007/008 and linked from the decisions index.
- Defined action outcome errors in the `events` package so the worker can classify without cycles.

## What I Would Improve

- Optional dashboard UI for rules/events/actions (Phase 9+)
- Narrower crash-window mitigation for comment/Slack duplicates (e.g. outbound request ledger) if assignment scope allowed

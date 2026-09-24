# AI Notes

Honest record of how AI tools were used on GitActionFlow. Fill in as development proceeds. Do not invent usage or mistakes that did not happen.

## AI Tools Used

Cursor agent (Composer) for Phase 5 webhook ingestion assistance.

## How AI Was Used

Implemented signature verification, webhook handler, delivery-id idempotency, and pending persistence.

## Engineering Decisions Made by Me

- Ack only after durable persist
- Ignore unknown repos with 200 to avoid GitHub retry storms
- No processing in the webhook request path

## Incorrect AI Suggestion / Hardest AI Mistake

_To be documented honestly during development._

## How I Corrected It

_To be completed during development._

## What I Would Improve

_To be completed before submission._

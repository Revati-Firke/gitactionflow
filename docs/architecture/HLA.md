# High-Level Architecture (HLA)

**Status:** Phase 6 — auth, one-repo connection, webhook ingestion, and durable event processing (no rules/actions yet).

**Project:** GitActionFlow — event-driven automation for Git repositories.

This document is the architecture source of truth. Do not redesign without an ADR and explicit agreement.

---

## System overview

GitActionFlow is a **modular monolith**:

- A **React** dashboard for authenticated users
- A **Go** HTTP backend for OAuth, repository connection, webhooks, processing, and dashboard APIs
- **PostgreSQL** as the durable source of truth (including the event processing queue)
- External systems: **GitHub** (OAuth, webhooks, REST API) and **Slack** (Incoming Webhook)
- Optional later: free-tier **AI** provider for summaries / suggestions (never required for core path)

```text
GitHub
  ↓
Webhook Handler
  ↓
PostgreSQL (pending)
  ↓
Event Worker
  ↓
Event Processor
  ↓
Processed / Retry / Failed
```

---

## Main components

| Component | Responsibility |
| --- | --- |
| React Web Dashboard | Login-gated UI: connected repo, rules, event/action history |
| Auth module | GitHub OAuth start/callback, session establishment, CSRF `state` (**implemented**) |
| Repository management | Connect one owned/admin repo (**Phase 4**; webhook registration later) |
| Dashboard APIs | Read models for events, actions, rules, connection status |
| Webhook handler | Signature verify, validate, dedupe, durable persist (**Phase 5**) |
| Event worker | Poll/claim pending and stale processing rows; retries (**Phase 6**) |
| Event processor | Validate persisted events; future rules/actions plug in here |
| Rule engine | Match configured rules — **not yet** |
| Actions | GitHub label/comment; Slack notify — **not yet** |
| Optional AI | Stretch only; never block core path |
| PostgreSQL | Users, sessions, repos, events, failures |

---

## Webhook ingestion (Phase 5)

```text
GitHub → POST /webhooks/github → verify HMAC → validate → UNIQUE delivery_id → pending → 2xx
```

## Event processing (Phase 6)

```text
pending → processing → processed
                 ↘ pending (+ next_retry_at)
                 ↘ failed
```

- Claim with `FOR UPDATE SKIP LOCKED`
- Stale `locked_at` reclaimed after `EVENT_PROCESSING_LEASE`
- Backoff: 1m / 5m / 15m; bound by `EVENT_MAX_RETRIES`
- No Redis/Kafka — see [ADR-006](../decisions/ADR-006-durable-event-processing.md)

Successful processing in Phase 6 means **validated pipeline**, not GitHub/Slack actions.

---

## Authentication flow (Phase 3)

See [ADR-005](../decisions/ADR-005-sessions-and-token-encryption.md).

---

## Reliability

- Delivery-ID ingest idempotency
- Persist before acknowledge
- Visible retries / failures in PostgreSQL
- Graceful worker shutdown via context cancel

## Explicit non-goals

- Microservices, Kafka/Redis as primary durability, non-GitHub VCS

## Related documents

- [API](../api/README.md) · [Database](../database/README.md) · [Deployment](../deployment/README.md) · [ADRs](../decisions/README.md)

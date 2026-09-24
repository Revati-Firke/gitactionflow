# API Plan

**Status:** Planned — **not implemented** in Phase 1.

This document describes the intended HTTP API surface for the Go backend. Paths, payloads, and status codes will be refined when handlers are built. Treat names below as design direction, not a frozen contract.

Base URL (planned): `/api` (or similar). All dashboard routes require an authenticated session unless noted.

---

## Planned categories

```text
Authentication
Repositories
Rules
Webhooks
Events
Actions
Dashboard
Health
```

---

## Authentication (planned)

| Concern | Direction |
| --- | --- |
| Start OAuth | Redirect user to GitHub authorize URL with `state` |
| OAuth callback | Exchange code, validate `state`, create session |
| Logout | Invalidate session |
| Current user | Return authenticated profile (no secrets) |

---

## Repositories (planned)

| Concern | Direction |
| --- | --- |
| List owned repos | Via GitHub API using stored user token (server-side) |
| Connect repository | Persist one connected repo; configure webhook |
| Connected status | Return current connection metadata (no secrets) |
| Disconnect | Remove connection / disable webhook (later) |

Assignment core: **one** connected repository per user is sufficient.

---

## Rules (planned)

| Concern | Direction |
| --- | --- |
| List rules | Rules for the connected repository |
| Create / update / delete | Simple matchers (e.g. title contains keyword → label + Slack) |
| Enable / disable | Soft control without deleting history |

---

## Webhooks (planned)

| Concern | Direction |
| --- | --- |
| Ingest endpoint | Public `POST` for GitHub deliveries |
| Auth model | `X-Hub-Signature-256` — not session cookies |
| Events | At least `issues` and `pull_request` |
| Response | Success after durable persistence of the delivery |

Not a dashboard CRUD API — a system integration endpoint.

---

## Events (planned)

| Concern | Direction |
| --- | --- |
| List events | Paginated history for the connected repo |
| Event detail | Payload summary, delivery ID, processing status |

---

## Actions (planned)

| Concern | Direction |
| --- | --- |
| List actions | Actions taken (GitHub label/comment, Slack notify) |
| Action detail | Status, error message if failed, timestamps |
| Retry (optional later) | Re-attempt failed actions when safe |

---

## Dashboard (planned)

| Concern | Direction |
| --- | --- |
| Summary | Connected repo, recent events, recent actions, rule count |
| May compose | Thin aggregation over Events + Actions + Repositories + Rules |

---

## Health (planned)

| Concern | Direction |
| --- | --- |
| Liveness | Process up |
| Readiness | Database reachable (and other critical deps if any) |

Health endpoints are typically unauthenticated and must not leak secrets or internal details.

---

## Cross-cutting (planned)

- JSON request/response
- Consistent error shape
- No secret fields in responses
- CSRF protection via OAuth `state` and session cookie practices appropriate to the chosen frontend hosting model

---

## Implementation note

Do not implement these routes in Phase 1. When implementing, update this document to mark endpoints as **implemented** vs **planned**.

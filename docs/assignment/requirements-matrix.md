# Assignment requirements matrix

Statuses use exactly: **COMPLETE** | **PARTIAL** | **NOT IMPLEMENTED**

Live:

| Role | URL |
| --- | --- |
| Frontend | https://gitactionflow.vercel.app |
| Backend | https://gitactionflow-backend.onrender.com |
| Source | https://github.com/Revati-Firke/gitactionflow |

---

## Core requirements

| Requirement | Implementation | Verification | Evidence | Status |
| --- | --- | --- | --- | --- |
| Public deployed app | Neon + Render + Vercel | HTTPS load + health | Live 200 | **COMPLETE** |
| GitHub OAuth | OAuth App, state, HttpOnly session | Live login | Dashboard session | **COMPLETE** |
| Connect one repository | Admin check, unique per user | Live UI | `Daily-Impression-AI-Model` | **COMPLETE** |
| GitHub webhook | HMAC + delivery ID + persist | Live events + forged 401 | Dashboard + curl forged | **COMPLETE** |
| Issues event | Ingest + worker | Live issue → event/Slack | Observed | **COMPLETE** |
| Pull request event | Same pipeline, `pull_request` | Code + tests; live PR once | Owner smoke | **PARTIAL** |
| Bot GitHub label | `github_label` action | Live needs existing label | 422 if missing | **PARTIAL** |
| Bot GitHub comment | `github_comment` action | Unit tests; live optional | Code | **PARTIAL** |
| Slack notification | Incoming Webhook env | Live Slack messages | `#gitactionflow-demo` | **COMPLETE** |
| Authenticated dashboard | React + cookies | Live | UI | **COMPLETE** |
| Event/action history | `/api/events`, `/api/actions` | Live | Lists + failures | **COMPLETE** |
| Configurable rules | CRUD + AND conditions | Live | Rules UI | **COMPLETE** |
| README / .env.example | Docs | Review | Repo | **COMPLETE** |
| Deployment instructions | `docs/deployment/` | Used for deploy | Repo | **COMPLETE** |
| AGENTS.md / AI_NOTES | Present | Review | Repo | **COMPLETE** |

## Quality requirements

| Requirement | Implementation | Verification | Evidence | Status |
| --- | --- | --- | --- | --- |
| Forged webhook rejection | HMAC `hmac.Equal` | Live POST bad sig → 401 | curl | **COMPLETE** |
| Duplicate delivery | `delivery_id` UNIQUE | Unit tests; live redeliver once | Tests | **PARTIAL** |
| Idempotent actions | `event:rule:type` UNIQUE | Unit tests | ADR-008 | **COMPLETE** |
| Downstream failure persistence | action status / last_error | Live failed label 422 visible | Dashboard | **COMPLETE** |
| Retries | Event + action backoff | Unit tests | Worker/executor | **COMPLETE** |
| No silent event loss | Persist before ack | Design + tests | Webhook 5xx on DB fail | **COMPLETE** |
| Secret protection | Redacted logs; no FE secrets | Bundle scan | Build scan | **COMPLETE** |
| Authz isolation | Session-scoped queries | Code review + tests | Handlers | **COMPLETE** |

## Stretch / optional

| Requirement | Status | Notes |
| --- | --- | --- |
| Configurable rules | **COMPLETE** | Core |
| AI assistance | **COMPLETE** | Optional; `AI_ENABLED` default false |
| Observability | **PARTIAL** | slog + request_id; no APM |
| Failure history | **COMPLETE** | Dashboard |
| Retries | **COMPLETE** | Core |
| Multi-repository | **NOT IMPLEMENTED** | One repo by design |
| GitHub App auth | **NOT IMPLEMENTED** | OAuth App kept |

## Rule semantics

- Conditions within a rule are **AND**.
- Unspecified optional conditions do not restrict.
- Disabled rules never match.
- Multiple matching rules may all fire; same event+rule+action type is idempotent.
- Event is **Failed** if any required action permanently fails (Slack may still succeed).

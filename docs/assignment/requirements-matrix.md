# Assignment requirements matrix

Statuses: **COMPLETE** | **PARTIAL** | **NOT IMPLEMENTED**

| Role | URL |
| --- | --- |
| Frontend | https://gitactionflow.vercel.app |
| Backend | https://gitactionflow-backend.onrender.com |
| Source | https://github.com/Revati-Firke/gitactionflow |

## Design choices that shaped the matrix

- **One repo per user** — meets the brief without multi-tenant product work.
- **OAuth App, not GitHub App** — enough for connect + writes on a take-home timeline.
- **Manual webhook** — durable signed ingest first; auto-register later if time allows.
- **AND rule conditions** — simple, predictable matching for demos and tests.
- **Optional AI off by default** — must not own label/comment/Slack.

---

## Core

| Requirement | Implementation | Verification | Evidence | Status |
| --- | --- | --- | --- | --- |
| Public deployed app | Neon + Render + Vercel | HTTPS + health | Live 200 | **COMPLETE** |
| GitHub OAuth | OAuth App, state, HttpOnly session | Live login | Dashboard | **COMPLETE** |
| Connect one repository | Admin check, unique per user | Live UI | Demo repo connected | **COMPLETE** |
| GitHub webhook | HMAC + delivery ID + persist | Live events + forged 401 | Dashboard + curl | **COMPLETE** |
| Issues event | Ingest + worker | Live issue → event/Slack | Observed | **COMPLETE** |
| Pull request event | Same pipeline | Code + tests; live PR once | Owner smoke | **PARTIAL** |
| Bot GitHub label | `github_label` | Needs existing label | 422 if missing | **PARTIAL** |
| Bot GitHub comment | `github_comment` | Unit tests; live optional | Code | **PARTIAL** |
| Slack notification | Incoming Webhook env | Live messages | `#gitactionflow-demo` | **COMPLETE** |
| Authenticated dashboard | React + cookies | Live | UI | **COMPLETE** |
| Event/action history | `/api/events`, `/api/actions` | Live | Lists + failures | **COMPLETE** |
| Configurable rules | CRUD + AND conditions | Live | Rules UI | **COMPLETE** |
| README / .env.example | Docs | Review | Repo | **COMPLETE** |
| Deployment instructions | `docs/deployment/` | Used for deploy | Repo | **COMPLETE** |
| AGENTS.md / AI_NOTES | Present | Review | Repo | **COMPLETE** |

## Quality

| Requirement | Implementation | Verification | Evidence | Status |
| --- | --- | --- | --- | --- |
| Forged webhook rejection | HMAC `hmac.Equal` | Live bad sig → 401 | curl | **COMPLETE** |
| Duplicate delivery | `delivery_id` UNIQUE | Unit tests; live redeliver once | Tests | **PARTIAL** |
| Idempotent actions | `event:rule:type` UNIQUE | Unit tests | ADR-008 | **COMPLETE** |
| Downstream failure persistence | action status / last_error | Live 422 visible | Dashboard | **COMPLETE** |
| Retries | Event + action backoff | Unit tests | Worker/executor | **COMPLETE** |
| No silent event loss | Persist before ack | Design + tests | Webhook 5xx on DB fail | **COMPLETE** |
| Secret protection | Redacted logs; no FE secrets | Bundle scan | Build scan | **COMPLETE** |
| Authz isolation | Session-scoped queries | Code + tests | Handlers | **COMPLETE** |

## Stretch

| Requirement | Status | Notes |
| --- | --- | --- |
| AI assistance | **COMPLETE** | Optional; default off |
| Observability | **PARTIAL** | slog + request_id; no APM |
| Multi-repository | **NOT IMPLEMENTED** | By design |
| GitHub App auth | **NOT IMPLEMENTED** | OAuth App kept |

## Rule semantics

- Conditions within a rule are **AND**.
- Unspecified optional conditions do not restrict.
- Disabled rules never match.
- Multiple matching rules may all fire; same event+rule+action type is idempotent.
- Event is **Failed** if any required action permanently fails (Slack may still succeed).

# AGENTS.md

Notes for anyone (or any tool) changing this repo. I wrote these to keep the Abstrabit take-home **small, secure, and demoable**.

## Product

GitHub OAuth → one connected repo → signed webhooks → rules → GitHub label/comment + Slack → dashboard history.

Optimize for a working free-tier demo. Skip platform features.

## My locked decisions

| Choice | Why |
| --- | --- |
| Modular monolith (Go + React + Postgres) | One deploy unit on Render |
| Neon + Render + Vercel | Free, no card |
| OAuth App + one repo | Matches the brief; skip GitHub App / multi-repo |
| Postgres queue (`SKIP LOCKED`) | No Redis/Kafka |
| Rule intents ≠ executor | Safe retries; testable matching |
| Persist before external HTTP | No silent loss |
| Manual repo webhook | Ingest correctness over auto-register |
| Vercel `/api` `/auth` proxy | First-party cookies after Incognito broke cross-site |
| `AI_ENABLED` default off | Stretch only |

Details: `docs/decisions/` · `AI_NOTES.md`

## Flow (don’t redesign quietly)

```text
webhook → HMAC → dedupe → persist → worker → rules → GitHub/Slack → dashboard
```

## Hard rules

- HttpOnly sessions; encrypted tokens; never trust client `user_id`
- Re-fetch repo by id; require `admin`
- HMAC on **raw** body; unique delivery ID; no side effects inside the webhook request
- Rule conditions AND; Slack URL only from env
- Retry 429/5xx; fail permanently on 401/403/404/422
- `VITE_API_BASE_URL` unset in production

## Out of scope

Multi-repo product, Redis/Kafka/K8s, paid APIs, workflow builders.

## Habits

Small diffs. Fail fast on config. Never log secrets. Tests for signature, idempotency, rules. Conventional Commits. Docs must match what runs.

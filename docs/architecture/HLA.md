# High-level architecture

One Go API + React UI + PostgreSQL.

```text
React dashboard ──► Go backend ──► Postgres
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
        OAuth      Webhooks      Worker
                       │            │
                       └──── rules + actions ──► GitHub API / Slack
```

```text
GitHub → verify HMAC → dedupe delivery ID → persist event
      → worker (SKIP LOCKED) → rules → executor → results → dashboard
```

| Piece | Role |
| --- | --- |
| Auth | OAuth App + HttpOnly session |
| Repository | One connected repo (admin) |
| Webhook | Signature, idempotency, persist only |
| Worker | Claim, retry, stale lease recovery |
| Rules | AND match → action intents |
| Executor | Persist action row, then GitHub/Slack |

Why this shape: free-tier ops, durable events, clear boundaries for tests. Decisions: `docs/decisions/`.

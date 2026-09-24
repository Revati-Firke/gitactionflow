# Security

Security principles for GitActionFlow. **Phase 1 documents these rules; runtime enforcement lands in later phases.**

## Reporting

If you discover a vulnerability or a committed secret in this repository, do not open a public issue with the secret contents. Rotate credentials immediately and notify the repository owner.

## Secrets

Never expose the following to the frontend, client bundles, public repos, or logs:

- GitHub OAuth client secret
- GitHub user / installation access tokens
- GitHub webhook secrets
- Slack Incoming Webhook URLs
- AI API keys (if used)
- Database credentials and connection strings with passwords

Rules:

- Use environment variables / host secret stores.
- Commit only `.env.example` with placeholders.
- Never commit `.env` or key files.
- Redact sensitive values in structured logs.

## GitHub webhooks (planned)

Incoming webhooks must be verified using:

```text
X-Hub-Signature-256
```

- Compute HMAC-SHA256 over the raw request body with the webhook secret.
- Compare signatures with a **constant-time** comparison.
- Reject requests with missing or invalid signatures.

## Replay and duplicate protection (planned)

- Persist GitHub `X-GitHub-Delivery` (delivery ID) as an idempotency key.
- A duplicate delivery must not cause duplicate GitHub actions or Slack notifications.
- Side effects should be gated on unique delivery / action records in PostgreSQL.

## OAuth (planned)

- Generate a cryptographically random `state` value.
- Store it server-side (session or short-lived store) and validate on callback.
- Reject callbacks with missing or mismatched `state` (CSRF protection).
- Prefer HTTPS callback URLs on the public deployment.

## Reliability and abuse resistance (planned)

- Validate event payloads and event types before processing.
- Persist first; then process — so brief downstream outages do not silently lose events.
- Make failures visible (action status / failure records) and retryable.

## Dependency and supply chain

- Prefer well-maintained, minimal dependencies.
- Do not vendor credentials in Docker images or CI logs.

## What Phase 1 does not claim

Signature verification, OAuth state validation, and idempotency are **not implemented yet**. This document states the target bar for later phases.

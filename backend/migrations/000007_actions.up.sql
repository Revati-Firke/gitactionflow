-- Phase 8: durable action execution records (GitHub / Slack).

CREATE TABLE actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES webhook_events (id) ON DELETE CASCADE,
    rule_id UUID REFERENCES rules (id) ON DELETE SET NULL,
    action_type TEXT NOT NULL,
    action_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    idempotency_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    last_error TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT actions_idempotency_key_unique UNIQUE (idempotency_key),
    CONSTRAINT actions_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT actions_action_type_check CHECK (action_type IN ('github_label', 'github_comment', 'slack_notification')),
    CONSTRAINT actions_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT actions_max_attempts_check CHECK (max_attempts >= 0)
);

CREATE INDEX actions_event_id_idx ON actions (event_id);
CREATE INDEX actions_status_idx ON actions (status);
CREATE INDEX actions_claim_pending_idx ON actions (status, next_retry_at)
    WHERE status = 'pending';

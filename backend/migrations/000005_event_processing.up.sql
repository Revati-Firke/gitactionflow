-- Phase 6: durable event processing fields (retries, lease, failure visibility).

ALTER TABLE webhook_events
    ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN max_retries INTEGER NOT NULL DEFAULT 3,
    ADD COLUMN next_retry_at TIMESTAMPTZ,
    ADD COLUMN last_error TEXT,
    ADD COLUMN locked_at TIMESTAMPTZ,
    ADD COLUMN failed_at TIMESTAMPTZ;

ALTER TABLE webhook_events
    ADD CONSTRAINT webhook_events_retry_count_check CHECK (retry_count >= 0),
    ADD CONSTRAINT webhook_events_max_retries_check CHECK (max_retries >= 0);

-- Worker claim / retry scheduling lookups.
CREATE INDEX webhook_events_claim_pending_idx
    ON webhook_events (status, next_retry_at, created_at)
    WHERE status = 'pending';

CREATE INDEX webhook_events_claim_processing_idx
    ON webhook_events (status, locked_at)
    WHERE status = 'processing';

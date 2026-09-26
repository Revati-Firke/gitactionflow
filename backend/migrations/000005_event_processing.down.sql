DROP INDEX IF EXISTS webhook_events_claim_processing_idx;
DROP INDEX IF EXISTS webhook_events_claim_pending_idx;

ALTER TABLE webhook_events
    DROP CONSTRAINT IF EXISTS webhook_events_max_retries_check,
    DROP CONSTRAINT IF EXISTS webhook_events_retry_count_check;

ALTER TABLE webhook_events
    DROP COLUMN IF EXISTS failed_at,
    DROP COLUMN IF EXISTS locked_at,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS next_retry_at,
    DROP COLUMN IF EXISTS max_retries,
    DROP COLUMN IF EXISTS retry_count;

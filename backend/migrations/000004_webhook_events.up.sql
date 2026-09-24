-- Phase 5: durable GitHub webhook event ingestion (pending only; no processing yet).

CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories (id) ON DELETE CASCADE,
    delivery_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    action TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT webhook_events_delivery_id_unique UNIQUE (delivery_id),
    CONSTRAINT webhook_events_status_check CHECK (status IN ('pending', 'processing', 'processed', 'failed'))
);

CREATE INDEX webhook_events_repository_id_idx ON webhook_events (repository_id);
CREATE INDEX webhook_events_status_idx ON webhook_events (status);
CREATE INDEX webhook_events_received_at_idx ON webhook_events (received_at DESC);

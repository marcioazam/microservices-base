-- V3: Create outbox events table for reliable event publishing
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE INDEX idx_outbox_unprocessed ON outbox_events(created_at) WHERE processed_at IS NULL;
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_type, aggregate_id);
CREATE INDEX idx_outbox_event_type ON outbox_events(event_type);

COMMENT ON TABLE outbox_events IS 'Transactional outbox for reliable event publishing';
COMMENT ON COLUMN outbox_events.aggregate_type IS 'Type of aggregate (e.g., User)';
COMMENT ON COLUMN outbox_events.event_type IS 'Event type (e.g., UserRegistered, EmailVerificationRequested)';
COMMENT ON COLUMN outbox_events.payload_json IS 'JSON payload of the event';
COMMENT ON COLUMN outbox_events.processed_at IS 'Timestamp when event was published, NULL if pending';

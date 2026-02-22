-- Outbound Webhooks
-- Stores webhook endpoints, subscribed event types, and delivery logs

-- Webhook endpoints table
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    secret_encrypted BYTEA NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    event_types JSONB NOT NULL DEFAULT '[]',
    headers JSONB DEFAULT '{}',
    retry_count INTEGER NOT NULL DEFAULT 3,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Webhook deliveries table (delivery log)
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_id UUID,
    payload JSONB NOT NULL,
    request_headers JSONB,
    response_status INTEGER,
    response_body TEXT,
    response_headers JSONB,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    delivered_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for webhook_endpoints
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_org_id ON webhook_endpoints(org_id);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_enabled ON webhook_endpoints(org_id, enabled);

-- Indexes for webhook_deliveries
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_org_id ON webhook_deliveries(org_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint_id ON webhook_deliveries(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_next_retry ON webhook_deliveries(next_retry_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event_type ON webhook_deliveries(event_type);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON webhook_deliveries(created_at DESC);

-- Unique constraint for endpoint names per org
CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_endpoints_org_name ON webhook_endpoints(org_id, name);

-- Status check constraint
ALTER TABLE webhook_deliveries ADD CONSTRAINT chk_webhook_delivery_status
    CHECK (status IN ('pending', 'delivered', 'failed', 'retrying'));

-- Comments
COMMENT ON TABLE webhook_endpoints IS 'Outbound webhook endpoints for event notifications';
COMMENT ON COLUMN webhook_endpoints.secret_encrypted IS 'HMAC secret for signing payloads (encrypted)';
COMMENT ON COLUMN webhook_endpoints.event_types IS 'JSON array of subscribed event types';
COMMENT ON COLUMN webhook_endpoints.headers IS 'Custom HTTP headers to include in requests';
COMMENT ON TABLE webhook_deliveries IS 'Log of all webhook delivery attempts';
COMMENT ON COLUMN webhook_deliveries.payload IS 'The JSON payload sent to the webhook';
COMMENT ON COLUMN webhook_deliveries.next_retry_at IS 'When to attempt next retry (for exponential backoff)';

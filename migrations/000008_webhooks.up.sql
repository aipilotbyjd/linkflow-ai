-- ============================================================================
-- Migration: 000008_webhooks
-- Description: Webhooks and logs tables
-- ============================================================================

CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    endpoint_url VARCHAR(500) NOT NULL UNIQUE,
    path VARCHAR(255),
    secret VARCHAR(255),
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers JSONB DEFAULT '{}',
    authentication_type VARCHAR(50),
    authentication_config JSONB DEFAULT '{}',
    headers_validation JSONB,
    body_validation JSONB,
    retry_config JSONB DEFAULT '{"maxRetries": 3, "retryDelay": 1000}',
    timeout_ms INTEGER DEFAULT 30000,
    rate_limit INTEGER DEFAULT 100,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_triggered_at TIMESTAMPTZ,
    trigger_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX idx_webhooks_workspace_id ON webhooks(workspace_id);
CREATE INDEX idx_webhooks_workflow_id ON webhooks(workflow_id);
CREATE INDEX idx_webhooks_endpoint_url ON webhooks(endpoint_url);
CREATE INDEX idx_webhooks_status ON webhooks(status);

CREATE TABLE webhook_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES executions(id) ON DELETE SET NULL,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500),
    headers JSONB,
    body TEXT,
    query_params JSONB,
    response_status INTEGER,
    response_body TEXT,
    ip_address INET,
    duration_ms INTEGER,
    error TEXT,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_logs_webhook_id ON webhook_logs(webhook_id);
CREATE INDEX idx_webhook_logs_execution_id ON webhook_logs(execution_id);
CREATE INDEX idx_webhook_logs_processed_at ON webhook_logs(processed_at DESC);

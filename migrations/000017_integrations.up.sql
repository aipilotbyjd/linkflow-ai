-- ============================================================================
-- Migration: 000017_integrations
-- Description: Third-party integrations
-- ============================================================================

CREATE TABLE integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    integration_type VARCHAR(100) NOT NULL,
    provider VARCHAR(100),
    config JSONB DEFAULT '{}',
    credentials_id UUID REFERENCES credentials(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    last_sync_at TIMESTAMPTZ,
    sync_frequency_minutes INTEGER,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_integrations_user_id ON integrations(user_id);
CREATE INDEX idx_integrations_workspace_id ON integrations(workspace_id);
CREATE INDEX idx_integrations_type ON integrations(integration_type);
CREATE INDEX idx_integrations_provider ON integrations(provider);
CREATE INDEX idx_integrations_status ON integrations(status);

CREATE TABLE integration_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    request_data JSONB,
    response_data JSONB,
    error TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_integration_logs_integration_id ON integration_logs(integration_id);
CREATE INDEX idx_integration_logs_action ON integration_logs(action);
CREATE INDEX idx_integration_logs_created_at ON integration_logs(created_at DESC);

CREATE TABLE integration_webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    endpoint_url TEXT NOT NULL,
    secret VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_integration_webhooks_integration_id ON integration_webhooks(integration_id);
CREATE INDEX idx_integration_webhooks_event_type ON integration_webhooks(event_type);

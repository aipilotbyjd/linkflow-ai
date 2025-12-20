-- ============================================================================
-- Migration: 000012_tenants
-- Description: Tenants, limits, and features
-- ============================================================================

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    trial_ends_at TIMESTAMPTZ,
    subscription_id VARCHAR(255),
    owner_id UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_plan ON tenants(plan);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_owner_id ON tenants(owner_id);

CREATE TABLE tenant_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE UNIQUE,
    max_workflows INTEGER DEFAULT 10,
    max_executions_per_day INTEGER DEFAULT 1000,
    max_nodes_per_workflow INTEGER DEFAULT 50,
    max_credentials INTEGER DEFAULT 20,
    max_team_members INTEGER DEFAULT 5,
    max_storage_mb INTEGER DEFAULT 1000,
    max_api_calls_per_minute INTEGER DEFAULT 100,
    max_webhooks INTEGER DEFAULT 10,
    max_schedules INTEGER DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_limits_tenant_id ON tenant_limits(tenant_id);

CREATE TABLE tenant_features (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    feature_name VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    config JSONB DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, feature_name)
);

CREATE INDEX idx_tenant_features_tenant_id ON tenant_features(tenant_id);
CREATE INDEX idx_tenant_features_feature_name ON tenant_features(feature_name);

CREATE TABLE resource_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    resource_type VARCHAR(50) NOT NULL,
    quantity BIGINT NOT NULL DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, workspace_id, resource_type, period_start)
);

CREATE INDEX idx_resource_usage_tenant_id ON resource_usage(tenant_id);
CREATE INDEX idx_resource_usage_workspace_id ON resource_usage(workspace_id);
CREATE INDEX idx_resource_usage_period ON resource_usage(period_start, period_end);

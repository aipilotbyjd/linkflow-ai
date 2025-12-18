-- Migration: Credentials and Tenants
-- Services: credential, tenant, executor

-- Credentials Service Schema
CREATE TABLE IF NOT EXISTS credentials (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- api_key, oauth2, basic, bearer, custom
    service VARCHAR(100), -- github, slack, aws, etc.
    encrypted_data TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active',
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_credentials_org ON credentials(organization_id);
CREATE INDEX idx_credentials_type ON credentials(type);
CREATE INDEX idx_credentials_service ON credentials(service);
CREATE INDEX idx_credentials_status ON credentials(status);

-- OAuth2 Tokens (for credential service)
CREATE TABLE IF NOT EXISTS oauth2_tokens (
    id VARCHAR(255) PRIMARY KEY,
    credential_id VARCHAR(255) NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(50) DEFAULT 'Bearer',
    scope TEXT,
    expires_at TIMESTAMP NOT NULL,
    refresh_expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_oauth2_tokens_credential ON oauth2_tokens(credential_id);
CREATE INDEX idx_oauth2_tokens_expires ON oauth2_tokens(expires_at);

-- Variables (environment variables, secrets)
CREATE TABLE IF NOT EXISTS variables (
    id VARCHAR(255) PRIMARY KEY,
    scope VARCHAR(50) NOT NULL, -- global, organization, workflow
    scope_id VARCHAR(255), -- org_id or workflow_id
    name VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    is_secret BOOLEAN DEFAULT FALSE,
    description TEXT,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scope, scope_id, name)
);

CREATE INDEX idx_variables_scope ON variables(scope, scope_id);

-- Tenant Service Schema
CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    plan VARCHAR(50) NOT NULL DEFAULT 'free', -- free, starter, pro, enterprise
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    trial_ends_at TIMESTAMP,
    subscription_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_plan ON tenants(plan);
CREATE INDEX idx_tenants_status ON tenants(status);

-- Tenant Resource Limits
CREATE TABLE IF NOT EXISTS tenant_limits (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    max_workflows INTEGER DEFAULT 10,
    max_executions_per_day INTEGER DEFAULT 1000,
    max_nodes_per_workflow INTEGER DEFAULT 50,
    max_credentials INTEGER DEFAULT 20,
    max_team_members INTEGER DEFAULT 5,
    max_storage_mb INTEGER DEFAULT 1000,
    max_api_calls_per_minute INTEGER DEFAULT 100,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tenant_limits_tenant ON tenant_limits(tenant_id);

-- Tenant Feature Flags
CREATE TABLE IF NOT EXISTS tenant_features (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    feature_name VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    config JSONB DEFAULT '{}',
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, feature_name)
);

CREATE INDEX idx_tenant_features_tenant ON tenant_features(tenant_id);

-- Billing & Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL REFERENCES tenants(id),
    invoice_number VARCHAR(100) NOT NULL UNIQUE,
    amount_cents INTEGER NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, paid, failed, refunded
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    due_date TIMESTAMP NOT NULL,
    paid_at TIMESTAMP,
    stripe_invoice_id VARCHAR(255),
    line_items JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoices_tenant ON invoices(tenant_id);
CREATE INDEX idx_invoices_status ON invoices(status);

-- Executor Service Schema
CREATE TABLE IF NOT EXISTS executor_workers (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'idle', -- idle, busy, offline
    capabilities JSONB DEFAULT '[]', -- supported node types
    current_load INTEGER DEFAULT 0,
    max_load INTEGER DEFAULT 10,
    last_heartbeat_at TIMESTAMP,
    host VARCHAR(255),
    port INTEGER,
    version VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_executor_workers_status ON executor_workers(status);

-- Execution Tasks (queue for executor)
CREATE TABLE IF NOT EXISTS execution_tasks (
    id VARCHAR(255) PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    worker_id VARCHAR(255) REFERENCES executor_workers(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, assigned, running, completed, failed
    priority INTEGER DEFAULT 0,
    input_data JSONB,
    output_data JSONB,
    error TEXT,
    timeout_seconds INTEGER DEFAULT 300,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_execution_tasks_status ON execution_tasks(status);
CREATE INDEX idx_execution_tasks_worker ON execution_tasks(worker_id);
CREATE INDEX idx_execution_tasks_priority ON execution_tasks(priority DESC);
CREATE INDEX idx_execution_tasks_scheduled ON execution_tasks(scheduled_at);

-- Resource Usage Tracking
CREATE TABLE IF NOT EXISTS resource_usage (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) NOT NULL, -- execution, storage, api_call
    quantity INTEGER NOT NULL,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resource_usage_tenant ON resource_usage(tenant_id);
CREATE INDEX idx_resource_usage_period ON resource_usage(period_start, period_end);
CREATE INDEX idx_resource_usage_type ON resource_usage(resource_type);

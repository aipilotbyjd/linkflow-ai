-- Initial migration for LinkFlow AI microservices

-- Auth Service Schema
CREATE TABLE IF NOT EXISTS auth_sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token VARCHAR(512) NOT NULL UNIQUE,
    refresh_token VARCHAR(512),
    expires_at TIMESTAMP NOT NULL,
    refresh_expires_at TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX idx_auth_sessions_token ON auth_sessions(token);
CREATE INDEX idx_auth_sessions_expires_at ON auth_sessions(expires_at);

-- User Service Schema
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url TEXT,
    phone VARCHAR(20),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    email_verified BOOLEAN DEFAULT FALSE,
    email_verified_at TIMESTAMP,
    phone_verified BOOLEAN DEFAULT FALSE,
    phone_verified_at TIMESTAMP,
    last_login_at TIMESTAMP,
    login_count INTEGER DEFAULT 0,
    failed_login_count INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(255),
    preferences JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    version INTEGER DEFAULT 0
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);

CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    website VARCHAR(255),
    logo_url TEXT,
    owner_id VARCHAR(255) NOT NULL REFERENCES users(id),
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    subscription_tier VARCHAR(50) DEFAULT 'free',
    subscription_expires_at TIMESTAMP,
    member_count INTEGER DEFAULT 0,
    workflow_count INTEGER DEFAULT 0,
    execution_count INTEGER DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    version INTEGER DEFAULT 0
);

CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_owner_id ON organizations(owner_id);
CREATE INDEX idx_organizations_status ON organizations(status);

CREATE TABLE IF NOT EXISTS organization_members (
    id VARCHAR(255) PRIMARY KEY,
    organization_id VARCHAR(255) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    permissions JSONB DEFAULT '[]',
    invited_by VARCHAR(255) REFERENCES users(id),
    invited_at TIMESTAMP,
    joined_at TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, user_id)
);

CREATE INDEX idx_organization_members_org ON organization_members(organization_id);
CREATE INDEX idx_organization_members_user ON organization_members(user_id);
CREATE INDEX idx_organization_members_role ON organization_members(role);

-- Workflow Service Schema
CREATE TABLE IF NOT EXISTS workflows (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    nodes JSONB NOT NULL DEFAULT '[]',
    connections JSONB NOT NULL DEFAULT '[]',
    config JSONB DEFAULT '{}',
    tags TEXT[],
    category VARCHAR(100),
    visibility VARCHAR(50) DEFAULT 'private',
    is_template BOOLEAN DEFAULT FALSE,
    parent_id VARCHAR(255),
    execution_count INTEGER DEFAULT 0,
    last_executed_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    archived_at TIMESTAMP,
    deleted_at TIMESTAMP,
    optimistic_lock_version INTEGER DEFAULT 0
);

CREATE INDEX idx_workflows_user_id ON workflows(user_id);
CREATE INDEX idx_workflows_organization_id ON workflows(organization_id);
CREATE INDEX idx_workflows_status ON workflows(status);
CREATE INDEX idx_workflows_visibility ON workflows(visibility);
CREATE INDEX idx_workflows_is_template ON workflows(is_template);
CREATE INDEX idx_workflows_created_at ON workflows(created_at);

CREATE TABLE IF NOT EXISTS workflow_versions (
    id VARCHAR(255) PRIMARY KEY,
    workflow_id VARCHAR(255) NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    nodes JSONB NOT NULL,
    connections JSONB NOT NULL,
    config JSONB,
    change_description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version)
);

CREATE INDEX idx_workflow_versions_workflow ON workflow_versions(workflow_id);
CREATE INDEX idx_workflow_versions_version ON workflow_versions(version);

-- Execution Service Schema
CREATE TABLE IF NOT EXISTS executions (
    id VARCHAR(255) PRIMARY KEY,
    workflow_id VARCHAR(255) NOT NULL,
    workflow_version INTEGER NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255),
    trigger_type VARCHAR(50) NOT NULL,
    trigger_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input_data JSONB DEFAULT '{}',
    output_data JSONB DEFAULT '{}',
    context JSONB DEFAULT '{}',
    node_executions JSONB DEFAULT '{}',
    execution_path JSONB DEFAULT '[]',
    error JSONB,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    paused_at TIMESTAMP,
    duration_ms BIGINT DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    parent_execution_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 0
);

CREATE INDEX idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX idx_executions_user_id ON executions(user_id);
CREATE INDEX idx_executions_organization_id ON executions(organization_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_trigger_type ON executions(trigger_type);
CREATE INDEX idx_executions_created_at ON executions(created_at);
CREATE INDEX idx_executions_started_at ON executions(started_at);

CREATE TABLE IF NOT EXISTS execution_logs (
    id VARCHAR(255) PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    node_id VARCHAR(255),
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_execution_logs_execution ON execution_logs(execution_id);
CREATE INDEX idx_execution_logs_level ON execution_logs(level);
CREATE INDEX idx_execution_logs_created_at ON execution_logs(created_at);

-- Node Service Schema
CREATE TABLE IF NOT EXISTS node_definitions (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    category VARCHAR(100),
    description TEXT,
    icon VARCHAR(255),
    color VARCHAR(20),
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    inputs JSONB DEFAULT '[]',
    outputs JSONB DEFAULT '[]',
    properties JSONB DEFAULT '{}',
    configuration JSONB DEFAULT '{}',
    validation_rules JSONB DEFAULT '{}',
    documentation TEXT,
    examples JSONB DEFAULT '[]',
    tags TEXT[],
    author VARCHAR(255),
    is_system BOOLEAN DEFAULT FALSE,
    is_premium BOOLEAN DEFAULT FALSE,
    execution_handler VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_node_definitions_type ON node_definitions(type);
CREATE INDEX idx_node_definitions_category ON node_definitions(category);
CREATE INDEX idx_node_definitions_status ON node_definitions(status);
CREATE INDEX idx_node_definitions_is_system ON node_definitions(is_system);

-- Webhook Service Schema
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255),
    workflow_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    endpoint_url VARCHAR(500) NOT NULL UNIQUE,
    secret VARCHAR(255),
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers JSONB DEFAULT '{}',
    query_params JSONB DEFAULT '{}',
    authentication_type VARCHAR(50),
    authentication_config JSONB DEFAULT '{}',
    retry_config JSONB DEFAULT '{"maxRetries": 3, "retryDelay": 1000}',
    timeout_ms INTEGER DEFAULT 30000,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_triggered_at TIMESTAMP,
    trigger_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX idx_webhooks_organization_id ON webhooks(organization_id);
CREATE INDEX idx_webhooks_workflow_id ON webhooks(workflow_id);
CREATE INDEX idx_webhooks_endpoint_url ON webhooks(endpoint_url);
CREATE INDEX idx_webhooks_status ON webhooks(status);

-- Schedule Service Schema
CREATE TABLE IF NOT EXISTS schedules (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    organization_id VARCHAR(255),
    workflow_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(255),
    timezone VARCHAR(100) DEFAULT 'UTC',
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_schedules_user_id ON schedules(user_id);
CREATE INDEX idx_schedules_organization_id ON schedules(organization_id);
CREATE INDEX idx_schedules_workflow_id ON schedules(workflow_id);
CREATE INDEX idx_schedules_status ON schedules(status);
CREATE INDEX idx_schedules_next_run_at ON schedules(next_run_at);

-- Notification Service Schema
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    channel VARCHAR(50) NOT NULL DEFAULT 'in_app',
    title VARCHAR(255) NOT NULL,
    message TEXT,
    data JSONB DEFAULT '{}',
    priority VARCHAR(20) DEFAULT 'normal',
    status VARCHAR(50) NOT NULL DEFAULT 'unread',
    read_at TIMESTAMP,
    action_url TEXT,
    action_label VARCHAR(100),
    expires_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_channel ON notifications(channel);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

-- Audit Log Schema
CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255),
    organization_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    changes JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

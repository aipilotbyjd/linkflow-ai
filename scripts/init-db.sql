-- LinkFlow AI Database Initialization Script

-- Create database if not exists
SELECT 'CREATE DATABASE linkflow'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'linkflow')\gexec

-- Connect to linkflow database
\c linkflow;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create schemas for each service
CREATE SCHEMA IF NOT EXISTS auth_service;
CREATE SCHEMA IF NOT EXISTS user_service;
CREATE SCHEMA IF NOT EXISTS workflow_service;
CREATE SCHEMA IF NOT EXISTS execution_service;
CREATE SCHEMA IF NOT EXISTS node_service;
CREATE SCHEMA IF NOT EXISTS webhook_service;
CREATE SCHEMA IF NOT EXISTS schedule_service;
CREATE SCHEMA IF NOT EXISTS notification_service;
CREATE SCHEMA IF NOT EXISTS event_store;
CREATE SCHEMA IF NOT EXISTS read_models;

-- Event Store Tables
CREATE TABLE IF NOT EXISTS event_store.domain_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_version INTEGER NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB,
    user_id VARCHAR(255),
    correlation_id VARCHAR(255),
    causation_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_aggregate ON event_store.domain_events(aggregate_id, event_version);
CREATE INDEX IF NOT EXISTS idx_event_type ON event_store.domain_events(event_type);
CREATE INDEX IF NOT EXISTS idx_created_at ON event_store.domain_events(created_at);
CREATE INDEX IF NOT EXISTS idx_correlation_id ON event_store.domain_events(correlation_id);

-- Snapshots for Event Sourcing
CREATE TABLE IF NOT EXISTS event_store.snapshots (
    aggregate_id VARCHAR(255) PRIMARY KEY,
    aggregate_type VARCHAR(100) NOT NULL,
    version INTEGER NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auth Service Tables
CREATE TABLE IF NOT EXISTS auth_service.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON auth_service.users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON auth_service.users(username);
CREATE INDEX IF NOT EXISTS idx_users_status ON auth_service.users(status);

CREATE TABLE IF NOT EXISTS auth_service.sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES auth_service.users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    refresh_token_hash VARCHAR(255) UNIQUE,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON auth_service.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON auth_service.sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON auth_service.sessions(expires_at);

-- User Service Tables
CREATE TABLE IF NOT EXISTS user_service.profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    display_name VARCHAR(200),
    avatar_url TEXT,
    bio TEXT,
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON user_service.profiles(user_id);

CREATE TABLE IF NOT EXISTS user_service.organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    logo_url TEXT,
    website VARCHAR(255),
    settings JSONB,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON user_service.organizations(slug);

CREATE TABLE IF NOT EXISTS user_service.organization_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES user_service.organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_members_org ON user_service.organization_members(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON user_service.organization_members(user_id);

-- Workflow Service Tables
CREATE TABLE IF NOT EXISTS workflow_service.workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    organization_id UUID,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    nodes JSONB NOT NULL DEFAULT '[]'::jsonb,
    connections JSONB NOT NULL DEFAULT '[]'::jsonb,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    variables JSONB DEFAULT '{}'::jsonb,
    tags TEXT[],
    version INTEGER NOT NULL DEFAULT 1,
    is_template BOOLEAN DEFAULT FALSE,
    template_category VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workflows_user ON workflow_service.workflows(user_id);
CREATE INDEX IF NOT EXISTS idx_workflows_org ON workflow_service.workflows(organization_id);
CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflow_service.workflows(status);
CREATE INDEX IF NOT EXISTS idx_workflows_template ON workflow_service.workflows(is_template) WHERE is_template = true;
CREATE INDEX IF NOT EXISTS idx_workflows_tags ON workflow_service.workflows USING gin(tags);

CREATE TABLE IF NOT EXISTS workflow_service.workflow_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflow_service.workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    nodes JSONB NOT NULL,
    connections JSONB NOT NULL,
    settings JSONB NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_id, version)
);

CREATE INDEX IF NOT EXISTS idx_workflow_versions ON workflow_service.workflow_versions(workflow_id, version);

-- Execution Service Tables
CREATE TABLE IF NOT EXISTS execution_service.executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL,
    workflow_version INTEGER NOT NULL,
    user_id UUID NOT NULL,
    trigger_type VARCHAR(50) NOT NULL,
    trigger_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input_data JSONB,
    output_data JSONB,
    error_message TEXT,
    error_details JSONB,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_executions_workflow ON execution_service.executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_user ON execution_service.executions(user_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON execution_service.executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_created ON execution_service.executions(created_at DESC);

CREATE TABLE IF NOT EXISTS execution_service.node_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID NOT NULL REFERENCES execution_service.executions(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    node_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input_data JSONB,
    output_data JSONB,
    error_message TEXT,
    error_details JSONB,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_executions ON execution_service.node_executions(execution_id);
CREATE INDEX IF NOT EXISTS idx_node_exec_status ON execution_service.node_executions(status);

-- Node Service Tables
CREATE TABLE IF NOT EXISTS node_service.node_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    type VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(255),
    config_schema JSONB NOT NULL,
    input_schema JSONB,
    output_schema JSONB,
    version VARCHAR(50) NOT NULL,
    is_public BOOLEAN DEFAULT FALSE,
    author_id UUID,
    documentation TEXT,
    examples JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_defs_type ON node_service.node_definitions(type);
CREATE INDEX IF NOT EXISTS idx_node_defs_category ON node_service.node_definitions(category);
CREATE INDEX IF NOT EXISTS idx_node_defs_public ON node_service.node_definitions(is_public);

-- Webhook Service Tables
CREATE TABLE IF NOT EXISTS webhook_service.webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL,
    endpoint_id VARCHAR(255) UNIQUE NOT NULL,
    path VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    is_active BOOLEAN DEFAULT TRUE,
    secret VARCHAR(255),
    headers_validation JSONB,
    body_validation JSONB,
    rate_limit INTEGER DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_workflow ON webhook_service.webhooks(workflow_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_endpoint ON webhook_service.webhooks(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhook_service.webhooks(is_active);

CREATE TABLE IF NOT EXISTS webhook_service.webhook_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id UUID NOT NULL REFERENCES webhook_service.webhooks(id) ON DELETE CASCADE,
    execution_id UUID,
    method VARCHAR(10) NOT NULL,
    headers JSONB,
    body TEXT,
    response_status INTEGER,
    response_body TEXT,
    ip_address INET,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_logs ON webhook_service.webhook_logs(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_time ON webhook_service.webhook_logs(processed_at DESC);

-- Schedule Service Tables
CREATE TABLE IF NOT EXISTS schedule_service.schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    is_active BOOLEAN DEFAULT TRUE,
    input_data JSONB,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schedules_workflow ON schedule_service.schedules(workflow_id);
CREATE INDEX IF NOT EXISTS idx_schedules_active ON schedule_service.schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedule_service.schedules(next_run_at) WHERE is_active = true;

-- Notification Service Tables
CREATE TABLE IF NOT EXISTS notification_service.notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    channel VARCHAR(50) NOT NULL,
    subject VARCHAR(255),
    content TEXT NOT NULL,
    data JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    read_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notification_service.notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notification_service.notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notification_service.notifications(type);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notification_service.notifications(read_at) WHERE read_at IS NULL;

-- Create Kong database for API Gateway
CREATE DATABASE kong;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE linkflow TO postgres;
GRANT ALL PRIVILEGES ON DATABASE kong TO postgres;

-- Add indexes for better query performance

-- User indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Organization indexes
CREATE INDEX idx_organizations_owner_id ON organizations(owner_id);
CREATE INDEX idx_organization_members_org_id ON organization_members(organization_id);
CREATE INDEX idx_organization_members_user_id ON organization_members(user_id);

-- Workflow indexes
CREATE INDEX idx_workflows_user_id ON workflows(user_id);
CREATE INDEX idx_workflows_organization_id ON workflows(organization_id);
CREATE INDEX idx_workflows_status ON workflows(status);
CREATE INDEX idx_workflows_created_at ON workflows(created_at DESC);
CREATE INDEX idx_workflows_updated_at ON workflows(updated_at DESC);

-- Execution indexes
CREATE INDEX idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX idx_executions_user_id ON executions(user_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_started_at ON executions(started_at DESC);
CREATE INDEX idx_executions_completed_at ON executions(completed_at DESC);

-- Node indexes
CREATE INDEX idx_nodes_type ON nodes(type);
CREATE INDEX idx_nodes_created_by ON nodes(created_by);

-- Schedule indexes
CREATE INDEX idx_schedules_workflow_id ON schedules(workflow_id);
CREATE INDEX idx_schedules_user_id ON schedules(user_id);
CREATE INDEX idx_schedules_status ON schedules(status);
CREATE INDEX idx_schedules_next_run_at ON schedules(next_run_at);

-- Webhook indexes
CREATE INDEX idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX idx_webhooks_status ON webhooks(status);

-- Notification indexes
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- Analytics indexes
CREATE INDEX idx_analytics_events_user_id ON analytics_events(user_id);
CREATE INDEX idx_analytics_events_session_id ON analytics_events(session_id);
CREATE INDEX idx_analytics_events_event_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_events_timestamp ON analytics_events(timestamp DESC);

-- Storage indexes
CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_files_is_public ON files(is_public);
CREATE INDEX idx_files_uploaded_at ON files(uploaded_at DESC);

-- Integration indexes
CREATE INDEX idx_integrations_user_id ON integrations(user_id);
CREATE INDEX idx_integrations_organization_id ON integrations(organization_id);
CREATE INDEX idx_integrations_status ON integrations(status);
CREATE INDEX idx_integrations_type ON integrations(integration_type);

-- Configuration indexes
CREATE INDEX idx_configurations_scope ON configurations(scope);
CREATE INDEX idx_configurations_scope_id ON configurations(scope_id);
CREATE INDEX idx_configurations_key ON configurations(key);

-- Composite indexes for common queries
CREATE INDEX idx_workflows_user_status ON workflows(user_id, status);
CREATE INDEX idx_executions_workflow_status ON executions(workflow_id, status);
CREATE INDEX idx_notifications_user_status_created ON notifications(user_id, status, created_at DESC);
CREATE INDEX idx_analytics_events_user_type_timestamp ON analytics_events(user_id, event_type, timestamp DESC);

-- Remove all indexes created in 002_add_indexes.up.sql

-- User indexes
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_organization_id;
DROP INDEX IF EXISTS idx_users_created_at;

-- Organization indexes
DROP INDEX IF EXISTS idx_organizations_owner_id;
DROP INDEX IF EXISTS idx_organization_members_org_id;
DROP INDEX IF EXISTS idx_organization_members_user_id;

-- Workflow indexes
DROP INDEX IF EXISTS idx_workflows_user_id;
DROP INDEX IF EXISTS idx_workflows_organization_id;
DROP INDEX IF EXISTS idx_workflows_status;
DROP INDEX IF EXISTS idx_workflows_created_at;
DROP INDEX IF EXISTS idx_workflows_updated_at;

-- Execution indexes
DROP INDEX IF EXISTS idx_executions_workflow_id;
DROP INDEX IF EXISTS idx_executions_user_id;
DROP INDEX IF EXISTS idx_executions_status;
DROP INDEX IF EXISTS idx_executions_started_at;
DROP INDEX IF EXISTS idx_executions_completed_at;

-- Node indexes
DROP INDEX IF EXISTS idx_nodes_type;
DROP INDEX IF EXISTS idx_nodes_created_by;

-- Schedule indexes
DROP INDEX IF EXISTS idx_schedules_workflow_id;
DROP INDEX IF EXISTS idx_schedules_user_id;
DROP INDEX IF EXISTS idx_schedules_status;
DROP INDEX IF EXISTS idx_schedules_next_run_at;

-- Webhook indexes
DROP INDEX IF EXISTS idx_webhooks_user_id;
DROP INDEX IF EXISTS idx_webhooks_status;

-- Notification indexes
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_notifications_status;
DROP INDEX IF EXISTS idx_notifications_created_at;

-- Analytics indexes
DROP INDEX IF EXISTS idx_analytics_events_user_id;
DROP INDEX IF EXISTS idx_analytics_events_session_id;
DROP INDEX IF EXISTS idx_analytics_events_event_type;
DROP INDEX IF EXISTS idx_analytics_events_timestamp;

-- Storage indexes
DROP INDEX IF EXISTS idx_files_user_id;
DROP INDEX IF EXISTS idx_files_is_public;
DROP INDEX IF EXISTS idx_files_uploaded_at;

-- Integration indexes
DROP INDEX IF EXISTS idx_integrations_user_id;
DROP INDEX IF EXISTS idx_integrations_organization_id;
DROP INDEX IF EXISTS idx_integrations_status;
DROP INDEX IF EXISTS idx_integrations_type;

-- Configuration indexes
DROP INDEX IF EXISTS idx_configurations_scope;
DROP INDEX IF EXISTS idx_configurations_scope_id;
DROP INDEX IF EXISTS idx_configurations_key;

-- Composite indexes
DROP INDEX IF EXISTS idx_workflows_user_status;
DROP INDEX IF EXISTS idx_executions_workflow_status;
DROP INDEX IF EXISTS idx_notifications_user_status_created;
DROP INDEX IF EXISTS idx_analytics_events_user_type_timestamp;

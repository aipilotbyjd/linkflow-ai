-- ============================================================================
-- Migration: 000005_workflows
-- Description: Workflows and versions tables
-- ============================================================================

CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    nodes JSONB NOT NULL DEFAULT '[]',
    connections JSONB NOT NULL DEFAULT '[]',
    settings JSONB DEFAULT '{}',
    variables JSONB DEFAULT '{}',
    tags TEXT[],
    category VARCHAR(100),
    visibility VARCHAR(50) DEFAULT 'private',
    is_template BOOLEAN DEFAULT FALSE,
    template_category VARCHAR(100),
    parent_id UUID,
    execution_count INTEGER DEFAULT 0,
    last_executed_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_workflows_user_id ON workflows(user_id);
CREATE INDEX idx_workflows_organization_id ON workflows(organization_id);
CREATE INDEX idx_workflows_workspace_id ON workflows(workspace_id);
CREATE INDEX idx_workflows_status ON workflows(status);
CREATE INDEX idx_workflows_visibility ON workflows(visibility);
CREATE INDEX idx_workflows_is_template ON workflows(is_template) WHERE is_template = TRUE;
CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX idx_workflows_created_at ON workflows(created_at DESC);

-- Add self-reference after table exists
ALTER TABLE workflows ADD CONSTRAINT fk_workflows_parent 
    FOREIGN KEY (parent_id) REFERENCES workflows(id) ON DELETE SET NULL;

CREATE TABLE workflow_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    nodes JSONB NOT NULL,
    connections JSONB NOT NULL,
    settings JSONB,
    change_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_id, version)
);

CREATE INDEX idx_workflow_versions_workflow_id ON workflow_versions(workflow_id);
CREATE INDEX idx_workflow_versions_version ON workflow_versions(workflow_id, version);

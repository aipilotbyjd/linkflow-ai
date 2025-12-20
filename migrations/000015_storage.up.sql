-- ============================================================================
-- Migration: 000015_storage
-- Description: Files and storage tables
-- ============================================================================

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255),
    path TEXT NOT NULL,
    mime_type VARCHAR(100),
    file_type VARCHAR(50) DEFAULT 'other',
    size BIGINT NOT NULL DEFAULT 0,
    checksum VARCHAR(64),
    storage_provider VARCHAR(50) DEFAULT 'local',
    bucket VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    tags TEXT[],
    is_public BOOLEAN DEFAULT FALSE,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_files_workspace_id ON files(workspace_id);
CREATE INDEX idx_files_file_type ON files(file_type);
CREATE INDEX idx_files_is_public ON files(is_public);
CREATE INDEX idx_files_created_at ON files(created_at DESC);
CREATE INDEX idx_files_tags ON files USING GIN(tags);

CREATE TABLE file_shares (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    shared_with_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    shared_with_email VARCHAR(255),
    permission VARCHAR(50) NOT NULL DEFAULT 'view',
    token VARCHAR(255) UNIQUE,
    expires_at TIMESTAMPTZ,
    accessed_at TIMESTAMPTZ,
    access_count INTEGER DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_file_shares_file_id ON file_shares(file_id);
CREATE INDEX idx_file_shares_shared_with_user_id ON file_shares(shared_with_user_id);
CREATE INDEX idx_file_shares_token ON file_shares(token);

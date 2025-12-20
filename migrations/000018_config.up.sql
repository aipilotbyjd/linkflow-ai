-- ============================================================================
-- Migration: 000018_config
-- Description: Configuration and settings tables
-- ============================================================================

CREATE TABLE configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scope VARCHAR(50) NOT NULL,
    scope_id UUID,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    value_type VARCHAR(50) DEFAULT 'string',
    is_encrypted BOOLEAN DEFAULT FALSE,
    is_secret BOOLEAN DEFAULT FALSE,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(scope, scope_id, key)
);

CREATE INDEX idx_configurations_scope ON configurations(scope, scope_id);
CREATE INDEX idx_configurations_key ON configurations(key);

CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    enabled BOOLEAN DEFAULT FALSE,
    conditions JSONB DEFAULT '{}',
    rollout_percentage INTEGER DEFAULT 0,
    variants JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feature_flags_name ON feature_flags(name);
CREATE INDEX idx_feature_flags_enabled ON feature_flags(enabled);

CREATE TABLE feature_flag_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_flag_id UUID NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    scope VARCHAR(50) NOT NULL,
    scope_id UUID NOT NULL,
    enabled BOOLEAN NOT NULL,
    variant VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(feature_flag_id, scope, scope_id)
);

CREATE INDEX idx_feature_flag_overrides_flag_id ON feature_flag_overrides(feature_flag_id);
CREATE INDEX idx_feature_flag_overrides_scope ON feature_flag_overrides(scope, scope_id);

CREATE TABLE system_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(255) NOT NULL UNIQUE,
    value JSONB NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_settings_key ON system_settings(key);

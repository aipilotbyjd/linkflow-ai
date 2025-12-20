-- ============================================================================
-- Migration: 000007_nodes
-- Description: Node definitions table
-- ============================================================================

CREATE TABLE node_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(255),
    color VARCHAR(20),
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    inputs JSONB DEFAULT '[]',
    outputs JSONB DEFAULT '[]',
    properties JSONB DEFAULT '{}',
    config_schema JSONB DEFAULT '{}',
    validation_rules JSONB DEFAULT '{}',
    documentation TEXT,
    examples JSONB DEFAULT '[]',
    tags TEXT[],
    author VARCHAR(255),
    is_system BOOLEAN DEFAULT FALSE,
    is_premium BOOLEAN DEFAULT FALSE,
    execution_handler VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_node_definitions_type ON node_definitions(type);
CREATE INDEX idx_node_definitions_category ON node_definitions(category);
CREATE INDEX idx_node_definitions_status ON node_definitions(status);
CREATE INDEX idx_node_definitions_is_system ON node_definitions(is_system);
CREATE INDEX idx_node_definitions_tags ON node_definitions USING GIN(tags);

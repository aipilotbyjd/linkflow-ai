-- ============================================================================
-- Migration: 000004_workspaces (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS workspace_invitations CASCADE;
DROP TABLE IF EXISTS workspace_members CASCADE;
DROP TABLE IF EXISTS workspaces CASCADE;

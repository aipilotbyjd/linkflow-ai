-- ============================================================================
-- Migration: 000003_organizations (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS organization_members CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;

-- ============================================================================
-- Migration: 000019_audit (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS security_events CASCADE;
DROP TABLE IF EXISTS activity_feed CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;

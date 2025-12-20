-- ============================================================================
-- Migration: 000010_credentials (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS variables CASCADE;
DROP TABLE IF EXISTS credential_oauth_tokens CASCADE;
DROP TABLE IF EXISTS credentials CASCADE;

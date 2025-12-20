-- ============================================================================
-- Migration: 000002_users_auth (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS login_attempts CASCADE;
DROP TABLE IF EXISTS oauth_connections CASCADE;
DROP TABLE IF EXISTS email_verification_tokens CASCADE;
DROP TABLE IF EXISTS password_reset_tokens CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS auth_sessions CASCADE;
DROP TABLE IF EXISTS users CASCADE;

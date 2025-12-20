-- ============================================================================
-- Migration: 000017_integrations (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS integration_webhooks CASCADE;
DROP TABLE IF EXISTS integration_logs CASCADE;
DROP TABLE IF EXISTS integrations CASCADE;

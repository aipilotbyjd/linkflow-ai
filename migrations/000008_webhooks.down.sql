-- ============================================================================
-- Migration: 000008_webhooks (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS webhook_logs CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;

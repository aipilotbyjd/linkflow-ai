-- ============================================================================
-- Migration: 000011_notifications (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS notification_preferences CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;

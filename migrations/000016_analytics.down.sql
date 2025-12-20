-- ============================================================================
-- Migration: 000016_analytics (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS analytics_dashboards CASCADE;
DROP TABLE IF EXISTS analytics_aggregates CASCADE;
DROP TABLE IF EXISTS analytics_events CASCADE;

-- ============================================================================
-- Migration: 000006_executions (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS node_executions CASCADE;
DROP TABLE IF EXISTS execution_logs CASCADE;
DROP TABLE IF EXISTS executions CASCADE;

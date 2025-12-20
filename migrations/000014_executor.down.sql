-- ============================================================================
-- Migration: 000014_executor (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS task_queue CASCADE;
DROP TABLE IF EXISTS execution_tasks CASCADE;
DROP TABLE IF EXISTS executor_workers CASCADE;

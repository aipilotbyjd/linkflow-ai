-- ============================================================================
-- Migration: 000018_config (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS system_settings CASCADE;
DROP TABLE IF EXISTS feature_flag_overrides CASCADE;
DROP TABLE IF EXISTS feature_flags CASCADE;
DROP TABLE IF EXISTS configurations CASCADE;

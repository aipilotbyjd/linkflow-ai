-- ============================================================================
-- Migration: 000012_tenants (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS resource_usage CASCADE;
DROP TABLE IF EXISTS tenant_features CASCADE;
DROP TABLE IF EXISTS tenant_limits CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

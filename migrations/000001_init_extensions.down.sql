-- ============================================================================
-- Migration: 000001_init_extensions (ROLLBACK)
-- ============================================================================

DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";

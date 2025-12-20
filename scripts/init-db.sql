-- ============================================================================
-- LinkFlow AI - Database Initialization Script
-- Run this before migrations to set up the database
-- ============================================================================

-- Create database (run as postgres superuser)
-- Note: This should be run separately if database doesn't exist
-- CREATE DATABASE linkflow;

-- Connect to linkflow database
\c linkflow;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create application user (optional, for production)
-- DO $$
-- BEGIN
--     IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'linkflow_app') THEN
--         CREATE ROLE linkflow_app WITH LOGIN PASSWORD 'changeme';
--     END IF;
-- END
-- $$;

-- Grant permissions (for production)
-- GRANT ALL PRIVILEGES ON DATABASE linkflow TO linkflow_app;
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO linkflow_app;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO linkflow_app;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO linkflow_app;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO linkflow_app;

-- Create Kong database (for API Gateway)
SELECT 'CREATE DATABASE kong'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'kong')\gexec

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE linkflow TO postgres;

\echo 'Database initialization completed successfully!'
\echo 'Run migrations with: make migrate-up'

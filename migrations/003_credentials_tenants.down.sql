-- Rollback: Credentials and Tenants

DROP TABLE IF EXISTS resource_usage;
DROP TABLE IF EXISTS execution_tasks;
DROP TABLE IF EXISTS executor_workers;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS tenant_features;
DROP TABLE IF EXISTS tenant_limits;
DROP TABLE IF EXISTS tenants;
DROP TABLE IF EXISTS variables;
DROP TABLE IF EXISTS oauth2_tokens;
DROP TABLE IF EXISTS credentials;

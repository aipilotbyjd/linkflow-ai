-- ============================================================================
-- Migration: 000013_billing (ROLLBACK)
-- ============================================================================

DROP TABLE IF EXISTS stripe_webhook_events CASCADE;
DROP TABLE IF EXISTS billing_usage CASCADE;
DROP TABLE IF EXISTS payment_methods CASCADE;
DROP TABLE IF EXISTS billing_invoices CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS billing_customers CASCADE;
DROP TABLE IF EXISTS billing_plans CASCADE;

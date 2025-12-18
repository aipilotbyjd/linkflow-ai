-- Billing plans
CREATE TABLE IF NOT EXISTS billing_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    monthly_price_id VARCHAR(100),
    yearly_price_id VARCHAR(100),
    monthly_price BIGINT NOT NULL DEFAULT 0,
    yearly_price BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    features JSONB DEFAULT '[]',
    limits JSONB DEFAULT '{}',
    is_public BOOLEAN DEFAULT true,
    trial_days INT DEFAULT 0,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default plans
INSERT INTO billing_plans (name, slug, description, monthly_price, yearly_price, features, limits, trial_days, sort_order) VALUES
('Free', 'free', 'For individuals getting started', 0, 0,
 '["5 workflows", "100 executions/month", "3 team members", "Community support"]',
 '{"maxMembers": 3, "maxWorkflows": 5, "maxExecutionsPerMonth": 100, "maxCredentials": 5, "maxWebhooks": 2, "retentionDays": 7, "supportLevel": "community"}',
 0, 1),
 
('Pro', 'pro', 'For growing teams', 2900, 29000,
 '["Unlimited workflows", "10,000 executions/month", "10 team members", "Email support", "30-day retention"]',
 '{"maxMembers": 10, "maxWorkflows": -1, "maxExecutionsPerMonth": 10000, "maxCredentials": 50, "maxWebhooks": 20, "retentionDays": 30, "supportLevel": "email"}',
 14, 2),
 
('Business', 'business', 'For scaling organizations', 9900, 99000,
 '["Unlimited everything", "Priority support", "SSO/SAML", "90-day retention", "Custom integrations"]',
 '{"maxMembers": -1, "maxWorkflows": -1, "maxExecutionsPerMonth": -1, "maxCredentials": -1, "maxWebhooks": -1, "retentionDays": 90, "supportLevel": "priority"}',
 14, 3),
 
('Enterprise', 'enterprise', 'For large enterprises', 0, 0,
 '["Everything in Business", "Dedicated support", "Custom contracts", "On-premise option", "SLA guarantee"]',
 '{"maxMembers": -1, "maxWorkflows": -1, "maxExecutionsPerMonth": -1, "maxCredentials": -1, "maxWebhooks": -1, "retentionDays": 365, "supportLevel": "dedicated"}',
 0, 4)
ON CONFLICT (slug) DO NOTHING;

-- Billing customers
CREATE TABLE IF NOT EXISTS billing_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE UNIQUE,
    stripe_customer_id VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    payment_method_id VARCHAR(100),
    default_currency VARCHAR(3) DEFAULT 'usd',
    tax_id VARCHAR(50),
    address JSONB,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_billing_customers_workspace ON billing_customers(workspace_id);
CREATE INDEX idx_billing_customers_stripe ON billing_customers(stripe_customer_id);

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES billing_plans(id),
    stripe_subscription_id VARCHAR(100) NOT NULL UNIQUE,
    stripe_customer_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    cancel_at_period_end BOOLEAN DEFAULT false,
    canceled_at TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    quantity INT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_workspace ON subscriptions(workspace_id);
CREATE INDEX idx_subscriptions_stripe ON subscriptions(stripe_subscription_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

-- Invoices
CREATE TABLE IF NOT EXISTS billing_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    stripe_invoice_id VARCHAR(100) NOT NULL UNIQUE,
    number VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    subtotal BIGINT NOT NULL DEFAULT 0,
    tax BIGINT DEFAULT 0,
    total BIGINT NOT NULL DEFAULT 0,
    amount_paid BIGINT DEFAULT 0,
    amount_due BIGINT DEFAULT 0,
    line_items JSONB DEFAULT '[]',
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    hosted_invoice_url TEXT,
    invoice_pdf_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_billing_invoices_workspace ON billing_invoices(workspace_id, created_at DESC);
CREATE INDEX idx_billing_invoices_stripe ON billing_invoices(stripe_invoice_id);

-- Payment methods
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    stripe_payment_method_id VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    is_default BOOLEAN DEFAULT false,
    card_details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_payment_methods_workspace ON payment_methods(workspace_id);

-- Usage tracking
CREATE TABLE IF NOT EXISTS billing_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    period DATE NOT NULL,
    executions_count INT DEFAULT 0,
    api_calls_count INT DEFAULT 0,
    storage_used_bytes BIGINT DEFAULT 0,
    active_workflows INT DEFAULT 0,
    active_members INT DEFAULT 0,
    webhooks_count INT DEFAULT 0,
    credentials_count INT DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(workspace_id, period)
);

CREATE INDEX idx_billing_usage_workspace_period ON billing_usage(workspace_id, period DESC);

-- Webhook events (for idempotency)
CREATE TABLE IF NOT EXISTS stripe_webhook_events (
    id VARCHAR(100) PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    data JSONB NOT NULL,
    processed BOOLEAN DEFAULT false,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_stripe_webhook_events_processed ON stripe_webhook_events(processed, created_at);

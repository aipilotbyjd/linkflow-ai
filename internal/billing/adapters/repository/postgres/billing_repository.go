// Package postgres provides PostgreSQL repository implementations for billing
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/billing/domain/model"
)

// CustomerRepository implements customer persistence
type CustomerRepository struct {
	db *sql.DB
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// Create creates a new customer
func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
	address, _ := json.Marshal(customer.Address)
	metadata, _ := json.Marshal(customer.Metadata)
	
	query := `
		INSERT INTO billing_customers (id, workspace_id, stripe_customer_id, email, name, payment_method_id, default_currency, tax_id, address, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		customer.ID,
		customer.WorkspaceID,
		customer.StripeCustomerID,
		customer.Email,
		customer.Name,
		customer.PaymentMethodID,
		customer.DefaultCurrency,
		customer.TaxID,
		address,
		metadata,
		customer.CreatedAt,
		customer.UpdatedAt,
	)
	
	return err
}

// FindByID finds a customer by ID
func (r *CustomerRepository) FindByID(ctx context.Context, id string) (*model.Customer, error) {
	return r.findBy(ctx, "id", id)
}

// FindByWorkspaceID finds a customer by workspace ID
func (r *CustomerRepository) FindByWorkspaceID(ctx context.Context, workspaceID string) (*model.Customer, error) {
	return r.findBy(ctx, "workspace_id", workspaceID)
}

// FindByStripeID finds a customer by Stripe customer ID
func (r *CustomerRepository) FindByStripeID(ctx context.Context, stripeID string) (*model.Customer, error) {
	return r.findBy(ctx, "stripe_customer_id", stripeID)
}

func (r *CustomerRepository) findBy(ctx context.Context, field, value string) (*model.Customer, error) {
	query := fmt.Sprintf(`
		SELECT id, workspace_id, stripe_customer_id, email, name, payment_method_id, default_currency, tax_id, address, metadata, created_at, updated_at
		FROM billing_customers
		WHERE %s = $1
	`, field)
	
	var c model.Customer
	var address, metadata []byte
	
	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&c.ID,
		&c.WorkspaceID,
		&c.StripeCustomerID,
		&c.Email,
		&c.Name,
		&c.PaymentMethodID,
		&c.DefaultCurrency,
		&c.TaxID,
		&address,
		&metadata,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrCustomerNotFound
	}
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(address, &c.Address)
	json.Unmarshal(metadata, &c.Metadata)
	
	return &c, nil
}

// Update updates a customer
func (r *CustomerRepository) Update(ctx context.Context, customer *model.Customer) error {
	address, _ := json.Marshal(customer.Address)
	metadata, _ := json.Marshal(customer.Metadata)
	
	query := `
		UPDATE billing_customers
		SET email = $1, name = $2, payment_method_id = $3, default_currency = $4, tax_id = $5, address = $6, metadata = $7, updated_at = $8
		WHERE id = $9
	`
	
	_, err := r.db.ExecContext(ctx, query,
		customer.Email,
		customer.Name,
		customer.PaymentMethodID,
		customer.DefaultCurrency,
		customer.TaxID,
		address,
		metadata,
		time.Now(),
		customer.ID,
	)
	
	return err
}

// Delete deletes a customer
func (r *CustomerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM billing_customers WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// SubscriptionRepository implements subscription persistence
type SubscriptionRepository struct {
	db *sql.DB
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Create creates a new subscription
func (r *SubscriptionRepository) Create(ctx context.Context, sub *model.Subscription) error {
	query := `
		INSERT INTO subscriptions (id, workspace_id, plan_id, stripe_subscription_id, stripe_customer_id, status, current_period_start, current_period_end, cancel_at_period_end, canceled_at, trial_start, trial_end, quantity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		sub.ID,
		sub.WorkspaceID,
		sub.PlanID,
		sub.StripeSubscriptionID,
		sub.StripeCustomerID,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd,
		sub.CanceledAt,
		sub.TrialStart,
		sub.TrialEnd,
		sub.Quantity,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	
	return err
}

// FindByID finds a subscription by ID
func (r *SubscriptionRepository) FindByID(ctx context.Context, id string) (*model.Subscription, error) {
	return r.findBy(ctx, "id", id)
}

// FindByWorkspaceID finds a subscription by workspace ID
func (r *SubscriptionRepository) FindByWorkspaceID(ctx context.Context, workspaceID string) (*model.Subscription, error) {
	return r.findBy(ctx, "workspace_id", workspaceID)
}

// FindByStripeID finds a subscription by Stripe subscription ID
func (r *SubscriptionRepository) FindByStripeID(ctx context.Context, stripeID string) (*model.Subscription, error) {
	return r.findBy(ctx, "stripe_subscription_id", stripeID)
}

func (r *SubscriptionRepository) findBy(ctx context.Context, field, value string) (*model.Subscription, error) {
	query := fmt.Sprintf(`
		SELECT id, workspace_id, plan_id, stripe_subscription_id, stripe_customer_id, status, current_period_start, current_period_end, cancel_at_period_end, canceled_at, trial_start, trial_end, quantity, created_at, updated_at
		FROM subscriptions
		WHERE %s = $1
	`, field)
	
	var s model.Subscription
	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&s.ID,
		&s.WorkspaceID,
		&s.PlanID,
		&s.StripeSubscriptionID,
		&s.StripeCustomerID,
		&s.Status,
		&s.CurrentPeriodStart,
		&s.CurrentPeriodEnd,
		&s.CancelAtPeriodEnd,
		&s.CanceledAt,
		&s.TrialStart,
		&s.TrialEnd,
		&s.Quantity,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrSubscriptionNotFound
	}
	return &s, err
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, sub *model.Subscription) error {
	query := `
		UPDATE subscriptions
		SET plan_id = $1, status = $2, current_period_start = $3, current_period_end = $4, cancel_at_period_end = $5, canceled_at = $6, trial_start = $7, trial_end = $8, quantity = $9, updated_at = $10
		WHERE id = $11
	`
	
	_, err := r.db.ExecContext(ctx, query,
		sub.PlanID,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd,
		sub.CanceledAt,
		sub.TrialStart,
		sub.TrialEnd,
		sub.Quantity,
		time.Now(),
		sub.ID,
	)
	
	return err
}

// Delete deletes a subscription
func (r *SubscriptionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM subscriptions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// PlanRepository implements plan persistence
type PlanRepository struct {
	db *sql.DB
}

// NewPlanRepository creates a new plan repository
func NewPlanRepository(db *sql.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

// FindByID finds a plan by ID
func (r *PlanRepository) FindByID(ctx context.Context, id string) (*model.Plan, error) {
	return r.findBy(ctx, "id", id)
}

// FindBySlug finds a plan by slug
func (r *PlanRepository) FindBySlug(ctx context.Context, slug string) (*model.Plan, error) {
	return r.findBy(ctx, "slug", slug)
}

func (r *PlanRepository) findBy(ctx context.Context, field, value string) (*model.Plan, error) {
	query := fmt.Sprintf(`
		SELECT id, name, slug, description, monthly_price_id, yearly_price_id, monthly_price, yearly_price, currency, features, limits, is_public, trial_days, created_at, updated_at
		FROM billing_plans
		WHERE %s = $1
	`, field)
	
	var p model.Plan
	var features, limits []byte
	
	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.MonthlyPriceID,
		&p.YearlyPriceID,
		&p.MonthlyPrice,
		&p.YearlyPrice,
		&p.Currency,
		&features,
		&limits,
		&p.IsPublic,
		&p.TrialDays,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrPlanNotFound
	}
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(features, &p.Features)
	json.Unmarshal(limits, &p.Limits)
	
	return &p, nil
}

// ListPublic lists all public plans
func (r *PlanRepository) ListPublic(ctx context.Context) ([]*model.Plan, error) {
	query := `
		SELECT id, name, slug, description, monthly_price_id, yearly_price_id, monthly_price, yearly_price, currency, features, limits, is_public, trial_days, created_at, updated_at
		FROM billing_plans
		WHERE is_public = true
		ORDER BY sort_order
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var plans []*model.Plan
	for rows.Next() {
		var p model.Plan
		var features, limits []byte
		
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Slug,
			&p.Description,
			&p.MonthlyPriceID,
			&p.YearlyPriceID,
			&p.MonthlyPrice,
			&p.YearlyPrice,
			&p.Currency,
			&features,
			&limits,
			&p.IsPublic,
			&p.TrialDays,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(features, &p.Features)
		json.Unmarshal(limits, &p.Limits)
		
		plans = append(plans, &p)
	}
	
	return plans, rows.Err()
}

// InvoiceRepository implements invoice persistence
type InvoiceRepository struct {
	db *sql.DB
}

// NewInvoiceRepository creates a new invoice repository
func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

// Create creates a new invoice
func (r *InvoiceRepository) Create(ctx context.Context, invoice *model.Invoice) error {
	lineItems, _ := json.Marshal(invoice.LineItems)
	
	query := `
		INSERT INTO billing_invoices (id, workspace_id, stripe_invoice_id, number, status, currency, subtotal, tax, total, amount_paid, amount_due, line_items, period_start, period_end, due_date, paid_at, hosted_invoice_url, invoice_pdf_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		invoice.ID,
		invoice.WorkspaceID,
		invoice.StripeInvoiceID,
		invoice.Number,
		invoice.Status,
		invoice.Currency,
		invoice.Subtotal,
		invoice.Tax,
		invoice.Total,
		invoice.AmountPaid,
		invoice.AmountDue,
		lineItems,
		invoice.PeriodStart,
		invoice.PeriodEnd,
		invoice.DueDate,
		invoice.PaidAt,
		invoice.HostedInvoiceURL,
		invoice.InvoicePDFURL,
		invoice.CreatedAt,
	)
	
	return err
}

// FindByID finds an invoice by ID
func (r *InvoiceRepository) FindByID(ctx context.Context, id string) (*model.Invoice, error) {
	query := `
		SELECT id, workspace_id, stripe_invoice_id, number, status, currency, subtotal, tax, total, amount_paid, amount_due, line_items, period_start, period_end, due_date, paid_at, hosted_invoice_url, invoice_pdf_url, created_at
		FROM billing_invoices
		WHERE id = $1
	`
	
	var inv model.Invoice
	var lineItems []byte
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID,
		&inv.WorkspaceID,
		&inv.StripeInvoiceID,
		&inv.Number,
		&inv.Status,
		&inv.Currency,
		&inv.Subtotal,
		&inv.Tax,
		&inv.Total,
		&inv.AmountPaid,
		&inv.AmountDue,
		&lineItems,
		&inv.PeriodStart,
		&inv.PeriodEnd,
		&inv.DueDate,
		&inv.PaidAt,
		&inv.HostedInvoiceURL,
		&inv.InvoicePDFURL,
		&inv.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(lineItems, &inv.LineItems)
	
	return &inv, nil
}

// FindByStripeID finds an invoice by Stripe invoice ID
func (r *InvoiceRepository) FindByStripeID(ctx context.Context, stripeID string) (*model.Invoice, error) {
	query := `
		SELECT id, workspace_id, stripe_invoice_id, number, status, currency, subtotal, tax, total, amount_paid, amount_due, line_items, period_start, period_end, due_date, paid_at, hosted_invoice_url, invoice_pdf_url, created_at
		FROM billing_invoices
		WHERE stripe_invoice_id = $1
	`
	
	var inv model.Invoice
	var lineItems []byte
	
	err := r.db.QueryRowContext(ctx, query, stripeID).Scan(
		&inv.ID,
		&inv.WorkspaceID,
		&inv.StripeInvoiceID,
		&inv.Number,
		&inv.Status,
		&inv.Currency,
		&inv.Subtotal,
		&inv.Tax,
		&inv.Total,
		&inv.AmountPaid,
		&inv.AmountDue,
		&lineItems,
		&inv.PeriodStart,
		&inv.PeriodEnd,
		&inv.DueDate,
		&inv.PaidAt,
		&inv.HostedInvoiceURL,
		&inv.InvoicePDFURL,
		&inv.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(lineItems, &inv.LineItems)
	
	return &inv, nil
}

// ListByWorkspace lists invoices for a workspace
func (r *InvoiceRepository) ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*model.Invoice, int64, error) {
	// Count total
	var total int64
	countQuery := `SELECT COUNT(*) FROM billing_invoices WHERE workspace_id = $1`
	r.db.QueryRowContext(ctx, countQuery, workspaceID).Scan(&total)
	
	// Get invoices
	query := `
		SELECT id, workspace_id, stripe_invoice_id, number, status, currency, subtotal, tax, total, amount_paid, amount_due, line_items, period_start, period_end, due_date, paid_at, hosted_invoice_url, invoice_pdf_url, created_at
		FROM billing_invoices
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var invoices []*model.Invoice
	for rows.Next() {
		var inv model.Invoice
		var lineItems []byte
		
		err := rows.Scan(
			&inv.ID,
			&inv.WorkspaceID,
			&inv.StripeInvoiceID,
			&inv.Number,
			&inv.Status,
			&inv.Currency,
			&inv.Subtotal,
			&inv.Tax,
			&inv.Total,
			&inv.AmountPaid,
			&inv.AmountDue,
			&lineItems,
			&inv.PeriodStart,
			&inv.PeriodEnd,
			&inv.DueDate,
			&inv.PaidAt,
			&inv.HostedInvoiceURL,
			&inv.InvoicePDFURL,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		
		json.Unmarshal(lineItems, &inv.LineItems)
		invoices = append(invoices, &inv)
	}
	
	return invoices, total, rows.Err()
}

// Update updates an invoice
func (r *InvoiceRepository) Update(ctx context.Context, invoice *model.Invoice) error {
	query := `
		UPDATE billing_invoices
		SET status = $1, amount_paid = $2, amount_due = $3, paid_at = $4
		WHERE id = $5
	`
	
	_, err := r.db.ExecContext(ctx, query,
		invoice.Status,
		invoice.AmountPaid,
		invoice.AmountDue,
		invoice.PaidAt,
		invoice.ID,
	)
	
	return err
}

// UsageRepository implements usage tracking persistence
type UsageRepository struct {
	db *sql.DB
}

// NewUsageRepository creates a new usage repository
func NewUsageRepository(db *sql.DB) *UsageRepository {
	return &UsageRepository{db: db}
}

// GetOrCreate gets or creates usage record for a workspace and period
func (r *UsageRepository) GetOrCreate(ctx context.Context, workspaceID string, period time.Time) (*model.Usage, error) {
	// Normalize period to first day of month
	period = time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	// Try to find existing
	usage, err := r.findByWorkspaceAndPeriod(ctx, workspaceID, period)
	if err == nil {
		return usage, nil
	}
	
	// Create new
	usage = model.NewUsage(workspaceID, period)
	
	query := `
		INSERT INTO billing_usage (id, workspace_id, period, executions_count, api_calls_count, storage_used_bytes, active_workflows, active_members, webhooks_count, credentials_count, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (workspace_id, period) DO NOTHING
	`
	
	r.db.ExecContext(ctx, query,
		usage.ID,
		usage.WorkspaceID,
		usage.Period,
		usage.ExecutionsCount,
		usage.APICallsCount,
		usage.StorageUsedBytes,
		usage.ActiveWorkflows,
		usage.ActiveMembers,
		usage.WebhooksCount,
		usage.CredentialsCount,
		usage.LastUpdated,
	)
	
	return r.findByWorkspaceAndPeriod(ctx, workspaceID, period)
}

func (r *UsageRepository) findByWorkspaceAndPeriod(ctx context.Context, workspaceID string, period time.Time) (*model.Usage, error) {
	query := `
		SELECT id, workspace_id, period, executions_count, api_calls_count, storage_used_bytes, active_workflows, active_members, webhooks_count, credentials_count, last_updated
		FROM billing_usage
		WHERE workspace_id = $1 AND period = $2
	`
	
	var u model.Usage
	err := r.db.QueryRowContext(ctx, query, workspaceID, period).Scan(
		&u.ID,
		&u.WorkspaceID,
		&u.Period,
		&u.ExecutionsCount,
		&u.APICallsCount,
		&u.StorageUsedBytes,
		&u.ActiveWorkflows,
		&u.ActiveMembers,
		&u.WebhooksCount,
		&u.CredentialsCount,
		&u.LastUpdated,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("usage not found")
	}
	return &u, err
}

// Update updates usage record
func (r *UsageRepository) Update(ctx context.Context, usage *model.Usage) error {
	query := `
		UPDATE billing_usage
		SET executions_count = $1, api_calls_count = $2, storage_used_bytes = $3, active_workflows = $4, active_members = $5, webhooks_count = $6, credentials_count = $7, last_updated = $8
		WHERE id = $9
	`
	
	_, err := r.db.ExecContext(ctx, query,
		usage.ExecutionsCount,
		usage.APICallsCount,
		usage.StorageUsedBytes,
		usage.ActiveWorkflows,
		usage.ActiveMembers,
		usage.WebhooksCount,
		usage.CredentialsCount,
		time.Now(),
		usage.ID,
	)
	
	return err
}

// GetCurrentUsage gets current month's usage for a workspace
func (r *UsageRepository) GetCurrentUsage(ctx context.Context, workspaceID string) (*model.Usage, error) {
	now := time.Now()
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return r.GetOrCreate(ctx, workspaceID, period)
}

// IncrementExecutions increments execution count
func (r *UsageRepository) IncrementExecutions(ctx context.Context, workspaceID string, count int) error {
	now := time.Now()
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	query := `
		INSERT INTO billing_usage (id, workspace_id, period, executions_count, last_updated)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (workspace_id, period) DO UPDATE
		SET executions_count = billing_usage.executions_count + $4, last_updated = $5
	`
	
	_, err := r.db.ExecContext(ctx, query,
		fmt.Sprintf("%s-%s", workspaceID, period.Format("2006-01")),
		workspaceID,
		period,
		count,
		now,
	)
	
	return err
}

// IncrementAPICalls increments API call count
func (r *UsageRepository) IncrementAPICalls(ctx context.Context, workspaceID string, count int) error {
	now := time.Now()
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	query := `
		INSERT INTO billing_usage (id, workspace_id, period, api_calls_count, last_updated)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (workspace_id, period) DO UPDATE
		SET api_calls_count = billing_usage.api_calls_count + $4, last_updated = $5
	`
	
	_, err := r.db.ExecContext(ctx, query,
		fmt.Sprintf("%s-%s", workspaceID, period.Format("2006-01")),
		workspaceID,
		period,
		count,
		now,
	)
	
	return err
}

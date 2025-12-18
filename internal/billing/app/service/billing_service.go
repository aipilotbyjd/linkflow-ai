// Package service provides billing business logic
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/billing/domain/model"
)

// StripeClient defines Stripe operations
type StripeClient interface {
	CreateCustomer(ctx context.Context, email, name string, metadata map[string]string) (string, error)
	UpdateCustomer(ctx context.Context, customerID string, params map[string]interface{}) error
	DeleteCustomer(ctx context.Context, customerID string) error
	
	CreateSubscription(ctx context.Context, customerID, priceID string, trialDays int) (*StripeSubscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, params map[string]interface{}) error
	CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error
	GetSubscription(ctx context.Context, subscriptionID string) (*StripeSubscription, error)
	
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (*CheckoutSession, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)
	
	GetInvoice(ctx context.Context, invoiceID string) (*StripeInvoice, error)
	ListInvoices(ctx context.Context, customerID string, limit int) ([]*StripeInvoice, error)
	
	AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error
	SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error
	ListPaymentMethods(ctx context.Context, customerID string) ([]*StripePaymentMethod, error)
	
	ConstructWebhookEvent(payload []byte, signature string) (*WebhookEvent, error)
}

// StripeSubscription represents Stripe subscription response
type StripeSubscription struct {
	ID                 string
	CustomerID         string
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
	TrialStart         *time.Time
	TrialEnd           *time.Time
	Items              []StripeSubscriptionItem
}

// StripeSubscriptionItem represents subscription item
type StripeSubscriptionItem struct {
	PriceID  string
	Quantity int
}

// CheckoutSession represents Stripe checkout session
type CheckoutSession struct {
	ID  string
	URL string
}

// StripeInvoice represents Stripe invoice
type StripeInvoice struct {
	ID               string
	CustomerID       string
	SubscriptionID   string
	Number           string
	Status           string
	Currency         string
	Subtotal         int64
	Tax              int64
	Total            int64
	AmountPaid       int64
	AmountDue        int64
	PeriodStart      time.Time
	PeriodEnd        time.Time
	DueDate          *time.Time
	PaidAt           *time.Time
	HostedInvoiceURL string
	InvoicePDF       string
}

// StripePaymentMethod represents Stripe payment method
type StripePaymentMethod struct {
	ID       string
	Type     string
	Card     *model.CardDetails
	IsDefault bool
}

// WebhookEvent represents Stripe webhook event
type WebhookEvent struct {
	ID   string
	Type string
	Data map[string]interface{}
}

// CustomerRepository defines customer persistence
type CustomerRepository interface {
	Create(ctx context.Context, customer *model.Customer) error
	FindByID(ctx context.Context, id string) (*model.Customer, error)
	FindByWorkspaceID(ctx context.Context, workspaceID string) (*model.Customer, error)
	FindByStripeID(ctx context.Context, stripeID string) (*model.Customer, error)
	Update(ctx context.Context, customer *model.Customer) error
	Delete(ctx context.Context, id string) error
}

// SubscriptionRepository defines subscription persistence
type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *model.Subscription) error
	FindByID(ctx context.Context, id string) (*model.Subscription, error)
	FindByWorkspaceID(ctx context.Context, workspaceID string) (*model.Subscription, error)
	FindByStripeID(ctx context.Context, stripeID string) (*model.Subscription, error)
	Update(ctx context.Context, subscription *model.Subscription) error
	Delete(ctx context.Context, id string) error
}

// PlanRepository defines plan persistence
type PlanRepository interface {
	FindByID(ctx context.Context, id string) (*model.Plan, error)
	FindBySlug(ctx context.Context, slug string) (*model.Plan, error)
	ListPublic(ctx context.Context) ([]*model.Plan, error)
}

// InvoiceRepository defines invoice persistence
type InvoiceRepository interface {
	Create(ctx context.Context, invoice *model.Invoice) error
	FindByID(ctx context.Context, id string) (*model.Invoice, error)
	FindByStripeID(ctx context.Context, stripeID string) (*model.Invoice, error)
	ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*model.Invoice, int64, error)
	Update(ctx context.Context, invoice *model.Invoice) error
}

// UsageRepository defines usage persistence
type UsageRepository interface {
	GetOrCreate(ctx context.Context, workspaceID string, period time.Time) (*model.Usage, error)
	Update(ctx context.Context, usage *model.Usage) error
	GetCurrentUsage(ctx context.Context, workspaceID string) (*model.Usage, error)
	IncrementExecutions(ctx context.Context, workspaceID string, count int) error
	IncrementAPICalls(ctx context.Context, workspaceID string, count int) error
}

// BillingService handles billing operations
type BillingService struct {
	stripe           StripeClient
	customerRepo     CustomerRepository
	subscriptionRepo SubscriptionRepository
	planRepo         PlanRepository
	invoiceRepo      InvoiceRepository
	usageRepo        UsageRepository
}

// NewBillingService creates a new billing service
func NewBillingService(
	stripe StripeClient,
	customerRepo CustomerRepository,
	subscriptionRepo SubscriptionRepository,
	planRepo PlanRepository,
	invoiceRepo InvoiceRepository,
	usageRepo UsageRepository,
) *BillingService {
	return &BillingService{
		stripe:           stripe,
		customerRepo:     customerRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		invoiceRepo:      invoiceRepo,
		usageRepo:        usageRepo,
	}
}

// CreateCustomerInput represents customer creation input
type CreateCustomerInput struct {
	WorkspaceID string
	Email       string
	Name        string
}

// CreateCustomer creates a billing customer
func (s *BillingService) CreateCustomer(ctx context.Context, input CreateCustomerInput) (*model.Customer, error) {
	// Check if customer exists
	existing, _ := s.customerRepo.FindByWorkspaceID(ctx, input.WorkspaceID)
	if existing != nil {
		return existing, nil
	}

	// Create in Stripe
	metadata := map[string]string{"workspace_id": input.WorkspaceID}
	stripeCustomerID, err := s.stripe.CreateCustomer(ctx, input.Email, input.Name, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	// Create in database
	customer := model.NewCustomer(input.WorkspaceID, stripeCustomerID, input.Email, input.Name)
	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	return customer, nil
}

// GetCustomer retrieves a customer by workspace ID
func (s *BillingService) GetCustomer(ctx context.Context, workspaceID string) (*model.Customer, error) {
	return s.customerRepo.FindByWorkspaceID(ctx, workspaceID)
}

// CreateSubscriptionInput represents subscription creation input
type CreateSubscriptionInput struct {
	WorkspaceID string
	PlanSlug    string
	BillingCycle string // monthly, yearly
}

// CreateSubscription creates a subscription
func (s *BillingService) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*model.Subscription, error) {
	// Get customer
	customer, err := s.customerRepo.FindByWorkspaceID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, model.ErrCustomerNotFound
	}

	// Get plan
	plan, err := s.planRepo.FindBySlug(ctx, input.PlanSlug)
	if err != nil {
		return nil, model.ErrPlanNotFound
	}

	// Determine price ID
	priceID := plan.MonthlyPriceID
	if input.BillingCycle == "yearly" {
		priceID = plan.YearlyPriceID
	}

	// Create in Stripe
	stripeSub, err := s.stripe.CreateSubscription(ctx, customer.StripeCustomerID, priceID, plan.TrialDays)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	// Create in database
	subscription := model.NewSubscription(input.WorkspaceID, plan.ID, stripeSub.ID, customer.StripeCustomerID)
	subscription.Status = model.SubscriptionStatus(stripeSub.Status)
	subscription.CurrentPeriodStart = stripeSub.CurrentPeriodStart
	subscription.CurrentPeriodEnd = stripeSub.CurrentPeriodEnd
	subscription.TrialStart = stripeSub.TrialStart
	subscription.TrialEnd = stripeSub.TrialEnd

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	return subscription, nil
}

// GetSubscription retrieves subscription for a workspace
func (s *BillingService) GetSubscription(ctx context.Context, workspaceID string) (*model.Subscription, error) {
	return s.subscriptionRepo.FindByWorkspaceID(ctx, workspaceID)
}

// CancelSubscription cancels a subscription
func (s *BillingService) CancelSubscription(ctx context.Context, workspaceID string, immediate bool) error {
	subscription, err := s.subscriptionRepo.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return model.ErrSubscriptionNotFound
	}

	// Cancel in Stripe
	if err := s.stripe.CancelSubscription(ctx, subscription.StripeSubscriptionID, !immediate); err != nil {
		return fmt.Errorf("failed to cancel Stripe subscription: %w", err)
	}

	// Update local
	subscription.Cancel()
	if immediate {
		subscription.Status = model.SubscriptionStatusCanceled
	}

	return s.subscriptionRepo.Update(ctx, subscription)
}

// ChangePlanInput represents plan change input
type ChangePlanInput struct {
	WorkspaceID  string
	NewPlanSlug  string
	BillingCycle string
}

// ChangePlan changes subscription plan
func (s *BillingService) ChangePlan(ctx context.Context, input ChangePlanInput) (*model.Subscription, error) {
	subscription, err := s.subscriptionRepo.FindByWorkspaceID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, model.ErrSubscriptionNotFound
	}

	// Get new plan
	newPlan, err := s.planRepo.FindBySlug(ctx, input.NewPlanSlug)
	if err != nil {
		return nil, model.ErrPlanNotFound
	}

	// Determine price ID
	priceID := newPlan.MonthlyPriceID
	if input.BillingCycle == "yearly" {
		priceID = newPlan.YearlyPriceID
	}

	// Update in Stripe
	params := map[string]interface{}{
		"items": []map[string]interface{}{
			{"price": priceID},
		},
		"proration_behavior": "create_prorations",
	}
	if err := s.stripe.UpdateSubscription(ctx, subscription.StripeSubscriptionID, params); err != nil {
		return nil, fmt.Errorf("failed to update Stripe subscription: %w", err)
	}

	// Update local
	subscription.PlanID = newPlan.ID
	subscription.UpdatedAt = time.Now()

	if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

// CreateCheckoutSession creates a Stripe checkout session
func (s *BillingService) CreateCheckoutSession(ctx context.Context, workspaceID, planSlug, successURL, cancelURL string) (string, error) {
	customer, err := s.customerRepo.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return "", model.ErrCustomerNotFound
	}

	plan, err := s.planRepo.FindBySlug(ctx, planSlug)
	if err != nil {
		return "", model.ErrPlanNotFound
	}

	session, err := s.stripe.CreateCheckoutSession(ctx, customer.StripeCustomerID, plan.MonthlyPriceID, successURL, cancelURL)
	if err != nil {
		return "", err
	}

	return session.URL, nil
}

// CreatePortalSession creates a Stripe billing portal session
func (s *BillingService) CreatePortalSession(ctx context.Context, workspaceID, returnURL string) (string, error) {
	customer, err := s.customerRepo.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return "", model.ErrCustomerNotFound
	}

	return s.stripe.CreatePortalSession(ctx, customer.StripeCustomerID, returnURL)
}

// ListPlans lists available plans
func (s *BillingService) ListPlans(ctx context.Context) ([]*model.Plan, error) {
	return s.planRepo.ListPublic(ctx)
}

// GetPlan retrieves a plan by slug
func (s *BillingService) GetPlan(ctx context.Context, slug string) (*model.Plan, error) {
	return s.planRepo.FindBySlug(ctx, slug)
}

// ListInvoices lists invoices for a workspace
func (s *BillingService) ListInvoices(ctx context.Context, workspaceID string, limit, offset int) ([]*model.Invoice, int64, error) {
	return s.invoiceRepo.ListByWorkspace(ctx, workspaceID, limit, offset)
}

// GetCurrentUsage retrieves current usage for a workspace
func (s *BillingService) GetCurrentUsage(ctx context.Context, workspaceID string) (*model.Usage, error) {
	return s.usageRepo.GetCurrentUsage(ctx, workspaceID)
}

// CheckLimit checks if workspace is within plan limits
func (s *BillingService) CheckLimit(ctx context.Context, workspaceID, limitType string) (bool, error) {
	subscription, err := s.subscriptionRepo.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		// No subscription, use free plan limits
		return s.checkFreePlanLimit(ctx, workspaceID, limitType)
	}

	plan, err := s.planRepo.FindByID(ctx, subscription.PlanID)
	if err != nil {
		return false, err
	}

	usage, err := s.usageRepo.GetCurrentUsage(ctx, workspaceID)
	if err != nil {
		return true, nil // Allow if can't get usage
	}

	return s.isWithinLimit(usage, &plan.Limits, limitType), nil
}

func (s *BillingService) checkFreePlanLimit(ctx context.Context, workspaceID, limitType string) (bool, error) {
	freeLimits := model.PlanLimits{
		MaxMembers:         3,
		MaxWorkflows:       5,
		MaxExecutionsMonth: 100,
		MaxCredentials:     5,
		MaxWebhooks:        2,
	}

	usage, err := s.usageRepo.GetCurrentUsage(ctx, workspaceID)
	if err != nil {
		return true, nil
	}

	return s.isWithinLimit(usage, &freeLimits, limitType), nil
}

func (s *BillingService) isWithinLimit(usage *model.Usage, limits *model.PlanLimits, limitType string) bool {
	switch limitType {
	case "members":
		return limits.MaxMembers == -1 || usage.ActiveMembers < limits.MaxMembers
	case "workflows":
		return limits.MaxWorkflows == -1 || usage.ActiveWorkflows < limits.MaxWorkflows
	case "executions":
		return limits.MaxExecutionsMonth == -1 || usage.ExecutionsCount < limits.MaxExecutionsMonth
	case "credentials":
		return limits.MaxCredentials == -1 || usage.CredentialsCount < limits.MaxCredentials
	case "webhooks":
		return limits.MaxWebhooks == -1 || usage.WebhooksCount < limits.MaxWebhooks
	default:
		return true
	}
}

// RecordExecution records a workflow execution for usage tracking
func (s *BillingService) RecordExecution(ctx context.Context, workspaceID string) error {
	return s.usageRepo.IncrementExecutions(ctx, workspaceID, 1)
}

// RecordAPICall records an API call for usage tracking
func (s *BillingService) RecordAPICall(ctx context.Context, workspaceID string) error {
	return s.usageRepo.IncrementAPICalls(ctx, workspaceID, 1)
}

// HandleWebhook processes Stripe webhook events
func (s *BillingService) HandleWebhook(ctx context.Context, event *WebhookEvent) error {
	switch event.Type {
	case model.EventSubscriptionCreated, model.EventSubscriptionUpdated:
		return s.handleSubscriptionUpdate(ctx, event)
	case model.EventSubscriptionDeleted:
		return s.handleSubscriptionDeleted(ctx, event)
	case model.EventInvoicePaid:
		return s.handleInvoicePaid(ctx, event)
	case model.EventInvoicePaymentFailed:
		return s.handlePaymentFailed(ctx, event)
	default:
		return nil
	}
}

func (s *BillingService) handleSubscriptionUpdate(ctx context.Context, event *WebhookEvent) error {
	data := event.Data
	stripeSubID, _ := data["id"].(string)

	subscription, err := s.subscriptionRepo.FindByStripeID(ctx, stripeSubID)
	if err != nil {
		return nil // Not our subscription
	}

	// Update from Stripe
	stripeSub, err := s.stripe.GetSubscription(ctx, stripeSubID)
	if err != nil {
		return err
	}

	subscription.Status = model.SubscriptionStatus(stripeSub.Status)
	subscription.CurrentPeriodStart = stripeSub.CurrentPeriodStart
	subscription.CurrentPeriodEnd = stripeSub.CurrentPeriodEnd
	subscription.CancelAtPeriodEnd = stripeSub.CancelAtPeriodEnd
	subscription.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *BillingService) handleSubscriptionDeleted(ctx context.Context, event *WebhookEvent) error {
	data := event.Data
	stripeSubID, _ := data["id"].(string)

	subscription, err := s.subscriptionRepo.FindByStripeID(ctx, stripeSubID)
	if err != nil {
		return nil
	}

	subscription.Status = model.SubscriptionStatusCanceled
	subscription.UpdatedAt = time.Now()

	return s.subscriptionRepo.Update(ctx, subscription)
}

func (s *BillingService) handleInvoicePaid(ctx context.Context, event *WebhookEvent) error {
	// Update invoice status
	return nil
}

func (s *BillingService) handlePaymentFailed(ctx context.Context, event *WebhookEvent) error {
	// Handle payment failure - notify user
	return nil
}

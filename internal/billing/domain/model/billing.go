// Package model defines billing domain models
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusTrialing        SubscriptionStatus = "trialing"
	SubscriptionStatusActive          SubscriptionStatus = "active"
	SubscriptionStatusPastDue         SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled        SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid          SubscriptionStatus = "unpaid"
	SubscriptionStatusIncomplete      SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
)

// Plan represents a billing plan
type Plan struct {
	ID              string
	Name            string
	Slug            string
	Description     string
	MonthlyPriceID  string // Stripe price ID
	YearlyPriceID   string // Stripe price ID
	MonthlyPrice    int64  // in cents
	YearlyPrice     int64  // in cents
	Currency        string
	Features        []string
	Limits          PlanLimits
	IsPublic        bool
	TrialDays       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PlanLimits defines plan resource limits
type PlanLimits struct {
	MaxMembers         int   `json:"maxMembers"`
	MaxWorkflows       int   `json:"maxWorkflows"`
	MaxExecutionsMonth int   `json:"maxExecutionsPerMonth"`
	MaxCredentials     int   `json:"maxCredentials"`
	MaxWebhooks        int   `json:"maxWebhooks"`
	MaxStorageGB       int   `json:"maxStorageGB"`
	RetentionDays      int   `json:"retentionDays"`
	SupportLevel       string `json:"supportLevel"` // community, email, priority, dedicated
}

// Subscription represents a workspace subscription
type Subscription struct {
	ID                   string
	WorkspaceID          string
	PlanID               string
	StripeSubscriptionID string
	StripeCustomerID     string
	Status               SubscriptionStatus
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	CancelAtPeriodEnd    bool
	CanceledAt           *time.Time
	TrialStart           *time.Time
	TrialEnd             *time.Time
	Quantity             int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// NewSubscription creates a new subscription
func NewSubscription(workspaceID, planID, stripeSubID, stripeCustomerID string) *Subscription {
	now := time.Now()
	return &Subscription{
		ID:                   uuid.New().String(),
		WorkspaceID:          workspaceID,
		PlanID:               planID,
		StripeSubscriptionID: stripeSubID,
		StripeCustomerID:     stripeCustomerID,
		Status:               SubscriptionStatusActive,
		Quantity:             1,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// IsActive checks if subscription is active
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// IsInTrial checks if subscription is in trial
func (s *Subscription) IsInTrial() bool {
	return s.Status == SubscriptionStatusTrialing
}

// Cancel marks subscription for cancellation
func (s *Subscription) Cancel() {
	s.CancelAtPeriodEnd = true
	now := time.Now()
	s.CanceledAt = &now
	s.UpdatedAt = now
}

// Customer represents a billing customer
type Customer struct {
	ID               string
	WorkspaceID      string
	StripeCustomerID string
	Email            string
	Name             string
	PaymentMethodID  string
	DefaultCurrency  string
	TaxID            string
	Address          *Address
	Metadata         map[string]string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Address represents a billing address
type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// NewCustomer creates a new customer
func NewCustomer(workspaceID, stripeCustomerID, email, name string) *Customer {
	now := time.Now()
	return &Customer{
		ID:               uuid.New().String(),
		WorkspaceID:      workspaceID,
		StripeCustomerID: stripeCustomerID,
		Email:            email,
		Name:             name,
		DefaultCurrency:  "usd",
		Metadata:         make(map[string]string),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// InvoiceStatus represents invoice status
type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusVoid          InvoiceStatus = "void"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
)

// Invoice represents a billing invoice
type Invoice struct {
	ID               string
	WorkspaceID      string
	StripeInvoiceID  string
	Number           string
	Status           InvoiceStatus
	Currency         string
	Subtotal         int64 // in cents
	Tax              int64
	Total            int64
	AmountPaid       int64
	AmountDue        int64
	LineItems        []LineItem
	PeriodStart      time.Time
	PeriodEnd        time.Time
	DueDate          *time.Time
	PaidAt           *time.Time
	HostedInvoiceURL string
	InvoicePDFURL    string
	CreatedAt        time.Time
}

// LineItem represents an invoice line item
type LineItem struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitAmount  int64  `json:"unitAmount"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
}

// PaymentMethod represents a payment method
type PaymentMethod struct {
	ID                    string
	WorkspaceID           string
	StripePaymentMethodID string
	Type                  string // card, bank_account, etc.
	IsDefault             bool
	Card                  *CardDetails
	CreatedAt             time.Time
}

// CardDetails represents card payment method details
type CardDetails struct {
	Brand       string `json:"brand"`
	Last4       string `json:"last4"`
	ExpMonth    int    `json:"expMonth"`
	ExpYear     int    `json:"expYear"`
	Fingerprint string `json:"fingerprint"`
}

// Usage represents resource usage for a workspace
type Usage struct {
	ID                  string
	WorkspaceID         string
	Period              time.Time // First day of the month
	ExecutionsCount     int
	APICallsCount       int
	StorageUsedBytes    int64
	ActiveWorkflows     int
	ActiveMembers       int
	WebhooksCount       int
	CredentialsCount    int
	LastUpdated         time.Time
}

// NewUsage creates a new usage record
func NewUsage(workspaceID string, period time.Time) *Usage {
	return &Usage{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Period:      period,
		LastUpdated: time.Now(),
	}
}

// UsageEvent represents a usage event for tracking
type UsageEvent struct {
	ID          string
	WorkspaceID string
	EventType   string // execution, api_call, storage, etc.
	Quantity    int64
	Metadata    map[string]interface{}
	Timestamp   time.Time
}

// Errors
var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrCustomerNotFound     = errors.New("customer not found")
	ErrPlanNotFound         = errors.New("plan not found")
	ErrInvoiceNotFound      = errors.New("invoice not found")
	ErrPaymentFailed        = errors.New("payment failed")
	ErrSubscriptionCanceled = errors.New("subscription is canceled")
	ErrLimitExceeded        = errors.New("plan limit exceeded")
)

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	ID        string
	Type      string
	Data      map[string]interface{}
	Processed bool
	Error     string
	CreatedAt time.Time
}

// Common Stripe webhook event types
const (
	EventCustomerCreated           = "customer.created"
	EventCustomerUpdated           = "customer.updated"
	EventCustomerDeleted           = "customer.deleted"
	EventSubscriptionCreated       = "customer.subscription.created"
	EventSubscriptionUpdated       = "customer.subscription.updated"
	EventSubscriptionDeleted       = "customer.subscription.deleted"
	EventSubscriptionTrialWillEnd  = "customer.subscription.trial_will_end"
	EventInvoiceCreated            = "invoice.created"
	EventInvoicePaid               = "invoice.paid"
	EventInvoicePaymentFailed      = "invoice.payment_failed"
	EventPaymentIntentSucceeded    = "payment_intent.succeeded"
	EventPaymentIntentFailed       = "payment_intent.payment_failed"
	EventPaymentMethodAttached     = "payment_method.attached"
	EventPaymentMethodDetached     = "payment_method.detached"
)

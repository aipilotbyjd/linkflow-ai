package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Plan represents a subscription plan
type Plan string

const (
	PlanFree       Plan = "free"
	PlanStarter    Plan = "starter"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

// TenantStatus represents the status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusCancelled TenantStatus = "cancelled"
	TenantStatusTrial     TenantStatus = "trial"
)

// Tenant represents a tenant/organization in the system
type Tenant struct {
	ID             string
	Name           string
	Slug           string
	OwnerID        string
	Plan           Plan
	Status         TenantStatus
	Settings       TenantSettings
	Limits         ResourceLimits
	BillingInfo    *BillingInfo
	TrialEndsAt    *time.Time
	SubscriptionID *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TenantSettings holds tenant-specific settings
type TenantSettings struct {
	AllowedDomains    []string          `json:"allowedDomains"`
	SSOEnabled        bool              `json:"ssoEnabled"`
	SSOProvider       string            `json:"ssoProvider"`
	SSOConfig         map[string]string `json:"ssoConfig"`
	DefaultTimezone   string            `json:"defaultTimezone"`
	DefaultLanguage   string            `json:"defaultLanguage"`
	NotifyOnNewMember bool              `json:"notifyOnNewMember"`
	AuditLogRetention int               `json:"auditLogRetentionDays"`
	CustomBranding    *Branding         `json:"customBranding"`
}

// Branding holds custom branding settings
type Branding struct {
	LogoURL       string `json:"logoUrl"`
	PrimaryColor  string `json:"primaryColor"`
	AccentColor   string `json:"accentColor"`
	FaviconURL    string `json:"faviconUrl"`
	CustomCSS     string `json:"customCss"`
}

// ResourceLimits defines resource limits for a tenant
type ResourceLimits struct {
	MaxUsers           int   `json:"maxUsers"`
	MaxWorkflows       int   `json:"maxWorkflows"`
	MaxExecutionsMonth int   `json:"maxExecutionsPerMonth"`
	MaxStorageGB       int   `json:"maxStorageGB"`
	MaxAPICallsMonth   int   `json:"maxApiCallsPerMonth"`
	MaxWebhooks        int   `json:"maxWebhooks"`
	MaxSchedules       int   `json:"maxSchedules"`
	MaxIntegrations    int   `json:"maxIntegrations"`
	RetentionDays      int   `json:"retentionDays"`
}

// GetPlanLimits returns the resource limits for a plan
func GetPlanLimits(plan Plan) ResourceLimits {
	switch plan {
	case PlanFree:
		return ResourceLimits{
			MaxUsers:           3,
			MaxWorkflows:       5,
			MaxExecutionsMonth: 100,
			MaxStorageGB:       1,
			MaxAPICallsMonth:   1000,
			MaxWebhooks:        2,
			MaxSchedules:       2,
			MaxIntegrations:    3,
			RetentionDays:      7,
		}
	case PlanStarter:
		return ResourceLimits{
			MaxUsers:           10,
			MaxWorkflows:       25,
			MaxExecutionsMonth: 1000,
			MaxStorageGB:       10,
			MaxAPICallsMonth:   10000,
			MaxWebhooks:        10,
			MaxSchedules:       10,
			MaxIntegrations:    10,
			RetentionDays:      30,
		}
	case PlanPro:
		return ResourceLimits{
			MaxUsers:           50,
			MaxWorkflows:       100,
			MaxExecutionsMonth: 10000,
			MaxStorageGB:       100,
			MaxAPICallsMonth:   100000,
			MaxWebhooks:        50,
			MaxSchedules:       50,
			MaxIntegrations:    50,
			RetentionDays:      90,
		}
	case PlanEnterprise:
		return ResourceLimits{
			MaxUsers:           -1, // Unlimited
			MaxWorkflows:       -1,
			MaxExecutionsMonth: -1,
			MaxStorageGB:       -1,
			MaxAPICallsMonth:   -1,
			MaxWebhooks:        -1,
			MaxSchedules:       -1,
			MaxIntegrations:    -1,
			RetentionDays:      365,
		}
	default:
		return GetPlanLimits(PlanFree)
	}
}

// NewTenant creates a new tenant
func NewTenant(name, slug, ownerID string, plan Plan) (*Tenant, error) {
	if name == "" {
		return nil, errors.New("tenant name is required")
	}
	if slug == "" {
		return nil, errors.New("tenant slug is required")
	}
	if ownerID == "" {
		return nil, errors.New("owner ID is required")
	}

	now := time.Now()
	trialEnd := now.Add(14 * 24 * time.Hour) // 14 day trial

	return &Tenant{
		ID:          uuid.New().String(),
		Name:        name,
		Slug:        slug,
		OwnerID:     ownerID,
		Plan:        plan,
		Status:      TenantStatusTrial,
		Limits:      GetPlanLimits(plan),
		TrialEndsAt: &trialEnd,
		Settings: TenantSettings{
			DefaultTimezone:   "UTC",
			DefaultLanguage:   "en",
			NotifyOnNewMember: true,
			AuditLogRetention: 30,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Activate activates the tenant
func (t *Tenant) Activate() error {
	if t.Status == TenantStatusCancelled {
		return errors.New("cannot activate cancelled tenant")
	}
	t.Status = TenantStatusActive
	t.UpdatedAt = time.Now()
	return nil
}

// Suspend suspends the tenant
func (t *Tenant) Suspend(reason string) {
	t.Status = TenantStatusSuspended
	t.UpdatedAt = time.Now()
}

// Cancel cancels the tenant subscription
func (t *Tenant) Cancel() {
	t.Status = TenantStatusCancelled
	t.UpdatedAt = time.Now()
}

// UpgradePlan upgrades the tenant to a new plan
func (t *Tenant) UpgradePlan(newPlan Plan) error {
	if t.Status == TenantStatusCancelled {
		return errors.New("cannot upgrade cancelled tenant")
	}
	t.Plan = newPlan
	t.Limits = GetPlanLimits(newPlan)
	t.UpdatedAt = time.Now()
	return nil
}

// IsWithinLimits checks if usage is within plan limits
func (t *Tenant) IsWithinLimits(usage ResourceUsage) bool {
	if t.Limits.MaxUsers != -1 && usage.Users > t.Limits.MaxUsers {
		return false
	}
	if t.Limits.MaxWorkflows != -1 && usage.Workflows > t.Limits.MaxWorkflows {
		return false
	}
	if t.Limits.MaxExecutionsMonth != -1 && usage.ExecutionsThisMonth > t.Limits.MaxExecutionsMonth {
		return false
	}
	return true
}

// ResourceUsage tracks current resource usage
type ResourceUsage struct {
	Users               int       `json:"users"`
	Workflows           int       `json:"workflows"`
	ExecutionsThisMonth int       `json:"executionsThisMonth"`
	StorageUsedGB       float64   `json:"storageUsedGB"`
	APICallsThisMonth   int       `json:"apiCallsThisMonth"`
	LastUpdated         time.Time `json:"lastUpdated"`
}

// BillingInfo holds billing information
type BillingInfo struct {
	CustomerID        string         `json:"customerId"`
	PaymentMethodID   string         `json:"paymentMethodId"`
	BillingEmail      string         `json:"billingEmail"`
	BillingAddress    *Address       `json:"billingAddress"`
	TaxID             string         `json:"taxId"`
	Currency          string         `json:"currency"`
	BillingCycle      string         `json:"billingCycle"` // monthly, yearly
	NextBillingDate   *time.Time     `json:"nextBillingDate"`
	CurrentPeriodEnd  *time.Time     `json:"currentPeriodEnd"`
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

// Invoice represents a billing invoice
type Invoice struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenantId"`
	Number        string        `json:"number"`
	Status        InvoiceStatus `json:"status"`
	Currency      string        `json:"currency"`
	Subtotal      int64         `json:"subtotal"`
	Tax           int64         `json:"tax"`
	Total         int64         `json:"total"`
	AmountPaid    int64         `json:"amountPaid"`
	AmountDue     int64         `json:"amountDue"`
	LineItems     []LineItem    `json:"lineItems"`
	PeriodStart   time.Time     `json:"periodStart"`
	PeriodEnd     time.Time     `json:"periodEnd"`
	DueDate       time.Time     `json:"dueDate"`
	PaidAt        *time.Time    `json:"paidAt"`
	InvoicePDF    string        `json:"invoicePdf"`
	CreatedAt     time.Time     `json:"createdAt"`
}

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusOpen      InvoiceStatus = "open"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusVoid      InvoiceStatus = "void"
	InvoiceStatusUncollect InvoiceStatus = "uncollectible"
)

// LineItem represents an invoice line item
type LineItem struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unitPrice"`
	Amount      int64  `json:"amount"`
}

// Subscription represents a subscription
type Subscription struct {
	ID                 string             `json:"id"`
	TenantID           string             `json:"tenantId"`
	Plan               Plan               `json:"plan"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time          `json:"currentPeriodStart"`
	CurrentPeriodEnd   time.Time          `json:"currentPeriodEnd"`
	CancelAtPeriodEnd  bool               `json:"cancelAtPeriodEnd"`
	CancelledAt        *time.Time         `json:"cancelledAt"`
	TrialStart         *time.Time         `json:"trialStart"`
	TrialEnd           *time.Time         `json:"trialEnd"`
	CreatedAt          time.Time          `json:"createdAt"`
	UpdatedAt          time.Time          `json:"updatedAt"`
}

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusTrialing       SubscriptionStatus = "trialing"
	SubscriptionStatusActive         SubscriptionStatus = "active"
	SubscriptionStatusPastDue        SubscriptionStatus = "past_due"
	SubscriptionStatusCancelled      SubscriptionStatus = "cancelled"
	SubscriptionStatusIncomplete     SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExp  SubscriptionStatus = "incomplete_expired"
)

// FeatureFlag represents a feature flag for a tenant
type FeatureFlag struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenantId"`
	Key         string                 `json:"key"`
	Enabled     bool                   `json:"enabled"`
	Value       interface{}            `json:"value"`
	Conditions  map[string]interface{} `json:"conditions"`
	Description string                 `json:"description"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

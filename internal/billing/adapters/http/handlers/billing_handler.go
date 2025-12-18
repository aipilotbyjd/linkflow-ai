// Package handlers provides HTTP handlers for billing service
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/linkflow-ai/linkflow-ai/internal/billing/app/service"
)

// BillingHandler handles billing HTTP requests
type BillingHandler struct {
	billingService *service.BillingService
	webhookSecret  string
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingService *service.BillingService, webhookSecret string) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
		webhookSecret:  webhookSecret,
	}
}

// RegisterRoutes registers billing routes
func (h *BillingHandler) RegisterRoutes(mux *http.ServeMux) {
	// Plans
	mux.HandleFunc("/api/v1/billing/plans", h.listPlans)
	mux.HandleFunc("/api/v1/billing/plans/", h.getPlan)
	
	// Subscription management
	mux.HandleFunc("/api/v1/billing/subscription", h.handleSubscription)
	mux.HandleFunc("/api/v1/billing/subscription/cancel", h.cancelSubscription)
	mux.HandleFunc("/api/v1/billing/subscription/change", h.changePlan)
	
	// Checkout & portal
	mux.HandleFunc("/api/v1/billing/checkout", h.createCheckout)
	mux.HandleFunc("/api/v1/billing/portal", h.createPortal)
	
	// Invoices
	mux.HandleFunc("/api/v1/billing/invoices", h.listInvoices)
	
	// Usage
	mux.HandleFunc("/api/v1/billing/usage", h.getUsage)
	
	// Customer
	mux.HandleFunc("/api/v1/billing/customer", h.handleCustomer)
	
	// Webhook
	mux.HandleFunc("/api/v1/billing/webhook", h.handleWebhook)
}

// Plan responses
type PlanResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	MonthlyPrice int64    `json:"monthlyPrice"`
	YearlyPrice  int64    `json:"yearlyPrice"`
	Currency     string   `json:"currency"`
	Features     []string `json:"features"`
	Limits       interface{} `json:"limits"`
}

func (h *BillingHandler) listPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	plans, err := h.billingService.ListPlans(r.Context())
	if err != nil {
		writeError(w, "Failed to list plans", http.StatusInternalServerError)
		return
	}

	response := make([]PlanResponse, len(plans))
	for i, p := range plans {
		response[i] = PlanResponse{
			ID:           p.ID,
			Name:         p.Name,
			Slug:         p.Slug,
			Description:  p.Description,
			MonthlyPrice: p.MonthlyPrice,
			YearlyPrice:  p.YearlyPrice,
			Currency:     p.Currency,
			Features:     p.Features,
			Limits:       p.Limits,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": response,
	})
}

func (h *BillingHandler) getPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/api/v1/billing/plans/")
	
	plan, err := h.billingService.GetPlan(r.Context(), slug)
	if err != nil {
		writeError(w, "Plan not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, PlanResponse{
		ID:           plan.ID,
		Name:         plan.Name,
		Slug:         plan.Slug,
		Description:  plan.Description,
		MonthlyPrice: plan.MonthlyPrice,
		YearlyPrice:  plan.YearlyPrice,
		Currency:     plan.Currency,
		Features:     plan.Features,
		Limits:       plan.Limits,
	})
}

func (h *BillingHandler) handleSubscription(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getSubscription(w, r, workspaceID)
	case http.MethodPost:
		h.createSubscription(w, r, workspaceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubscriptionResponse represents subscription response
type SubscriptionResponse struct {
	ID                 string  `json:"id"`
	PlanID             string  `json:"planId"`
	Status             string  `json:"status"`
	CurrentPeriodStart string  `json:"currentPeriodStart"`
	CurrentPeriodEnd   string  `json:"currentPeriodEnd"`
	CancelAtPeriodEnd  bool    `json:"cancelAtPeriodEnd"`
	TrialEnd           *string `json:"trialEnd,omitempty"`
}

func (h *BillingHandler) getSubscription(w http.ResponseWriter, r *http.Request, workspaceID string) {
	subscription, err := h.billingService.GetSubscription(r.Context(), workspaceID)
	if err != nil {
		writeError(w, "No active subscription", http.StatusNotFound)
		return
	}

	var trialEnd *string
	if subscription.TrialEnd != nil {
		t := subscription.TrialEnd.Format("2006-01-02T15:04:05Z")
		trialEnd = &t
	}

	writeJSON(w, http.StatusOK, SubscriptionResponse{
		ID:                 subscription.ID,
		PlanID:             subscription.PlanID,
		Status:             string(subscription.Status),
		CurrentPeriodStart: subscription.CurrentPeriodStart.Format("2006-01-02T15:04:05Z"),
		CurrentPeriodEnd:   subscription.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z"),
		CancelAtPeriodEnd:  subscription.CancelAtPeriodEnd,
		TrialEnd:           trialEnd,
	})
}

// CreateSubscriptionRequest represents subscription creation request
type CreateSubscriptionRequest struct {
	PlanSlug     string `json:"planSlug"`
	BillingCycle string `json:"billingCycle"` // monthly, yearly
}

func (h *BillingHandler) createSubscription(w http.ResponseWriter, r *http.Request, workspaceID string) {
	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PlanSlug == "" {
		writeError(w, "Plan slug is required", http.StatusBadRequest)
		return
	}

	if req.BillingCycle == "" {
		req.BillingCycle = "monthly"
	}

	subscription, err := h.billingService.CreateSubscription(r.Context(), service.CreateSubscriptionInput{
		WorkspaceID:  workspaceID,
		PlanSlug:     req.PlanSlug,
		BillingCycle: req.BillingCycle,
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, SubscriptionResponse{
		ID:                 subscription.ID,
		PlanID:             subscription.PlanID,
		Status:             string(subscription.Status),
		CurrentPeriodStart: subscription.CurrentPeriodStart.Format("2006-01-02T15:04:05Z"),
		CurrentPeriodEnd:   subscription.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *BillingHandler) cancelSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Immediate bool `json:"immediate"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.billingService.CancelSubscription(r.Context(), workspaceID, req.Immediate); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Subscription canceled"})
}

// ChangePlanRequest represents plan change request
type ChangePlanRequest struct {
	PlanSlug     string `json:"planSlug"`
	BillingCycle string `json:"billingCycle"`
}

func (h *BillingHandler) changePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	var req ChangePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	subscription, err := h.billingService.ChangePlan(r.Context(), service.ChangePlanInput{
		WorkspaceID:  workspaceID,
		NewPlanSlug:  req.PlanSlug,
		BillingCycle: req.BillingCycle,
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, SubscriptionResponse{
		ID:                 subscription.ID,
		PlanID:             subscription.PlanID,
		Status:             string(subscription.Status),
		CurrentPeriodStart: subscription.CurrentPeriodStart.Format("2006-01-02T15:04:05Z"),
		CurrentPeriodEnd:   subscription.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z"),
	})
}

// CreateCheckoutRequest represents checkout session request
type CreateCheckoutRequest struct {
	PlanSlug   string `json:"planSlug"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
}

func (h *BillingHandler) createCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	var req CreateCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	url, err := h.billingService.CreateCheckoutSession(r.Context(), workspaceID, req.PlanSlug, req.SuccessURL, req.CancelURL)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// CreatePortalRequest represents portal session request
type CreatePortalRequest struct {
	ReturnURL string `json:"returnUrl"`
}

func (h *BillingHandler) createPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	var req CreatePortalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	url, err := h.billingService.CreatePortalSession(r.Context(), workspaceID, req.ReturnURL)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *BillingHandler) listInvoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit == 0 {
		limit = 20
	}

	invoices, total, err := h.billingService.ListInvoices(r.Context(), workspaceID, limit, offset)
	if err != nil {
		writeError(w, "Failed to list invoices", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(invoices))
	for i, inv := range invoices {
		response[i] = map[string]interface{}{
			"id":         inv.ID,
			"number":     inv.Number,
			"status":     inv.Status,
			"total":      inv.Total,
			"currency":   inv.Currency,
			"periodStart": inv.PeriodStart.Format("2006-01-02"),
			"periodEnd":   inv.PeriodEnd.Format("2006-01-02"),
			"pdfUrl":     inv.InvoicePDFURL,
			"createdAt":  inv.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": response,
		"total": total,
	})
}

// UsageResponse represents usage response
type UsageResponse struct {
	ExecutionsCount  int   `json:"executionsCount"`
	APICallsCount    int   `json:"apiCallsCount"`
	StorageUsedBytes int64 `json:"storageUsedBytes"`
	ActiveWorkflows  int   `json:"activeWorkflows"`
	ActiveMembers    int   `json:"activeMembers"`
	Period           string `json:"period"`
}

func (h *BillingHandler) getUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	usage, err := h.billingService.GetCurrentUsage(r.Context(), workspaceID)
	if err != nil {
		writeError(w, "Failed to get usage", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, UsageResponse{
		ExecutionsCount:  usage.ExecutionsCount,
		APICallsCount:    usage.APICallsCount,
		StorageUsedBytes: usage.StorageUsedBytes,
		ActiveWorkflows:  usage.ActiveWorkflows,
		ActiveMembers:    usage.ActiveMembers,
		Period:           usage.Period.Format("2006-01"),
	})
}

func (h *BillingHandler) handleCustomer(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getCustomer(w, r, workspaceID)
	case http.MethodPost:
		h.createCustomer(w, r, workspaceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BillingHandler) getCustomer(w http.ResponseWriter, r *http.Request, workspaceID string) {
	customer, err := h.billingService.GetCustomer(r.Context(), workspaceID)
	if err != nil {
		writeError(w, "Customer not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":        customer.ID,
		"email":     customer.Email,
		"name":      customer.Name,
		"createdAt": customer.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// CreateCustomerRequest represents customer creation request
type CreateCustomerRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (h *BillingHandler) createCustomer(w http.ResponseWriter, r *http.Request, workspaceID string) {
	var req CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	customer, err := h.billingService.CreateCustomer(r.Context(), service.CreateCustomerInput{
		WorkspaceID: workspaceID,
		Email:       req.Email,
		Name:        req.Name,
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        customer.ID,
		"email":     customer.Email,
		"name":      customer.Name,
		"createdAt": customer.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *BillingHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// In production, verify webhook signature using h.webhookSecret
	// For now, just parse the event
	var event service.WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		writeError(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	if err := h.billingService.HandleWebhook(r.Context(), &event); err != nil {
		writeError(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}

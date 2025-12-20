package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/tenant/domain/model"
)

const (
	serviceName = "tenant-service"
	servicePort = 8019
)

type Server struct {
	tenants  map[string]*model.Tenant
	invoices map[string]*model.Invoice
	mu       sync.RWMutex
	logger   logger.Logger
}

func main() {
	log := logger.New(config.LoggerConfig{Level: "info", Format: "json", OutputPath: "stdout"})
	log.Info("Starting Tenant Service", "port", servicePort)

	srv := &Server{
		tenants:  make(map[string]*model.Tenant),
		invoices: make(map[string]*model.Invoice),
		logger:   log,
	}

	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", servicePort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/tenants", s.handleTenants)
	mux.HandleFunc("/api/v1/tenants/", s.handleTenantByID)
	mux.HandleFunc("/api/v1/billing/invoices", s.handleInvoices)
	mux.HandleFunc("/api/v1/billing/subscriptions", s.handleSubscriptions)
	mux.HandleFunc("/api/v1/plans", s.handlePlans)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "healthy", "service": serviceName})
}

func (s *Server) handleTenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name    string     `json:"name"`
			Slug    string     `json:"slug"`
			OwnerID string     `json:"ownerId"`
			Plan    model.Plan `json:"plan"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tenant, err := model.NewTenant(req.Name, req.Slug, req.OwnerID, req.Plan)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.tenants[tenant.ID] = tenant
		s.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(tenant)
	case http.MethodGet:
		s.mu.RLock()
		tenants := make([]*model.Tenant, 0, len(s.tenants))
		for _, t := range s.tenants {
			tenants = append(tenants, t)
		}
		s.mu.RUnlock()
		json.NewEncoder(w).Encode(map[string]interface{}{"tenants": tenants})
	}
}

func (s *Server) handleTenantByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/tenants/"):]
	s.mu.RLock()
	tenant, ok := s.tenants[id]
	s.mu.RUnlock()
	if !ok {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(tenant)
	case http.MethodPut:
		var req struct {
			Plan model.Plan `json:"plan"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Plan != "" {
			tenant.UpgradePlan(req.Plan)
		}
		json.NewEncoder(w).Encode(tenant)
	case http.MethodDelete:
		tenant.Cancel()
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleInvoices(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	invoices := make([]*model.Invoice, 0, len(s.invoices))
	for _, inv := range s.invoices {
		invoices = append(invoices, inv)
	}
	s.mu.RUnlock()
	json.NewEncoder(w).Encode(map[string]interface{}{"invoices": invoices})
}

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{"subscriptions": []interface{}{}})
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	plans := []map[string]interface{}{
		{"id": "free", "name": "Free", "price": 0, "limits": model.GetPlanLimits(model.PlanFree)},
		{"id": "starter", "name": "Starter", "price": 29, "limits": model.GetPlanLimits(model.PlanStarter)},
		{"id": "pro", "name": "Pro", "price": 99, "limits": model.GetPlanLimits(model.PlanPro)},
		{"id": "enterprise", "name": "Enterprise", "price": -1, "limits": model.GetPlanLimits(model.PlanEnterprise)},
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"plans": plans})
}

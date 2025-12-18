package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/analytics/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

type AnalyticsHandler struct {
	service *service.AnalyticsService
	logger  logger.Logger
}

func NewAnalyticsHandler(service *service.AnalyticsService, logger logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
		logger:  logger,
	}
}

func (h *AnalyticsHandler) RegisterRoutes(router interface{}) {
	r := router.(*http.ServeMux)
	r.HandleFunc("/api/v1/analytics/track", h.TrackEvent)
	r.HandleFunc("/api/v1/analytics/metrics", h.GetMetrics)
	r.HandleFunc("/api/v1/analytics/user", h.GetUserAnalytics)
}

func (h *AnalyticsHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var cmd service.TrackEventCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd.IP = r.RemoteAddr
	cmd.UserAgent = r.UserAgent()

	if err := h.service.TrackEvent(ctx, cmd); err != nil {
		h.logger.Error("Failed to track event", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to track event")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "tracked"})
}

func (h *AnalyticsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := service.GetMetricsQuery{
		StartDate: time.Now().AddDate(0, 0, -7),
		EndDate:   time.Now(),
	}

	metrics, err := h.service.GetMetrics(ctx, query)
	if err != nil {
		h.logger.Error("Failed to get metrics", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to get metrics")
		return
	}

	h.respondJSON(w, http.StatusOK, metrics)
}

func (h *AnalyticsHandler) GetUserAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.URL.Query().Get("user_id")

	query := service.GetUserAnalyticsQuery{
		UserID:    userID,
		StartDate: time.Now().AddDate(0, 0, -30),
		EndDate:   time.Now(),
	}

	analytics, err := h.service.GetUserAnalytics(ctx, query)
	if err != nil {
		h.logger.Error("Failed to get user analytics", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to get user analytics")
		return
	}

	h.respondJSON(w, http.StatusOK, analytics)
}

func (h *AnalyticsHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *AnalyticsHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

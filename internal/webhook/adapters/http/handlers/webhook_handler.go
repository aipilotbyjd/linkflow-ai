package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/webhook/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/webhook/domain/model"
)

type WebhookHandler struct {
	service *service.WebhookService
	logger  logger.Logger
}

func NewWebhookHandler(service *service.WebhookService, logger logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		service: service,
		logger:  logger,
	}
}

func (h *WebhookHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/webhooks", h.CreateWebhook).Methods("POST")
	router.HandleFunc("/webhooks/{id}", h.GetWebhook).Methods("GET")
	router.HandleFunc("/webhooks/{id}/trigger", h.TriggerWebhook).Methods("POST")
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.CreateWebhookCommand
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	webhook, err := h.service.CreateWebhook(ctx, req)
	if err != nil {
		h.logger.Error("Failed to create webhook", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create webhook")
		return
	}

	h.respondJSON(w, http.StatusCreated, webhook)
}

func (h *WebhookHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	webhookID := vars["id"]

	webhook, err := h.service.GetWebhook(ctx, model.WebhookID(webhookID))
	if err != nil {
		h.respondError(w, http.StatusNotFound, "webhook not found")
		return
	}

	h.respondJSON(w, http.StatusOK, webhook)
}

func (h *WebhookHandler) TriggerWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	webhookID := vars["id"]

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		payload = make(map[string]interface{})
	}

	err := h.service.TriggerWebhook(ctx, model.WebhookID(webhookID), payload)
	if err != nil {
		h.logger.Error("Failed to trigger webhook", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to trigger webhook")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "triggered"})
}

func (h *WebhookHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *WebhookHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

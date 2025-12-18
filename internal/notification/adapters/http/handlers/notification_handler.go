package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/notification/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/middleware"
)

type NotificationHandler struct {
	service *service.NotificationService
	logger  logger.Logger
}

func NewNotificationHandler(service *service.NotificationService, logger logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		service: service,
		logger:  logger,
	}
}

func (h *NotificationHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/notifications", h.CreateNotification).Methods("POST")
	router.HandleFunc("/notifications", h.ListNotifications).Methods("GET")
	router.HandleFunc("/notifications/{id}", h.GetNotification).Methods("GET")
	router.HandleFunc("/notifications/{id}/read", h.MarkAsRead).Methods("POST")
}

func (h *NotificationHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req service.CreateNotificationCommand
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	notification, err := h.service.CreateNotification(ctx, req)
	if err != nil {
		h.logger.Error("Failed to create notification", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create notification")
		return
	}

	h.respondJSON(w, http.StatusCreated, notification)
}

func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, _ := middleware.ExtractUserID(ctx)
	vars := mux.Vars(r)
	notificationID := vars["id"]

	notification, err := h.service.GetNotification(ctx, model.NotificationID(notificationID), userID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "notification not found")
		return
	}

	h.respondJSON(w, http.StatusOK, notification)
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, _ := middleware.ExtractUserID(ctx)

	notifications, total, err := h.service.ListNotifications(ctx, service.ListNotificationsQuery{
		UserID: userID,
		Offset: 0,
		Limit:  20,
	})
	if err != nil {
		h.logger.Error("Failed to list notifications", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}

	resp := map[string]interface{}{
		"items": notifications,
		"total": total,
	}
	h.respondJSON(w, http.StatusOK, resp)
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, _ := middleware.ExtractUserID(ctx)
	vars := mux.Vars(r)
	notificationID := vars["id"]

	err := h.service.MarkAsRead(ctx, model.NotificationID(notificationID), userID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "marked as read"})
}

func (h *NotificationHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *NotificationHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

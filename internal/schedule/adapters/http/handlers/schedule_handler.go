package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/middleware"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/adapters/http/dto"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/domain/model"
)

// ScheduleHandler handles HTTP requests for schedules
type ScheduleHandler struct {
	service *service.ScheduleService
	logger  logger.Logger
}

// NewScheduleHandler creates a new schedule handler
func NewScheduleHandler(service *service.ScheduleService, logger logger.Logger) *ScheduleHandler {
	return &ScheduleHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers schedule routes
func (h *ScheduleHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/schedules", h.CreateSchedule).Methods("POST")
	router.HandleFunc("/schedules", h.ListSchedules).Methods("GET")
	router.HandleFunc("/schedules/{id}", h.GetSchedule).Methods("GET")
	router.HandleFunc("/schedules/{id}", h.UpdateSchedule).Methods("PUT")
	router.HandleFunc("/schedules/{id}", h.DeleteSchedule).Methods("DELETE")
	router.HandleFunc("/schedules/{id}/pause", h.PauseSchedule).Methods("POST")
	router.HandleFunc("/schedules/{id}/resume", h.ResumeSchedule).Methods("POST")
	router.HandleFunc("/schedules/{id}/execute", h.ExecuteSchedule).Methods("POST")
	router.HandleFunc("/workflows/{workflowId}/schedules", h.ListWorkflowSchedules).Methods("GET")
	router.HandleFunc("/schedules/validate-cron", h.ValidateCron).Methods("POST")
}

// CreateSchedule creates a new schedule
func (h *ScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse request
	var req dto.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse dates
	var startDate, endDate *time.Time
	if req.StartDate != "" {
		t, err := time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid start date format")
			return
		}
		startDate = &t
	}
	if req.EndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid end date format")
			return
		}
		endDate = &t
	}

	// Create schedule
	schedule, err := h.service.CreateSchedule(ctx, service.CreateScheduleCommand{
		UserID:         userID,
		OrganizationID: req.OrganizationID,
		WorkflowID:     req.WorkflowID,
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		StartDate:      startDate,
		EndDate:        endDate,
		InputData:      req.InputData,
	})
	if err != nil {
		h.logger.Error("Failed to create schedule", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create schedule")
		return
	}

	// Convert to response DTO
	resp := h.scheduleToDTO(schedule)
	h.respondJSON(w, http.StatusCreated, resp)
}

// GetSchedule gets a schedule by ID
func (h *ScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get schedule ID from path
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Get schedule
	schedule, err := h.service.GetSchedule(ctx, model.ScheduleID(scheduleID), userID)
	if err != nil {
		if err == service.ErrScheduleNotFound {
			h.respondError(w, http.StatusNotFound, "schedule not found")
			return
		}
		if err == service.ErrUnauthorized {
			h.respondError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to get schedule", "error", err, "schedule_id", scheduleID)
		h.respondError(w, http.StatusInternalServerError, "failed to get schedule")
		return
	}

	// Convert to response DTO
	resp := h.scheduleToDTO(schedule)
	h.respondJSON(w, http.StatusOK, resp)
}

// UpdateSchedule updates a schedule
func (h *ScheduleHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get schedule ID from path
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Parse request
	var req dto.UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse dates
	var startDate, endDate *time.Time
	if req.StartDate != "" {
		t, err := time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid start date format")
			return
		}
		startDate = &t
	}
	if req.EndDate != "" {
		t, err := time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "invalid end date format")
			return
		}
		endDate = &t
	}

	// Update schedule
	schedule, err := h.service.UpdateSchedule(ctx, service.UpdateScheduleCommand{
		ID:             scheduleID,
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		StartDate:      startDate,
		EndDate:        endDate,
		InputData:      req.InputData,
	})
	if err != nil {
		if err == service.ErrScheduleNotFound {
			h.respondError(w, http.StatusNotFound, "schedule not found")
			return
		}
		if err == service.ErrUnauthorized {
			h.respondError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to update schedule", "error", err, "schedule_id", scheduleID)
		h.respondError(w, http.StatusInternalServerError, "failed to update schedule")
		return
	}

	// Convert to response DTO
	resp := h.scheduleToDTO(schedule)
	h.respondJSON(w, http.StatusOK, resp)
}

// DeleteSchedule deletes a schedule
func (h *ScheduleHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get schedule ID from path
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Delete schedule
	err := h.service.DeleteSchedule(ctx, model.ScheduleID(scheduleID), userID)
	if err != nil {
		if err == service.ErrScheduleNotFound {
			h.respondError(w, http.StatusNotFound, "schedule not found")
			return
		}
		if err == service.ErrUnauthorized {
			h.respondError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to delete schedule", "error", err, "schedule_id", scheduleID)
		h.respondError(w, http.StatusInternalServerError, "failed to delete schedule")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "schedule deleted successfully"})
}

// PauseSchedule pauses a schedule
func (h *ScheduleHandler) PauseSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get schedule ID from path
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Pause schedule
	err := h.service.PauseSchedule(ctx, model.ScheduleID(scheduleID), userID)
	if err != nil {
		if err == service.ErrScheduleNotFound {
			h.respondError(w, http.StatusNotFound, "schedule not found")
			return
		}
		if err == service.ErrUnauthorized {
			h.respondError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to pause schedule", "error", err, "schedule_id", scheduleID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "schedule paused"})
}

// ResumeSchedule resumes a paused schedule
func (h *ScheduleHandler) ResumeSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get schedule ID from path
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Resume schedule
	err := h.service.ResumeSchedule(ctx, model.ScheduleID(scheduleID), userID)
	if err != nil {
		if err == service.ErrScheduleNotFound {
			h.respondError(w, http.StatusNotFound, "schedule not found")
			return
		}
		if err == service.ErrUnauthorized {
			h.respondError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to resume schedule", "error", err, "schedule_id", scheduleID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "schedule resumed"})
}

// ExecuteSchedule manually executes a schedule
func (h *ScheduleHandler) ExecuteSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get schedule ID from path
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Execute schedule
	err := h.service.ExecuteSchedule(ctx, model.ScheduleID(scheduleID), userID)
	if err != nil {
		if err == service.ErrScheduleNotFound {
			h.respondError(w, http.StatusNotFound, "schedule not found")
			return
		}
		if err == service.ErrUnauthorized {
			h.respondError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("Failed to execute schedule", "error", err, "schedule_id", scheduleID)
		h.respondError(w, http.StatusInternalServerError, "failed to execute schedule")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "schedule execution triggered"})
}

// ListSchedules lists schedules
func (h *ScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))

	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// List schedules
	schedules, total, err := h.service.ListSchedules(ctx, service.ListSchedulesQuery{
		UserID:         userID,
		OrganizationID: query.Get("organizationId"),
		WorkflowID:     query.Get("workflowId"),
		Status:         query.Get("status"),
		Offset:         offset,
		Limit:          limit,
	})
	if err != nil {
		h.logger.Error("Failed to list schedules", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to list schedules")
		return
	}

	// Convert to response DTOs
	items := make([]dto.ScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		items[i] = h.scheduleToDTO(schedule)
	}

	resp := dto.ListSchedulesResponse{
		Items: items,
		Total: total,
		Pagination: dto.Pagination{
			Offset: offset,
			Limit:  limit,
			Total:  total,
		},
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// ListWorkflowSchedules lists schedules for a workflow
func (h *ScheduleHandler) ListWorkflowSchedules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["workflowId"]

	// List schedules for workflow
	schedules, total, err := h.service.ListSchedules(ctx, service.ListSchedulesQuery{
		UserID:     userID,
		WorkflowID: workflowID,
		Offset:     0,
		Limit:      100,
	})
	if err != nil {
		h.logger.Error("Failed to list workflow schedules", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusInternalServerError, "failed to list schedules")
		return
	}

	// Convert to response DTOs
	items := make([]dto.ScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		items[i] = h.scheduleToDTO(schedule)
	}

	resp := dto.ListSchedulesResponse{
		Items: items,
		Total: total,
		Pagination: dto.Pagination{
			Offset: 0,
			Limit:  100,
			Total:  total,
		},
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// ValidateCron validates a cron expression
func (h *ScheduleHandler) ValidateCron(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req dto.ValidateCronRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate cron expression
	_, err := model.NewSchedule(
		"test",
		"test",
		"test",
		req.CronExpression,
		req.Timezone,
	)

	if err != nil {
		h.respondJSON(w, http.StatusOK, dto.ValidateCronResponse{
			Valid:   false,
			Message: err.Error(),
		})
		return
	}

	h.respondJSON(w, http.StatusOK, dto.ValidateCronResponse{
		Valid:   true,
		Message: "Valid cron expression",
	})
}

// Helper methods

func (h *ScheduleHandler) scheduleToDTO(schedule *model.Schedule) dto.ScheduleResponse {
	resp := dto.ScheduleResponse{
		ID:             schedule.ID().String(),
		UserID:         schedule.UserID(),
		OrganizationID: schedule.OrganizationID(),
		WorkflowID:     schedule.WorkflowID(),
		Name:           schedule.Name(),
		Description:    schedule.Description(),
		CronExpression: schedule.CronExpression(),
		Timezone:       schedule.Timezone(),
		Status:         string(schedule.Status()),
		LastRunAt:      schedule.LastRunAt(),
		NextRunAt:      schedule.NextRunAt(),
		RunCount:       schedule.RunCount(),
		SuccessCount:   schedule.SuccessCount(),
		FailureCount:   schedule.FailureCount(),
		LastError:      schedule.LastError(),
		InputData:      schedule.InputData(),
		CreatedAt:      schedule.CreatedAt(),
		UpdatedAt:      schedule.UpdatedAt(),
	}

	// Format dates
	if schedule.StartDate() != nil {
		formatted := schedule.StartDate().Format(time.RFC3339)
		resp.StartDate = &formatted
	}
	if schedule.EndDate() != nil {
		formatted := schedule.EndDate().Format(time.RFC3339)
		resp.EndDate = &formatted
	}

	return resp
}

func (h *ScheduleHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *ScheduleHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

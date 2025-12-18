package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/adapters/http/dto"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

// WorkflowHandler handles HTTP requests for workflows
type WorkflowHandler struct {
	service *service.WorkflowService
	logger  logger.Logger
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(service *service.WorkflowService, logger logger.Logger) *WorkflowHandler {
	return &WorkflowHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers workflow routes
func (h *WorkflowHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/workflows", h.CreateWorkflow).Methods("POST")
	router.HandleFunc("/workflows", h.ListWorkflows).Methods("GET")
	router.HandleFunc("/workflows/{id}", h.GetWorkflow).Methods("GET")
	router.HandleFunc("/workflows/{id}", h.UpdateWorkflow).Methods("PUT")
	router.HandleFunc("/workflows/{id}", h.DeleteWorkflow).Methods("DELETE")
	router.HandleFunc("/workflows/{id}/activate", h.ActivateWorkflow).Methods("POST")
	router.HandleFunc("/workflows/{id}/deactivate", h.DeactivateWorkflow).Methods("POST")
	router.HandleFunc("/workflows/{id}/duplicate", h.DuplicateWorkflow).Methods("POST")
}

// CreateWorkflow creates a new workflow
func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req dto.CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get user ID from context (would come from auth middleware)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "test-user-123" // Default for testing
	}

	// Create workflow
	workflow, err := h.service.CreateWorkflow(ctx, service.CreateWorkflowCommand{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Nodes:       h.convertNodeDTOs(req.Nodes),
		Connections: h.convertConnectionDTOs(req.Connections),
	})
	if err != nil {
		h.logger.Error("Failed to create workflow", "error", err, "user_id", userID)
		h.respondError(w, http.StatusInternalServerError, "Failed to create workflow")
		return
	}

	// Convert to response DTO
	resp := h.workflowToDTO(workflow)
	h.respondJSON(w, http.StatusCreated, resp)
}

// GetWorkflow gets a workflow by ID
func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["id"]
	
	// Get workflow
	workflow, err := h.service.GetWorkflow(ctx, model.WorkflowID(workflowID))
	if err != nil {
		if err == service.ErrWorkflowNotFound {
			h.respondError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to get workflow", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusInternalServerError, "Failed to get workflow")
		return
	}

	// Convert to response DTO
	resp := h.workflowToDTO(workflow)
	h.respondJSON(w, http.StatusOK, resp)
}

// ListWorkflows lists workflows for a user
func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	// Get user ID from context
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "test-user-123"
	}

	// List workflows
	workflows, total, err := h.service.ListWorkflows(ctx, service.ListWorkflowsQuery{
		UserID: userID,
		Offset: offset,
		Limit:  limit,
		Status: query.Get("status"),
	})
	if err != nil {
		h.logger.Error("Failed to list workflows", "error", err, "user_id", userID)
		h.respondError(w, http.StatusInternalServerError, "Failed to list workflows")
		return
	}

	// Convert to response DTOs
	items := make([]dto.WorkflowResponse, len(workflows))
	for i, wf := range workflows {
		items[i] = h.workflowToDTO(wf)
	}

	resp := dto.ListWorkflowsResponse{
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

// UpdateWorkflow updates a workflow
func (h *WorkflowHandler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["id"]
	
	// Parse request
	var req dto.UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update workflow
	workflow, err := h.service.UpdateWorkflow(ctx, service.UpdateWorkflowCommand{
		WorkflowID:  model.WorkflowID(workflowID),
		Name:        req.Name,
		Description: req.Description,
		Nodes:       h.convertNodeDTOs(req.Nodes),
		Connections: h.convertConnectionDTOs(req.Connections),
	})
	if err != nil {
		if err == service.ErrWorkflowNotFound {
			h.respondError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to update workflow", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusInternalServerError, "Failed to update workflow")
		return
	}

	// Convert to response DTO
	resp := h.workflowToDTO(workflow)
	h.respondJSON(w, http.StatusOK, resp)
}

// DeleteWorkflow deletes a workflow
func (h *WorkflowHandler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["id"]
	
	// Delete workflow
	err := h.service.DeleteWorkflow(ctx, model.WorkflowID(workflowID))
	if err != nil {
		if err == service.ErrWorkflowNotFound {
			h.respondError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to delete workflow", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ActivateWorkflow activates a workflow
func (h *WorkflowHandler) ActivateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["id"]
	
	// Activate workflow
	workflow, err := h.service.ActivateWorkflow(ctx, model.WorkflowID(workflowID))
	if err != nil {
		if err == service.ErrWorkflowNotFound {
			h.respondError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to activate workflow", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to response DTO
	resp := h.workflowToDTO(workflow)
	h.respondJSON(w, http.StatusOK, resp)
}

// DeactivateWorkflow deactivates a workflow
func (h *WorkflowHandler) DeactivateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["id"]
	
	// Deactivate workflow
	workflow, err := h.service.DeactivateWorkflow(ctx, model.WorkflowID(workflowID))
	if err != nil {
		if err == service.ErrWorkflowNotFound {
			h.respondError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to deactivate workflow", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to response DTO
	resp := h.workflowToDTO(workflow)
	h.respondJSON(w, http.StatusOK, resp)
}

// DuplicateWorkflow duplicates a workflow
func (h *WorkflowHandler) DuplicateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get workflow ID from path
	vars := mux.Vars(r)
	workflowID := vars["id"]
	
	// Parse request
	var req dto.DuplicateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Duplicate workflow
	workflow, err := h.service.DuplicateWorkflow(ctx, model.WorkflowID(workflowID), req.Name)
	if err != nil {
		if err == service.ErrWorkflowNotFound {
			h.respondError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to duplicate workflow", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusInternalServerError, "Failed to duplicate workflow")
		return
	}

	// Convert to response DTO
	resp := h.workflowToDTO(workflow)
	h.respondJSON(w, http.StatusCreated, resp)
}

// Helper methods

func (h *WorkflowHandler) workflowToDTO(workflow *model.Workflow) dto.WorkflowResponse {
	return dto.WorkflowResponse{
		ID:          workflow.ID().String(),
		Name:        workflow.Name(),
		Description: workflow.Description(),
		Status:      string(workflow.Status()),
		Nodes:       h.convertNodesToDTO(workflow.Nodes()),
		Connections: h.convertConnectionsToDTO(workflow.Connections()),
		Settings:    h.convertSettingsToDTO(workflow.Settings()),
		CreatedAt:   workflow.CreatedAt(),
		UpdatedAt:   workflow.UpdatedAt(),
	}
}

func (h *WorkflowHandler) convertNodeDTOs(nodes []dto.NodeDTO) []model.Node {
	result := make([]model.Node, len(nodes))
	for i, n := range nodes {
		result[i] = model.Node{
			ID:          n.ID,
			Type:        model.NodeType(n.Type),
			Name:        n.Name,
			Description: n.Description,
			Config:      n.Config,
			Position: model.Position{
				X: n.Position.X,
				Y: n.Position.Y,
			},
		}
	}
	return result
}

func (h *WorkflowHandler) convertConnectionDTOs(connections []dto.ConnectionDTO) []model.Connection {
	result := make([]model.Connection, len(connections))
	for i, c := range connections {
		result[i] = model.Connection{
			ID:           c.ID,
			SourceNodeID: c.SourceNodeID,
			TargetNodeID: c.TargetNodeID,
			SourcePort:   c.SourcePort,
			TargetPort:   c.TargetPort,
		}
	}
	return result
}

func (h *WorkflowHandler) convertNodesToDTO(nodes []model.Node) []dto.NodeDTO {
	result := make([]dto.NodeDTO, len(nodes))
	for i, n := range nodes {
		result[i] = dto.NodeDTO{
			ID:          n.ID,
			Type:        string(n.Type),
			Name:        n.Name,
			Description: n.Description,
			Config:      n.Config,
			Position: dto.PositionDTO{
				X: n.Position.X,
				Y: n.Position.Y,
			},
		}
	}
	return result
}

func (h *WorkflowHandler) convertConnectionsToDTO(connections []model.Connection) []dto.ConnectionDTO {
	result := make([]dto.ConnectionDTO, len(connections))
	for i, c := range connections {
		result[i] = dto.ConnectionDTO{
			ID:           c.ID,
			SourceNodeID: c.SourceNodeID,
			TargetNodeID: c.TargetNodeID,
			SourcePort:   c.SourcePort,
			TargetPort:   c.TargetPort,
		}
	}
	return result
}

func (h *WorkflowHandler) convertSettingsToDTO(settings model.Settings) dto.SettingsDTO {
	return dto.SettingsDTO{
		MaxExecutionTime: settings.MaxExecutionTime,
		RetryPolicy: dto.RetryPolicyDTO{
			MaxAttempts: settings.RetryPolicy.MaxAttempts,
			BackoffType: settings.RetryPolicy.BackoffType,
			Delay:       settings.RetryPolicy.Delay,
		},
		ErrorHandling: string(settings.ErrorHandling),
		Metadata:      settings.Metadata,
	}
}

func (h *WorkflowHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *WorkflowHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

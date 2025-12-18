package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/execution/adapters/http/dto"
	"github.com/linkflow-ai/linkflow-ai/internal/execution/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/execution/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/middleware"
)

// ExecutionHandler handles HTTP requests for executions
type ExecutionHandler struct {
	service *service.ExecutionService
	logger  logger.Logger
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(service *service.ExecutionService, logger logger.Logger) *ExecutionHandler {
	return &ExecutionHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers execution routes
func (h *ExecutionHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/executions", h.StartExecution).Methods("POST")
	router.HandleFunc("/executions", h.ListExecutions).Methods("GET")
	router.HandleFunc("/executions/{id}", h.GetExecution).Methods("GET")
	router.HandleFunc("/executions/{id}/cancel", h.CancelExecution).Methods("POST")
	router.HandleFunc("/executions/{id}/pause", h.PauseExecution).Methods("POST")
	router.HandleFunc("/executions/{id}/resume", h.ResumeExecution).Methods("POST")
	router.HandleFunc("/workflows/{workflowId}/execute", h.ExecuteWorkflow).Methods("POST")
	router.HandleFunc("/executions/{id}/logs", h.GetExecutionLogs).Methods("GET")
}

// StartExecution starts a new execution
func (h *ExecutionHandler) StartExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user ID from context
	userID, ok := middleware.ExtractUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse request
	var req dto.StartExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Start execution
	execution, err := h.service.StartExecution(ctx, service.StartExecutionCommand{
		WorkflowID:  req.WorkflowID,
		UserID:      userID,
		TriggerType: req.TriggerType,
		InputData:   req.InputData,
	})
	if err != nil {
		h.logger.Error("Failed to start execution", "error", err, "workflow_id", req.WorkflowID)
		h.respondError(w, http.StatusInternalServerError, "failed to start execution")
		return
	}

	// Convert to response DTO
	resp := h.executionToDTO(execution)
	h.respondJSON(w, http.StatusCreated, resp)
}

// ExecuteWorkflow executes a workflow directly
func (h *ExecutionHandler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
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

	// Parse request
	var req dto.ExecuteWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body, use empty input
		req.InputData = make(map[string]interface{})
	}

	// Execute workflow
	execution, err := h.service.ExecuteWorkflow(ctx, workflowID, userID, req.InputData)
	if err != nil {
		h.logger.Error("Failed to execute workflow", "error", err, "workflow_id", workflowID)
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to response DTO
	resp := h.executionToDTO(execution)
	h.respondJSON(w, http.StatusOK, resp)
}

// GetExecution gets an execution by ID
func (h *ExecutionHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get execution ID from path
	vars := mux.Vars(r)
	executionID := vars["id"]

	// Get execution
	execution, err := h.service.GetExecution(ctx, model.ExecutionID(executionID))
	if err != nil {
		if err == service.ErrExecutionNotFound {
			h.respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		h.logger.Error("Failed to get execution", "error", err, "execution_id", executionID)
		h.respondError(w, http.StatusInternalServerError, "failed to get execution")
		return
	}

	// Convert to response DTO
	resp := h.executionToDTO(execution)
	h.respondJSON(w, http.StatusOK, resp)
}

// ListExecutions lists executions
func (h *ExecutionHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
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

	// List executions
	executions, total, err := h.service.ListExecutions(ctx, service.ListExecutionsQuery{
		UserID:     userID,
		WorkflowID: query.Get("workflowId"),
		Status:     query.Get("status"),
		Offset:     offset,
		Limit:      limit,
	})
	if err != nil {
		h.logger.Error("Failed to list executions", "error", err, "user_id", userID)
		h.respondError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}

	// Convert to response DTOs
	items := make([]dto.ExecutionResponse, len(executions))
	for i, exec := range executions {
		items[i] = h.executionToDTO(exec)
	}

	resp := dto.ListExecutionsResponse{
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

// CancelExecution cancels a running execution
func (h *ExecutionHandler) CancelExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get execution ID from path
	vars := mux.Vars(r)
	executionID := vars["id"]

	// Cancel execution
	err := h.service.CancelExecution(ctx, model.ExecutionID(executionID))
	if err != nil {
		if err == service.ErrExecutionNotFound {
			h.respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		h.logger.Error("Failed to cancel execution", "error", err, "execution_id", executionID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "execution cancelled"})
}

// PauseExecution pauses a running execution
func (h *ExecutionHandler) PauseExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get execution ID from path
	vars := mux.Vars(r)
	executionID := vars["id"]

	// Pause execution
	err := h.service.PauseExecution(ctx, model.ExecutionID(executionID))
	if err != nil {
		if err == service.ErrExecutionNotFound {
			h.respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		h.logger.Error("Failed to pause execution", "error", err, "execution_id", executionID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "execution paused"})
}

// ResumeExecution resumes a paused execution
func (h *ExecutionHandler) ResumeExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get execution ID from path
	vars := mux.Vars(r)
	executionID := vars["id"]

	// Resume execution
	err := h.service.ResumeExecution(ctx, model.ExecutionID(executionID))
	if err != nil {
		if err == service.ErrExecutionNotFound {
			h.respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		h.logger.Error("Failed to resume execution", "error", err, "execution_id", executionID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "execution resumed"})
}

// GetExecutionLogs gets execution logs
func (h *ExecutionHandler) GetExecutionLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get execution ID from path
	vars := mux.Vars(r)
	executionID := vars["id"]

	// Get execution for logs
	execution, err := h.service.GetExecution(ctx, model.ExecutionID(executionID))
	if err != nil {
		if err == service.ErrExecutionNotFound {
			h.respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		h.logger.Error("Failed to get execution logs", "error", err, "execution_id", executionID)
		h.respondError(w, http.StatusInternalServerError, "failed to get execution logs")
		return
	}

	// Build logs from node executions
	logs := []dto.ExecutionLog{}
	for nodeID, nodeExec := range execution.NodeExecutions() {
		log := dto.ExecutionLog{
			NodeID:    nodeID,
			NodeType:  nodeExec.NodeType,
			Status:    string(nodeExec.Status),
			StartedAt: nodeExec.StartedAt,
			CompletedAt: nodeExec.CompletedAt,
			DurationMs: nodeExec.DurationMs,
			InputData: nodeExec.InputData,
			OutputData: nodeExec.OutputData,
		}
		
		if nodeExec.Error != nil {
			log.Error = &dto.ExecutionError{
				Code:    nodeExec.Error.Code,
				Message: nodeExec.Error.Message,
				Details: nodeExec.Error.Details,
			}
		}
		
		logs = append(logs, log)
	}

	resp := dto.ExecutionLogsResponse{
		ExecutionID: executionID,
		Logs:        logs,
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// Helper methods

func (h *ExecutionHandler) executionToDTO(execution *model.Execution) dto.ExecutionResponse {
	resp := dto.ExecutionResponse{
		ID:              execution.ID().String(),
		WorkflowID:      execution.WorkflowID(),
		WorkflowVersion: execution.WorkflowVersion(),
		UserID:          execution.UserID(),
		TriggerType:     string(execution.TriggerType()),
		Status:          string(execution.Status()),
		InputData:       execution.InputData(),
		OutputData:      execution.OutputData(),
		StartedAt:       execution.StartedAt(),
		CompletedAt:     execution.CompletedAt(),
		DurationMs:      execution.DurationMs(),
		CreatedAt:       execution.CreatedAt(),
	}

	// Add error if present
	if execution.Error() != nil {
		resp.Error = &dto.ExecutionError{
			Code:    execution.Error().Code,
			Message: execution.Error().Message,
			Details: execution.Error().Details,
		}
	}

	// Add node executions
	resp.NodeExecutions = make(map[string]dto.NodeExecutionResponse)
	for nodeID, nodeExec := range execution.NodeExecutions() {
		nodeResp := dto.NodeExecutionResponse{
			NodeID:      nodeExec.NodeID,
			NodeType:    nodeExec.NodeType,
			Status:      string(nodeExec.Status),
			StartedAt:   nodeExec.StartedAt,
			CompletedAt: nodeExec.CompletedAt,
			DurationMs:  nodeExec.DurationMs,
			RetryCount:  nodeExec.RetryCount,
		}
		
		if nodeExec.Error != nil {
			nodeResp.Error = &dto.ExecutionError{
				Code:    nodeExec.Error.Code,
				Message: nodeExec.Error.Message,
			}
		}
		
		resp.NodeExecutions[nodeID] = nodeResp
	}

	return resp
}

func (h *ExecutionHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *ExecutionHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

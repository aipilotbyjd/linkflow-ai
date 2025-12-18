// Package handlers provides HTTP handlers for the executor service
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/linkflow-ai/linkflow-ai/internal/executor/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/executor/domain/model"
)

// ExecutorHandler handles executor HTTP requests
type ExecutorHandler struct {
	service *service.ExecutorService
}

// NewExecutorHandler creates a new executor handler
func NewExecutorHandler(svc *service.ExecutorService) *ExecutorHandler {
	return &ExecutorHandler{service: svc}
}

// RegisterRoutes registers executor routes
func (h *ExecutorHandler) RegisterRoutes(mux *http.ServeMux) {
	// Worker endpoints
	mux.HandleFunc("/api/v1/workers", h.handleWorkers)
	mux.HandleFunc("/api/v1/workers/", h.handleWorker)
	mux.HandleFunc("/api/v1/workers/register", h.registerWorker)
	mux.HandleFunc("/api/v1/workers/heartbeat", h.heartbeat)

	// Task endpoints
	mux.HandleFunc("/api/v1/tasks", h.handleTasks)
	mux.HandleFunc("/api/v1/tasks/", h.handleTask)
	mux.HandleFunc("/api/v1/tasks/submit", h.submitTask)
}

func (h *ExecutorHandler) handleWorkers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listWorkers(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ExecutorHandler) handleWorker(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/workers/"):]
	if id == "" {
		http.Error(w, "Worker ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getWorker(w, r, id)
	case http.MethodDelete:
		h.unregisterWorker(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ExecutorHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.submitTask(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ExecutorHandler) handleTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/tasks/"):]
	
	// Handle sub-paths
	if len(id) > 0 && id[len(id)-1] == '/' {
		id = id[:len(id)-1]
	}

	// Check for action paths
	if idx := findIndex(id, "/"); idx != -1 {
		taskID := id[:idx]
		action := id[idx+1:]

		switch action {
		case "complete":
			h.completeTask(w, r, taskID)
		case "fail":
			h.failTask(w, r, taskID)
		default:
			http.Error(w, "Unknown action", http.StatusBadRequest)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getTask(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// RegisterWorkerRequest represents worker registration request
type RegisterWorkerRequest struct {
	Name     string   `json:"name"`
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Capacity int      `json:"capacity"`
	Tags     []string `json:"tags"`
}

// WorkerResponse represents worker response
type WorkerResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Status        string   `json:"status"`
	Capacity      int      `json:"capacity"`
	CurrentLoad   int      `json:"currentLoad"`
	Tags          []string `json:"tags"`
	LastHeartbeat string   `json:"lastHeartbeat"`
	RegisteredAt  string   `json:"registeredAt"`
}

func (h *ExecutorHandler) registerWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Capacity == 0 {
		req.Capacity = 10
	}

	worker, err := h.service.RegisterWorker(r.Context(), service.RegisterWorkerInput{
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		Capacity: req.Capacity,
		Tags:     req.Tags,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toWorkerResponse(worker))
}

func (h *ExecutorHandler) unregisterWorker(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.UnregisterWorker(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HeartbeatRequest represents heartbeat request
type HeartbeatRequest struct {
	WorkerID    string `json:"workerId"`
	Status      string `json:"status"`
	CurrentLoad int    `json:"currentLoad"`
}

func (h *ExecutorHandler) heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	status := model.WorkerStatus(req.Status)
	if status == "" {
		status = model.WorkerStatusIdle
	}

	if err := h.service.Heartbeat(r.Context(), req.WorkerID, status, req.CurrentLoad); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *ExecutorHandler) listWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := h.service.GetWorkers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]WorkerResponse, len(workers))
	for i, worker := range workers {
		response[i] = toWorkerResponse(worker)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": response,
		"total": len(response),
	})
}

func (h *ExecutorHandler) getWorker(w http.ResponseWriter, r *http.Request, id string) {
	worker, err := h.service.GetWorker(r.Context(), id)
	if err != nil {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toWorkerResponse(worker))
}

// SubmitTaskRequest represents task submission request
type SubmitTaskRequest struct {
	ExecutionID string                 `json:"executionId"`
	NodeID      string                 `json:"nodeId"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Input       map[string]interface{} `json:"input"`
	MaxRetries  int                    `json:"maxRetries"`
}

// TaskResponse represents task response
type TaskResponse struct {
	ID          string                 `json:"id"`
	ExecutionID string                 `json:"executionId"`
	NodeID      string                 `json:"nodeId"`
	WorkerID    string                 `json:"workerId,omitempty"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Priority    int                    `json:"priority"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"maxRetries"`
	CreatedAt   string                 `json:"createdAt"`
	StartedAt   string                 `json:"startedAt,omitempty"`
	CompletedAt string                 `json:"completedAt,omitempty"`
}

func (h *ExecutorHandler) submitTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}

	task, err := h.service.SubmitTask(r.Context(), service.SubmitTaskInput{
		ExecutionID: req.ExecutionID,
		NodeID:      req.NodeID,
		Type:        req.Type,
		Priority:    req.Priority,
		Input:       req.Input,
		MaxRetries:  req.MaxRetries,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

func (h *ExecutorHandler) getTask(w http.ResponseWriter, r *http.Request, id string) {
	task, err := h.service.GetTask(r.Context(), id)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

// CompleteTaskRequest represents task completion request
type CompleteTaskRequest struct {
	Output map[string]interface{} `json:"output"`
}

func (h *ExecutorHandler) completeTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CompleteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.CompleteTask(r.Context(), taskID, req.Output); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}

// FailTaskRequest represents task failure request
type FailTaskRequest struct {
	Error string `json:"error"`
}

func (h *ExecutorHandler) failTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FailTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.FailTask(r.Context(), taskID, req.Error); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "failed"})
}

func toWorkerResponse(w *model.Worker) WorkerResponse {
	return WorkerResponse{
		ID:            w.ID,
		Name:          w.Name,
		Host:          w.Host,
		Port:          w.Port,
		Status:        string(w.Status),
		Capacity:      w.Capacity,
		CurrentLoad:   w.CurrentLoad,
		Tags:          w.Tags,
		LastHeartbeat: w.LastHeartbeat.Format("2006-01-02T15:04:05Z"),
		RegisteredAt:  w.RegisteredAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toTaskResponse(t *model.Task) TaskResponse {
	resp := TaskResponse{
		ID:          t.ID,
		ExecutionID: t.ExecutionID,
		NodeID:      t.NodeID,
		WorkerID:    t.WorkerID,
		Type:        t.Type,
		Status:      string(t.Status),
		Priority:    t.Priority,
		Input:       t.Input,
		Output:      t.Output,
		Error:       t.Error,
		Retries:     t.Retries,
		MaxRetries:  t.MaxRetries,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if t.StartedAt != nil {
		resp.StartedAt = t.StartedAt.Format("2006-01-02T15:04:05Z")
	}
	if t.CompletedAt != nil {
		resp.CompletedAt = t.CompletedAt.Format("2006-01-02T15:04:05Z")
	}

	return resp
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// parseInt helper
func parseInt(s string, defaultVal int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}

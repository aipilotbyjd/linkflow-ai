// Package handlers provides HTTP handlers for migration operations
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/linkflow-ai/linkflow-ai/internal/migration/app/service"
)

// MigrationHandler handles migration HTTP requests
type MigrationHandler struct {
	svc *service.MigrationService
}

// NewMigrationHandler creates a new migration handler
func NewMigrationHandler(svc *service.MigrationService) *MigrationHandler {
	return &MigrationHandler{svc: svc}
}

// StatusResponse represents migration status response
type StatusResponse struct {
	CurrentVersion int64               `json:"current_version"`
	Pending        int                 `json:"pending"`
	Applied        int                 `json:"applied"`
	LastApplied    string              `json:"last_applied,omitempty"`
	Migrations     []MigrationResponse `json:"migrations,omitempty"`
}

// MigrationResponse represents a single migration in response
type MigrationResponse struct {
	Version    int64  `json:"version"`
	Name       string `json:"name"`
	Direction  string `json:"direction"`
	Status     string `json:"status"`
	ExecutedAt string `json:"executed_at,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// MigrateRequest represents migration request body
type MigrateRequest struct {
	Steps  int  `json:"steps"`
	DryRun bool `json:"dry_run"`
}

// MigrateResponse represents migration result response
type MigrateResponse struct {
	Applied []MigrationResultResponse `json:"applied"`
	Errors  []string                  `json:"errors,omitempty"`
}

// MigrationResultResponse represents a migration execution result
type MigrationResultResponse struct {
	Version    int64  `json:"version"`
	Name       string `json:"name"`
	Direction  string `json:"direction"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// HandleStatus returns migration status
func (h *MigrationHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := StatusResponse{
		CurrentVersion: status.CurrentVersion,
		Pending:        status.Pending,
		Applied:        status.Applied,
		LastApplied:    status.LastApplied,
		Migrations:     make([]MigrationResponse, 0, len(status.Migrations)),
	}

	for _, m := range status.Migrations {
		mr := MigrationResponse{
			Version:    m.Version,
			Name:       m.Name,
			Direction:  string(m.Direction),
			Status:     string(m.Status),
			DurationMs: m.DurationMs,
		}
		if m.ExecutedAt != nil {
			mr.ExecutedAt = m.ExecutedAt.Format("2006-01-02T15:04:05Z")
		}
		resp.Migrations = append(resp.Migrations, mr)
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleList lists migrations with optional filter
func (h *MigrationHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	migrations, err := h.svc.ListMigrations(r.Context(), statusFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]MigrationResponse, 0, len(migrations))
	for _, m := range migrations {
		mr := MigrationResponse{
			Version:    m.Version,
			Name:       m.Name,
			Direction:  string(m.Direction),
			Status:     string(m.Status),
			DurationMs: m.DurationMs,
		}
		if m.ExecutedAt != nil {
			mr.ExecutedAt = m.ExecutedAt.Format("2006-01-02T15:04:05Z")
		}
		resp = append(resp, mr)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"migrations": resp})
}

// HandleGetMigration returns a specific migration
func (h *MigrationHandler) HandleGetMigration(w http.ResponseWriter, r *http.Request) {
	versionStr := r.URL.Query().Get("version")
	if versionStr == "" {
		writeError(w, http.StatusBadRequest, "version is required")
		return
	}

	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid version")
		return
	}

	migration, err := h.svc.GetMigration(r.Context(), version)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	resp := MigrationResponse{
		Version:   migration.Version,
		Name:      migration.Name,
		Direction: string(migration.Direction),
		Status:    string(migration.Status),
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleMigrateUp runs pending migrations
func (h *MigrationHandler) HandleMigrateUp(w http.ResponseWriter, r *http.Request) {
	var req MigrateRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	result, err := h.svc.MigrateUp(r.Context(), service.MigrateInput{
		Steps:  req.Steps,
		DryRun: req.DryRun,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := MigrateResponse{
		Applied: make([]MigrationResultResponse, 0, len(result.Applied)),
		Errors:  result.Errors,
	}

	for _, m := range result.Applied {
		resp.Applied = append(resp.Applied, MigrationResultResponse{
			Version:    m.Version,
			Name:       m.Name,
			Direction:  string(m.Direction),
			Status:     string(m.Status),
			DurationMs: m.DurationMs,
			Error:      m.Error,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleMigrateDown rolls back migrations
func (h *MigrationHandler) HandleMigrateDown(w http.ResponseWriter, r *http.Request) {
	var req MigrateRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	// Default to 1 step for safety
	if req.Steps == 0 {
		req.Steps = 1
	}

	result, err := h.svc.MigrateDown(r.Context(), service.MigrateInput{
		Steps:  req.Steps,
		DryRun: req.DryRun,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := MigrateResponse{
		Applied: make([]MigrationResultResponse, 0, len(result.Applied)),
		Errors:  result.Errors,
	}

	for _, m := range result.Applied {
		resp.Applied = append(resp.Applied, MigrationResultResponse{
			Version:    m.Version,
			Name:       m.Name,
			Direction:  string(m.Direction),
			Status:     string(m.Status),
			DurationMs: m.DurationMs,
			Error:      m.Error,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleReset resets the database
func (h *MigrationHandler) HandleReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DryRun bool `json:"dry_run"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	result, err := h.svc.Reset(r.Context(), req.DryRun)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := MigrateResponse{
		Applied: make([]MigrationResultResponse, 0, len(result.Applied)),
		Errors:  result.Errors,
	}

	for _, m := range result.Applied {
		resp.Applied = append(resp.Applied, MigrationResultResponse{
			Version:    m.Version,
			Name:       m.Name,
			Direction:  string(m.Direction),
			Status:     string(m.Status),
			DurationMs: m.DurationMs,
			Error:      m.Error,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleSeed seeds the database
func (h *MigrationHandler) HandleSeed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tables []string `json:"tables"`
		Force  bool     `json:"force"`
	}
	if r.Body != nil && r.ContentLength > 0 {
		json.NewDecoder(r.Body).Decode(&req)
	}

	err := h.svc.Seed(r.Context(), service.SeedInput{
		Tables: req.Tables,
		Force:  req.Force,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "seed completed"})
}

// HandleVersion returns the current version
func (h *MigrationHandler) HandleVersion(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"version": status.CurrentVersion})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

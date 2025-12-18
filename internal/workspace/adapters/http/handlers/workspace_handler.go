// Package handlers provides HTTP handlers for workspace service
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/linkflow-ai/linkflow-ai/internal/workspace/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/workspace/domain/model"
)

// WorkspaceHandler handles workspace HTTP requests
type WorkspaceHandler struct {
	workspaceService *service.WorkspaceService
}

// NewWorkspaceHandler creates a new workspace handler
func NewWorkspaceHandler(workspaceService *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceService: workspaceService}
}

// RegisterRoutes registers workspace routes
func (h *WorkspaceHandler) RegisterRoutes(mux *http.ServeMux) {
	// Workspace CRUD
	mux.HandleFunc("/api/v1/workspaces", h.handleWorkspaces)
	mux.HandleFunc("/api/v1/workspaces/", h.handleWorkspace)
	
	// Members
	mux.HandleFunc("/api/v1/workspaces/members", h.handleMembers)
	
	// Invitations
	mux.HandleFunc("/api/v1/workspaces/invitations", h.handleInvitations)
	mux.HandleFunc("/api/v1/invitations/accept", h.acceptInvitation)
	mux.HandleFunc("/api/v1/invitations/decline", h.declineInvitation)
	mux.HandleFunc("/api/v1/invitations/pending", h.pendingInvitations)
	
	// Audit logs
	mux.HandleFunc("/api/v1/workspaces/audit-logs", h.getAuditLogs)
}

func (h *WorkspaceHandler) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listWorkspaces(w, r)
	case http.MethodPost:
		h.createWorkspace(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *WorkspaceHandler) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/workspaces/")
	parts := strings.Split(path, "/")
	workspaceID := parts[0]

	if len(parts) > 1 {
		switch parts[1] {
		case "members":
			h.handleWorkspaceMembers(w, r, workspaceID, parts)
		case "invitations":
			h.handleWorkspaceInvitations(w, r, workspaceID)
		case "audit-logs":
			h.getWorkspaceAuditLogs(w, r, workspaceID)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getWorkspace(w, r, workspaceID)
	case http.MethodPut:
		h.updateWorkspace(w, r, workspaceID)
	case http.MethodDelete:
		h.deleteWorkspace(w, r, workspaceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CreateWorkspaceRequest represents workspace creation request
type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// WorkspaceResponse represents workspace response
type WorkspaceResponse struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	Description string                  `json:"description"`
	Plan        string                  `json:"plan"`
	Settings    model.WorkspaceSettings `json:"settings"`
	Limits      model.WorkspaceLimits   `json:"limits"`
	CreatedAt   string                  `json:"createdAt"`
	UpdatedAt   string                  `json:"updatedAt"`
}

func (h *WorkspaceHandler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Slug == "" {
		writeError(w, "Name and slug are required", http.StatusBadRequest)
		return
	}

	workspace, err := h.workspaceService.CreateWorkspace(r.Context(), service.CreateWorkspaceInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		OwnerID:     userID,
		Plan:        model.PlanFree,
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, toWorkspaceResponse(workspace))
}

func (h *WorkspaceHandler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	workspaces, err := h.workspaceService.ListUserWorkspaces(r.Context(), userID)
	if err != nil {
		writeError(w, "Failed to list workspaces", http.StatusInternalServerError)
		return
	}

	response := make([]WorkspaceResponse, len(workspaces))
	for i, ws := range workspaces {
		response[i] = toWorkspaceResponse(ws)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": response,
		"total": len(response),
	})
}

func (h *WorkspaceHandler) getWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	workspace, err := h.workspaceService.GetWorkspace(r.Context(), id)
	if err != nil {
		writeError(w, "Workspace not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, toWorkspaceResponse(workspace))
}

func (h *WorkspaceHandler) updateWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Settings    *model.WorkspaceSettings `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	workspace, err := h.workspaceService.UpdateWorkspace(r.Context(), service.UpdateWorkspaceInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Settings:    req.Settings,
		ActorID:     userID,
	})
	if err != nil {
		if err == model.ErrInsufficientPermission {
			writeError(w, "Insufficient permission", http.StatusForbidden)
			return
		}
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, toWorkspaceResponse(workspace))
}

func (h *WorkspaceHandler) deleteWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.workspaceService.DeleteWorkspace(r.Context(), id, userID); err != nil {
		if err == model.ErrInsufficientPermission {
			writeError(w, "Only workspace owner can delete", http.StatusForbidden)
			return
		}
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Member handlers

func (h *WorkspaceHandler) handleMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listMembers(w, r, workspaceID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *WorkspaceHandler) handleWorkspaceMembers(w http.ResponseWriter, r *http.Request, workspaceID string, parts []string) {
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			h.listMembers(w, r, workspaceID)
		case http.MethodPost:
			h.inviteMember(w, r, workspaceID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) >= 3 {
		memberID := parts[2]
		switch r.Method {
		case http.MethodPut:
			h.updateMemberRole(w, r, workspaceID, memberID)
		case http.MethodDelete:
			h.removeMember(w, r, workspaceID, memberID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// MemberResponse represents member response
type MemberResponse struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	Role        string `json:"role"`
	JoinedAt    string `json:"joinedAt"`
}

func (h *WorkspaceHandler) listMembers(w http.ResponseWriter, r *http.Request, workspaceID string) {
	members, err := h.workspaceService.ListMembers(r.Context(), workspaceID)
	if err != nil {
		writeError(w, "Failed to list members", http.StatusInternalServerError)
		return
	}

	response := make([]MemberResponse, len(members))
	for i, m := range members {
		response[i] = MemberResponse{
			ID:       m.ID,
			UserID:   m.UserID,
			Role:     string(m.Role),
			JoinedAt: m.JoinedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": response,
		"total": len(response),
	})
}

// InviteMemberRequest represents invitation request
type InviteMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *WorkspaceHandler) inviteMember(w http.ResponseWriter, r *http.Request, workspaceID string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		writeError(w, "Email is required", http.StatusBadRequest)
		return
	}

	role := model.MemberRole(req.Role)
	if role == "" {
		role = model.RoleMember
	}

	invitation, err := h.workspaceService.InviteMember(r.Context(), service.InviteMemberInput{
		WorkspaceID: workspaceID,
		Email:       req.Email,
		Role:        role,
		InviterID:   userID,
	})
	if err != nil {
		if err == model.ErrInsufficientPermission {
			writeError(w, "Insufficient permission", http.StatusForbidden)
			return
		}
		if err == model.ErrMemberLimitReached {
			writeError(w, "Member limit reached. Please upgrade your plan.", http.StatusPaymentRequired)
			return
		}
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        invitation.ID,
		"email":     invitation.Email,
		"role":      invitation.Role,
		"expiresAt": invitation.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *WorkspaceHandler) updateMemberRole(w http.ResponseWriter, r *http.Request, workspaceID, memberID string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.workspaceService.UpdateMemberRole(r.Context(), service.UpdateMemberRoleInput{
		WorkspaceID: workspaceID,
		MemberID:    memberID,
		NewRole:     model.MemberRole(req.Role),
		ActorID:     userID,
	}); err != nil {
		if err == model.ErrInsufficientPermission {
			writeError(w, "Insufficient permission", http.StatusForbidden)
			return
		}
		if err == model.ErrCannotRemoveOwner {
			writeError(w, "Cannot change owner's role", http.StatusBadRequest)
			return
		}
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Role updated"})
}

func (h *WorkspaceHandler) removeMember(w http.ResponseWriter, r *http.Request, workspaceID, memberID string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.workspaceService.RemoveMember(r.Context(), workspaceID, memberID, userID); err != nil {
		if err == model.ErrInsufficientPermission {
			writeError(w, "Insufficient permission", http.StatusForbidden)
			return
		}
		if err == model.ErrCannotRemoveOwner {
			writeError(w, "Cannot remove workspace owner", http.StatusBadRequest)
			return
		}
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Invitation handlers

func (h *WorkspaceHandler) handleInvitations(w http.ResponseWriter, r *http.Request) {
	// List invitations for current workspace
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}

	// TODO: List invitations
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
}

func (h *WorkspaceHandler) handleWorkspaceInvitations(w http.ResponseWriter, r *http.Request, workspaceID string) {
	// TODO: List workspace invitations
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
}

func (h *WorkspaceHandler) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	member, err := h.workspaceService.AcceptInvitation(r.Context(), req.Token, userID)
	if err != nil {
		if err == model.ErrInvitationExpired {
			writeError(w, "Invitation has expired", http.StatusBadRequest)
			return
		}
		if err == model.ErrAlreadyMember {
			writeError(w, "Already a member", http.StatusBadRequest)
			return
		}
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, MemberResponse{
		ID:       member.ID,
		UserID:   member.UserID,
		Role:     string(member.Role),
		JoinedAt: member.JoinedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *WorkspaceHandler) declineInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.workspaceService.DeclineInvitation(r.Context(), req.Token); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkspaceHandler) pendingInvitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		writeError(w, "Email is required", http.StatusBadRequest)
		return
	}

	invitations, err := h.workspaceService.GetPendingInvitations(r.Context(), email)
	if err != nil {
		writeError(w, "Failed to get invitations", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(invitations))
	for i, inv := range invitations {
		response[i] = map[string]interface{}{
			"id":          inv.ID,
			"workspaceId": inv.WorkspaceID,
			"role":        inv.Role,
			"expiresAt":   inv.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": response,
		"total": len(response),
	})
}

// Audit log handlers

func (h *WorkspaceHandler) getAuditLogs(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		writeError(w, "Workspace ID required", http.StatusBadRequest)
		return
	}
	h.getWorkspaceAuditLogs(w, r, workspaceID)
}

func (h *WorkspaceHandler) getWorkspaceAuditLogs(w http.ResponseWriter, r *http.Request, workspaceID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}

	logs, total, err := h.workspaceService.ListAuditLogs(r.Context(), workspaceID, offset, limit)
	if err != nil {
		writeError(w, "Failed to list audit logs", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		response[i] = map[string]interface{}{
			"id":         log.ID,
			"userId":     log.UserID,
			"action":     log.Action,
			"resource":   log.Resource,
			"resourceId": log.ResourceID,
			"details":    log.Details,
			"createdAt":  log.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": response,
		"total": total,
	})
}

// Helper functions

func toWorkspaceResponse(ws *model.Workspace) WorkspaceResponse {
	return WorkspaceResponse{
		ID:          ws.ID.String(),
		Name:        ws.Name,
		Slug:        ws.Slug,
		Description: ws.Description,
		Plan:        string(ws.Plan),
		Settings:    ws.Settings,
		Limits:      ws.Limits,
		CreatedAt:   ws.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   ws.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

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

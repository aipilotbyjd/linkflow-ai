package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/user/adapters/http/dto"
	"github.com/linkflow-ai/linkflow-ai/internal/user/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/user/domain/model"
)

// UserHandler handles HTTP requests for users
type UserHandler struct {
	service *service.UserService
	logger  logger.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(service *service.UserService, logger logger.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers user routes
func (h *UserHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/users/register", h.Register).Methods("POST")
	router.HandleFunc("/users/login", h.Login).Methods("POST")
	router.HandleFunc("/users/profile", h.GetProfile).Methods("GET")
	router.HandleFunc("/users/profile", h.UpdateProfile).Methods("PUT")
	router.HandleFunc("/users/{id}", h.GetUser).Methods("GET")
	router.HandleFunc("/users", h.ListUsers).Methods("GET")
	router.HandleFunc("/users/{id}/block", h.BlockUser).Methods("POST")
	router.HandleFunc("/users/{id}/unblock", h.UnblockUser).Methods("POST")
	router.HandleFunc("/users/change-password", h.ChangePassword).Methods("POST")
	router.HandleFunc("/organizations", h.CreateOrganization).Methods("POST")
	router.HandleFunc("/organizations/{id}/members", h.AddOrganizationMember).Methods("POST")
}

// Register registers a new user
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.Register(ctx, service.RegisterCommand{
		Email:     req.Email,
		Username:  req.Username,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		h.logger.Error("Failed to register user", "error", err)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := h.userToDTO(user)
	h.respondJSON(w, http.StatusCreated, resp)
}

// Login authenticates a user
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, token, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.logger.Error("Failed to login", "error", err, "email", req.Email)
		h.respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	resp := dto.LoginResponse{
		User:  h.userToDTO(user),
		Token: token,
	}
	h.respondJSON(w, http.StatusOK, resp)
}

// GetProfile gets the current user's profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get user ID from context (would come from auth middleware)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.service.GetUser(ctx, model.UserID(userID))
	if err != nil {
		h.respondError(w, http.StatusNotFound, "User not found")
		return
	}

	resp := h.userToDTO(user)
	h.respondJSON(w, http.StatusOK, resp)
}

// UpdateProfile updates the current user's profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(ctx, model.UserID(userID), service.UpdateProfileCommand{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		AvatarURL: req.AvatarURL,
	})
	if err != nil {
		h.logger.Error("Failed to update profile", "error", err, "user_id", userID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := h.userToDTO(user)
	h.respondJSON(w, http.StatusOK, resp)
}

// GetUser gets a user by ID
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	vars := mux.Vars(r)
	userID := vars["id"]

	user, err := h.service.GetUser(ctx, model.UserID(userID))
	if err != nil {
		h.respondError(w, http.StatusNotFound, "User not found")
		return
	}

	resp := h.userToDTO(user)
	h.respondJSON(w, http.StatusOK, resp)
}

// ListUsers lists all users (admin only)
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: Check admin permission

	users, err := h.service.ListUsers(ctx, 0, 100)
	if err != nil {
		h.logger.Error("Failed to list users", "error", err)
		h.respondError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	resp := make([]dto.UserResponse, len(users))
	for i, user := range users {
		resp[i] = h.userToDTO(user)
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// BlockUser blocks a user (admin only)
func (h *UserHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	vars := mux.Vars(r)
	userID := vars["id"]

	err := h.service.BlockUser(ctx, model.UserID(userID))
	if err != nil {
		h.logger.Error("Failed to block user", "error", err, "user_id", userID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "User blocked successfully"})
}

// UnblockUser unblocks a user (admin only)
func (h *UserHandler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	vars := mux.Vars(r)
	userID := vars["id"]

	err := h.service.UnblockUser(ctx, model.UserID(userID))
	if err != nil {
		h.logger.Error("Failed to unblock user", "error", err, "user_id", userID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "User unblocked successfully"})
}

// ChangePassword changes user password
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.service.ChangePassword(ctx, model.UserID(userID), req.CurrentPassword, req.NewPassword)
	if err != nil {
		h.logger.Error("Failed to change password", "error", err, "user_id", userID)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// CreateOrganization creates a new organization
func (h *UserHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	org, err := h.service.CreateOrganization(ctx, model.UserID(userID), req.Name, req.Description)
	if err != nil {
		h.logger.Error("Failed to create organization", "error", err)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := dto.OrganizationResponse{
		ID:          org.ID().String(),
		Name:        org.Name(),
		Description: org.Description(),
		CreatedAt:   org.CreatedAt(),
	}
	h.respondJSON(w, http.StatusCreated, resp)
}

// AddOrganizationMember adds a member to an organization
func (h *UserHandler) AddOrganizationMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	vars := mux.Vars(r)
	orgID := vars["id"]

	var req dto.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.service.AddOrganizationMember(ctx, model.OrganizationID(orgID), model.UserID(req.UserID), model.Role(req.Role))
	if err != nil {
		h.logger.Error("Failed to add organization member", "error", err)
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Member added successfully"})
}

// Helper methods

func (h *UserHandler) userToDTO(user *model.User) dto.UserResponse {
	roles := make([]string, len(user.Roles()))
	for i, role := range user.Roles() {
		roles[i] = string(role)
	}

	return dto.UserResponse{
		ID:            user.ID().String(),
		Email:         user.Email(),
		Username:      user.Username(),
		FirstName:     user.FirstName(),
		LastName:      user.LastName(),
		FullName:      user.FullName(),
		AvatarURL:     user.AvatarURL(),
		Status:        string(user.Status()),
		EmailVerified: user.EmailVerified(),
		Roles:         roles,
		CreatedAt:     user.CreatedAt(),
		UpdatedAt:     user.UpdatedAt(),
	}
}

func (h *UserHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *UserHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

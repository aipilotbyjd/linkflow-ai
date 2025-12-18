package dto

import (
	"errors"
	"time"
)

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// Validate validates the register request
func (r *RegisterRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Username == "" {
		return errors.New("username is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	FullName      string    `json:"fullName"`
	AvatarURL     string    `json:"avatarUrl,omitempty"`
	Status        string    `json:"status"`
	EmailVerified bool      `json:"emailVerified"`
	Roles         []string  `json:"roles"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

// CreateOrganizationRequest represents an organization creation request
type CreateOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// OrganizationResponse represents an organization response
type OrganizationResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AddMemberRequest represents a request to add a member to an organization
type AddMemberRequest struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

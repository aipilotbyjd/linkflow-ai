package repository

import (
	"context"
	"errors"
	
	"github.com/linkflow-ai/linkflow-ai/internal/user/domain/model"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	
	// ErrOrganizationNotFound is returned when an organization is not found
	ErrOrganizationNotFound = errors.New("organization not found")
	
	// ErrDuplicateEmail is returned when email already exists
	ErrDuplicateEmail = errors.New("email already exists")
	
	// ErrDuplicateUsername is returned when username already exists
	ErrDuplicateUsername = errors.New("username already exists")
)

// UserRepository defines the interface for user persistence
type UserRepository interface {
	// Save saves a new user
	Save(ctx context.Context, user *model.User) error
	
	// FindByID finds a user by ID
	FindByID(ctx context.Context, id model.UserID) (*model.User, error)
	
	// FindByEmail finds a user by email
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	
	// FindByUsername finds a user by username
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	
	// FindAll finds all users with pagination
	FindAll(ctx context.Context, offset, limit int) ([]*model.User, error)
	
	// Update updates an existing user
	Update(ctx context.Context, user *model.User) error
	
	// Delete deletes a user
	Delete(ctx context.Context, id model.UserID) error
	
	// ExistsByEmail checks if a user with email exists
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	
	// ExistsByUsername checks if a user with username exists
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	
	// Count counts total users
	Count(ctx context.Context) (int64, error)
}

// OrganizationRepository defines the interface for organization persistence
type OrganizationRepository interface {
	// Save saves a new organization
	Save(ctx context.Context, org *model.Organization) error
	
	// FindByID finds an organization by ID
	FindByID(ctx context.Context, id model.OrganizationID) (*model.Organization, error)
	
	// FindBySlug finds an organization by slug
	FindBySlug(ctx context.Context, slug string) (*model.Organization, error)
	
	// FindByUserID finds organizations where user is a member
	FindByUserID(ctx context.Context, userID string) ([]*model.Organization, error)
	
	// Update updates an existing organization
	Update(ctx context.Context, org *model.Organization) error
	
	// Delete deletes an organization
	Delete(ctx context.Context, id model.OrganizationID) error
	
	// ExistsBySlug checks if an organization with slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

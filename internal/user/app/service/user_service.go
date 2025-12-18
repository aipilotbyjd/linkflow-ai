package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/user/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/user/domain/repository"
)

// UserService handles user application logic
type UserService struct {
	userRepo repository.UserRepository
	orgRepo  repository.OrganizationRepository
	logger   logger.Logger
	jwtSecret []byte
}

// NewUserService creates a new user service
func NewUserService(userRepo repository.UserRepository, orgRepo repository.OrganizationRepository, logger logger.Logger) *UserService {
	return &UserService{
		userRepo:  userRepo,
		orgRepo:   orgRepo,
		logger:    logger,
		jwtSecret: []byte("your-secret-key"), // TODO: Get from config
	}
}

// RegisterCommand represents a user registration command
type RegisterCommand struct {
	Email     string
	Username  string
	Password  string
	FirstName string
	LastName  string
}

// Register registers a new user
func (s *UserService) Register(ctx context.Context, cmd RegisterCommand) (*model.User, error) {
	// Check if user already exists
	existsByEmail, err := s.userRepo.ExistsByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if existsByEmail {
		return nil, errors.New("email already registered")
	}

	existsByUsername, err := s.userRepo.ExistsByUsername(ctx, cmd.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if existsByUsername {
		return nil, errors.New("username already taken")
	}

	// Create user
	user, err := model.NewUser(cmd.Email, cmd.Username, cmd.Password, cmd.FirstName, cmd.LastName)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Save user
	if err := s.userRepo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	s.logger.Info("User registered successfully", "user_id", user.ID(), "email", cmd.Email)
	return user, nil
}

// Login authenticates a user and returns a JWT token
func (s *UserService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, "", errors.New("invalid credentials")
		}
		return nil, "", fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := user.VerifyPassword(password); err != nil {
		// Update user with failed attempt
		_ = s.userRepo.Update(ctx, user)
		return nil, "", err
	}

	// Update last login
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update last login", "error", err, "user_id", user.ID())
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("User logged in successfully", "user_id", user.ID(), "email", email)
	return user, token, nil
}

// GetUser gets a user by ID
func (s *UserService) GetUser(ctx context.Context, userID model.UserID) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// ListUsers lists all users
func (s *UserService) ListUsers(ctx context.Context, offset, limit int) ([]*model.User, error) {
	users, err := s.userRepo.FindAll(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// UpdateProfileCommand represents a profile update command
type UpdateProfileCommand struct {
	FirstName string
	LastName  string
	AvatarURL string
}

// UpdateProfile updates a user's profile
func (s *UserService) UpdateProfile(ctx context.Context, userID model.UserID, cmd UpdateProfileCommand) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.UpdateProfile(cmd.FirstName, cmd.LastName, cmd.AvatarURL)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("User profile updated", "user_id", userID)
	return user, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, userID model.UserID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := user.ChangePassword(currentPassword, newPassword); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("User password changed", "user_id", userID)
	return nil
}

// BlockUser blocks a user
func (s *UserService) BlockUser(ctx context.Context, userID model.UserID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := user.Block(); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("User blocked", "user_id", userID)
	return nil
}

// UnblockUser unblocks a user
func (s *UserService) UnblockUser(ctx context.Context, userID model.UserID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if err := user.Unblock(); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("User unblocked", "user_id", userID)
	return nil
}

// CreateOrganization creates a new organization
func (s *UserService) CreateOrganization(ctx context.Context, userID model.UserID, name, description string) (*model.Organization, error) {
	// Create organization
	org, err := model.NewOrganization(name, description, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Save organization
	if err := s.orgRepo.Save(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to save organization: %w", err)
	}

	// Update user with organization
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.SetOrganization(org.ID().String())
	user.AssignRole(model.RoleOwner)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("Organization created", "org_id", org.ID(), "user_id", userID)
	return org, nil
}

// AddOrganizationMember adds a member to an organization
func (s *UserService) AddOrganizationMember(ctx context.Context, orgID model.OrganizationID, userID model.UserID, role model.Role) error {
	// Get organization
	org, err := s.orgRepo.FindByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	// Add member
	if err := org.AddMember(userID.String(), role); err != nil {
		return err
	}

	// Update organization
	if err := s.orgRepo.Update(ctx, org); err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	// Update user with organization
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.SetOrganization(orgID.String())
	user.AssignRole(role)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("Member added to organization", "org_id", orgID, "user_id", userID, "role", role)
	return nil
}

// generateToken generates a JWT token for a user
func (s *UserService) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID().String(),
		"email":    user.Email(),
		"username": user.Username(),
		"roles":    user.Roles(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

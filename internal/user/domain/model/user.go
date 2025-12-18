package model

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
)

// UserID represents a unique user identifier
type UserID string

// NewUserID creates a new user ID
func NewUserID() UserID {
	return UserID(uuid.New().String())
}

func (id UserID) String() string {
	return string(id)
}

// UserStatus represents the status of a user
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBlocked  UserStatus = "blocked"
	UserStatusDeleted  UserStatus = "deleted"
)

// Role represents a user role
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
	RoleOwner Role = "owner"
)

// User aggregate root
type User struct {
	id               UserID
	email            string
	username         string
	passwordHash     string
	firstName        string
	lastName         string
	avatarURL        string
	status           UserStatus
	emailVerified    bool
	roles            []Role
	organizationID   *string
	metadata         map[string]interface{}
	lastLoginAt      *time.Time
	failedAttempts   int
	lockedUntil      *time.Time
	createdAt        time.Time
	updatedAt        time.Time
	version          int
}

// NewUser creates a new user
func NewUser(email, username, password, firstName, lastName string) (*User, error) {
	// Validate email
	if !isValidEmail(email) {
		return nil, errors.New("invalid email format")
	}

	// Validate username
	if err := validateUsername(username); err != nil {
		return nil, err
	}

	// Validate password
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	// Hash password
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &User{
		id:             NewUserID(),
		email:          email,
		username:       username,
		passwordHash:   hashedPassword,
		firstName:      firstName,
		lastName:       lastName,
		status:         UserStatusActive,
		emailVerified:  false,
		roles:          []Role{RoleUser},
		metadata:       make(map[string]interface{}),
		failedAttempts: 0,
		createdAt:      now,
		updatedAt:      now,
		version:        0,
	}

	return user, nil
}

// Getters
func (u *User) ID() UserID                     { return u.id }
func (u *User) Email() string                  { return u.email }
func (u *User) Username() string               { return u.username }
func (u *User) FirstName() string              { return u.firstName }
func (u *User) LastName() string               { return u.lastName }
func (u *User) FullName() string               { return fmt.Sprintf("%s %s", u.firstName, u.lastName) }
func (u *User) AvatarURL() string              { return u.avatarURL }
func (u *User) Status() UserStatus             { return u.status }
func (u *User) EmailVerified() bool            { return u.emailVerified }
func (u *User) Roles() []Role                  { return u.roles }
func (u *User) OrganizationID() *string        { return u.organizationID }
func (u *User) Metadata() map[string]interface{} { return u.metadata }
func (u *User) LastLoginAt() *time.Time        { return u.lastLoginAt }
func (u *User) CreatedAt() time.Time           { return u.createdAt }
func (u *User) UpdatedAt() time.Time           { return u.updatedAt }
func (u *User) Version() int                   { return u.version }

// VerifyPassword verifies a password against the user's password hash
func (u *User) VerifyPassword(password string) error {
	// Check if account is locked
	if u.lockedUntil != nil && u.lockedUntil.After(time.Now()) {
		return errors.New("account is locked")
	}

	// Verify password
	err := bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password))
	if err != nil {
		u.recordFailedAttempt()
		return errors.New("invalid credentials")
	}

	// Reset failed attempts on successful login
	u.failedAttempts = 0
	u.lockedUntil = nil
	now := time.Now()
	u.lastLoginAt = &now
	u.updatedAt = time.Now()

	return nil
}

// ChangePassword changes the user's password
func (u *User) ChangePassword(currentPassword, newPassword string) error {
	// Verify current password
	if err := u.VerifyPassword(currentPassword); err != nil {
		return errors.New("current password is incorrect")
	}

	// Validate new password
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	u.passwordHash = hashedPassword
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// UpdateProfile updates user profile information
func (u *User) UpdateProfile(firstName, lastName, avatarURL string) {
	if firstName != "" {
		u.firstName = firstName
	}
	if lastName != "" {
		u.lastName = lastName
	}
	if avatarURL != "" {
		u.avatarURL = avatarURL
	}
	u.updatedAt = time.Now()
	u.version++
}

// VerifyEmail marks the user's email as verified
func (u *User) VerifyEmail() error {
	if u.emailVerified {
		return errors.New("email already verified")
	}

	u.emailVerified = true
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// Block blocks the user account
func (u *User) Block() error {
	if u.status == UserStatusBlocked {
		return errors.New("user is already blocked")
	}

	u.status = UserStatusBlocked
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// Unblock unblocks the user account
func (u *User) Unblock() error {
	if u.status != UserStatusBlocked {
		return errors.New("user is not blocked")
	}

	u.status = UserStatusActive
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// Delete soft deletes the user
func (u *User) Delete() error {
	if u.status == UserStatusDeleted {
		return errors.New("user is already deleted")
	}

	u.status = UserStatusDeleted
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// AssignRole assigns a role to the user
func (u *User) AssignRole(role Role) error {
	// Check if role already assigned
	for _, r := range u.roles {
		if r == role {
			return errors.New("role already assigned")
		}
	}

	u.roles = append(u.roles, role)
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// RemoveRole removes a role from the user
func (u *User) RemoveRole(role Role) error {
	// Cannot remove last role
	if len(u.roles) == 1 && u.roles[0] == role {
		return errors.New("cannot remove last role")
	}

	newRoles := []Role{}
	found := false
	for _, r := range u.roles {
		if r != role {
			newRoles = append(newRoles, r)
		} else {
			found = true
		}
	}

	if !found {
		return errors.New("role not assigned")
	}

	u.roles = newRoles
	u.updatedAt = time.Now()
	u.version++

	return nil
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role Role) bool {
	for _, r := range u.roles {
		if r == role {
			return true
		}
	}
	return false
}

// SetOrganization sets the user's organization
func (u *User) SetOrganization(orgID string) {
	u.organizationID = &orgID
	u.updatedAt = time.Now()
	u.version++
}

// recordFailedAttempt records a failed login attempt
func (u *User) recordFailedAttempt() {
	u.failedAttempts++
	
	// Lock account after 5 failed attempts
	if u.failedAttempts >= 5 {
		lockUntil := time.Now().Add(15 * time.Minute)
		u.lockedUntil = &lockUntil
		u.status = UserStatusBlocked
	}
	
	u.updatedAt = time.Now()
}

// Helper functions

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 30 {
		return errors.New("username must be between 3 and 30 characters")
	}
	
	// Check for valid characters (alphanumeric, underscore, hyphen)
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}
	
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	
	// Check for at least one uppercase letter
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return errors.New("password must contain at least one uppercase letter")
	}
	
	// Check for at least one lowercase letter
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return errors.New("password must contain at least one lowercase letter")
	}
	
	// Check for at least one digit
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return errors.New("password must contain at least one digit")
	}
	
	return nil
}

func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// ReconstructUser reconstructs a user from persisted state
func ReconstructUser(
	id UserID,
	email string,
	username string,
	passwordHash string,
	firstName string,
	lastName string,
	avatarURL string,
	status UserStatus,
	emailVerified bool,
	roles []Role,
	organizationID *string,
	metadata map[string]interface{},
	lastLoginAt *time.Time,
	failedAttempts int,
	lockedUntil *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
	version int,
) *User {
	return &User{
		id:             id,
		email:          email,
		username:       username,
		passwordHash:   passwordHash,
		firstName:      firstName,
		lastName:       lastName,
		avatarURL:      avatarURL,
		status:         status,
		emailVerified:  emailVerified,
		roles:          roles,
		organizationID: organizationID,
		metadata:       metadata,
		lastLoginAt:    lastLoginAt,
		failedAttempts: failedAttempts,
		lockedUntil:    lockedUntil,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
		version:        version,
	}
}

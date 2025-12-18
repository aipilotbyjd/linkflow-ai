package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock User struct for testing (matches internal/user/domain/model)
type User struct {
	ID             string
	Email          string
	PasswordHash   string
	FirstName      string
	LastName       string
	OrganizationID string
	Roles          []string
	Status         UserStatus
	MFAEnabled     bool
	FailedAttempts int
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusPending  UserStatus = "pending"
	UserStatusLocked   UserStatus = "locked"
)

func NewUser(email, firstName, lastName string) (*User, error) {
	if email == "" {
		return nil, assert.AnError
	}
	if firstName == "" {
		return nil, assert.AnError
	}
	if lastName == "" {
		return nil, assert.AnError
	}

	now := time.Now()
	return &User{
		ID:        "user-" + email,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Roles:     []string{"user"},
		Status:    UserStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) Activate() error {
	if u.Status == UserStatusLocked {
		return assert.AnError
	}
	u.Status = UserStatusActive
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) Deactivate() {
	u.Status = UserStatusInactive
	u.UpdatedAt = time.Now()
}

func (u *User) Lock(duration time.Duration) {
	u.Status = UserStatusLocked
	lockedUntil := time.Now().Add(duration)
	u.LockedUntil = &lockedUntil
	u.UpdatedAt = time.Now()
}

func (u *User) Unlock() {
	u.Status = UserStatusActive
	u.LockedUntil = nil
	u.FailedAttempts = 0
	u.UpdatedAt = time.Now()
}

func (u *User) RecordFailedLogin() bool {
	u.FailedAttempts++
	if u.FailedAttempts >= 5 {
		u.Lock(15 * time.Minute)
		return true
	}
	return false
}

func (u *User) ResetFailedAttempts() {
	u.FailedAttempts = 0
}

func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (u *User) AddRole(role string) {
	if !u.HasRole(role) {
		u.Roles = append(u.Roles, role)
		u.UpdatedAt = time.Now()
	}
}

func (u *User) RemoveRole(role string) {
	for i, r := range u.Roles {
		if r == role {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
			u.UpdatedAt = time.Now()
			return
		}
	}
}

func (u *User) EnableMFA() {
	u.MFAEnabled = true
	u.UpdatedAt = time.Now()
}

func (u *User) DisableMFA() {
	u.MFAEnabled = false
	u.UpdatedAt = time.Now()
}

func (u *User) IsLocked() bool {
	if u.Status != UserStatusLocked {
		return false
	}
	if u.LockedUntil != nil && time.Now().After(*u.LockedUntil) {
		u.Unlock()
		return false
	}
	return true
}

// Tests
func TestNewUser(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		firstName string
		lastName  string
		wantErr   bool
	}{
		{
			name:      "valid user",
			email:     "test@example.com",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   false,
		},
		{
			name:      "empty email",
			email:     "",
			firstName: "John",
			lastName:  "Doe",
			wantErr:   true,
		},
		{
			name:      "empty first name",
			email:     "test@example.com",
			firstName: "",
			lastName:  "Doe",
			wantErr:   true,
		},
		{
			name:      "empty last name",
			email:     "test@example.com",
			firstName: "John",
			lastName:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.email, tt.firstName, tt.lastName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)

				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, tt.firstName, user.FirstName)
				assert.Equal(t, tt.lastName, user.LastName)
				assert.Equal(t, UserStatusPending, user.Status)
				assert.Contains(t, user.Roles, "user")
			}
		})
	}
}

func TestUserFullName(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.FullName())
}

func TestUserActivation(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	assert.Equal(t, UserStatusPending, user.Status)

	err = user.Activate()
	assert.NoError(t, err)
	assert.Equal(t, UserStatusActive, user.Status)
}

func TestUserDeactivation(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	user.Activate()
	user.Deactivate()

	assert.Equal(t, UserStatusInactive, user.Status)
}

func TestUserLocking(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	user.Activate()
	user.Lock(15 * time.Minute)

	assert.Equal(t, UserStatusLocked, user.Status)
	assert.NotNil(t, user.LockedUntil)
	assert.True(t, user.IsLocked())

	user.Unlock()
	assert.Equal(t, UserStatusActive, user.Status)
	assert.Nil(t, user.LockedUntil)
	assert.False(t, user.IsLocked())
}

func TestUserFailedLoginAttempts(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	user.Activate()

	// First 4 attempts should not lock
	for i := 0; i < 4; i++ {
		locked := user.RecordFailedLogin()
		assert.False(t, locked)
		assert.Equal(t, i+1, user.FailedAttempts)
	}

	// 5th attempt should lock
	locked := user.RecordFailedLogin()
	assert.True(t, locked)
	assert.Equal(t, UserStatusLocked, user.Status)
}

func TestUserRoles(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	// Default role
	assert.True(t, user.HasRole("user"))
	assert.False(t, user.HasRole("admin"))

	// Add role
	user.AddRole("admin")
	assert.True(t, user.HasRole("admin"))
	assert.Len(t, user.Roles, 2)

	// Add duplicate role (should not add)
	user.AddRole("admin")
	assert.Len(t, user.Roles, 2)

	// Remove role
	user.RemoveRole("admin")
	assert.False(t, user.HasRole("admin"))
	assert.Len(t, user.Roles, 1)
}

func TestUserMFA(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	assert.False(t, user.MFAEnabled)

	user.EnableMFA()
	assert.True(t, user.MFAEnabled)

	user.DisableMFA()
	assert.False(t, user.MFAEnabled)
}

func TestUserCannotActivateWhenLocked(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	user.Lock(15 * time.Minute)

	err = user.Activate()
	assert.Error(t, err)
	assert.Equal(t, UserStatusLocked, user.Status)
}

func TestUserAutoUnlockAfterDuration(t *testing.T) {
	user, err := NewUser("test@example.com", "John", "Doe")
	require.NoError(t, err)

	user.Activate()
	
	// Lock for a very short duration (past)
	pastTime := time.Now().Add(-1 * time.Minute)
	user.Status = UserStatusLocked
	user.LockedUntil = &pastTime

	// IsLocked should auto-unlock
	assert.False(t, user.IsLocked())
	assert.Equal(t, UserStatusActive, user.Status)
}

package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OrganizationID represents a unique organization identifier
type OrganizationID string

// NewOrganizationID creates a new organization ID
func NewOrganizationID() OrganizationID {
	return OrganizationID(uuid.New().String())
}

func (id OrganizationID) String() string {
	return string(id)
}

// OrganizationStatus represents the status of an organization
type OrganizationStatus string

const (
	OrganizationStatusActive   OrganizationStatus = "active"
	OrganizationStatusInactive OrganizationStatus = "inactive"
	OrganizationStatusSuspended OrganizationStatus = "suspended"
)

// Member represents an organization member
type Member struct {
	UserID   string    `json:"userId"`
	Role     Role      `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

// Organization aggregate
type Organization struct {
	id          OrganizationID
	name        string
	slug        string
	description string
	logoURL     string
	website     string
	status      OrganizationStatus
	createdBy   string
	members     []Member
	settings    map[string]interface{}
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewOrganization creates a new organization
func NewOrganization(name, description, createdBy string) (*Organization, error) {
	if name == "" {
		return nil, errors.New("organization name is required")
	}
	if createdBy == "" {
		return nil, errors.New("created by is required")
	}

	slug := generateSlug(name)
	now := time.Now()

	org := &Organization{
		id:          NewOrganizationID(),
		name:        name,
		slug:        slug,
		description: description,
		status:      OrganizationStatusActive,
		createdBy:   createdBy,
		members: []Member{
			{
				UserID:   createdBy,
				Role:     RoleOwner,
				JoinedAt: now,
			},
		},
		settings:  make(map[string]interface{}),
		createdAt: now,
		updatedAt: now,
		version:   0,
	}

	return org, nil
}

// Getters
func (o *Organization) ID() OrganizationID           { return o.id }
func (o *Organization) Name() string                 { return o.name }
func (o *Organization) Slug() string                 { return o.slug }
func (o *Organization) Description() string          { return o.description }
func (o *Organization) LogoURL() string              { return o.logoURL }
func (o *Organization) Website() string              { return o.website }
func (o *Organization) Status() OrganizationStatus   { return o.status }
func (o *Organization) CreatedBy() string            { return o.createdBy }
func (o *Organization) Members() []Member            { return o.members }
func (o *Organization) Settings() map[string]interface{} { return o.settings }
func (o *Organization) CreatedAt() time.Time         { return o.createdAt }
func (o *Organization) UpdatedAt() time.Time         { return o.updatedAt }
func (o *Organization) Version() int                 { return o.version }

// UpdateDetails updates organization details
func (o *Organization) UpdateDetails(name, description, logoURL, website string) {
	if name != "" && name != o.name {
		o.name = name
		o.slug = generateSlug(name)
	}
	if description != "" {
		o.description = description
	}
	if logoURL != "" {
		o.logoURL = logoURL
	}
	if website != "" {
		o.website = website
	}
	o.updatedAt = time.Now()
	o.version++
}

// AddMember adds a member to the organization
func (o *Organization) AddMember(userID string, role Role) error {
	// Check if member already exists
	for _, member := range o.members {
		if member.UserID == userID {
			return errors.New("user is already a member")
		}
	}

	o.members = append(o.members, Member{
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	})

	o.updatedAt = time.Now()
	o.version++
	return nil
}

// RemoveMember removes a member from the organization
func (o *Organization) RemoveMember(userID string) error {
	if userID == o.createdBy {
		return errors.New("cannot remove organization owner")
	}

	newMembers := []Member{}
	found := false
	for _, member := range o.members {
		if member.UserID != userID {
			newMembers = append(newMembers, member)
		} else {
			found = true
		}
	}

	if !found {
		return errors.New("user is not a member")
	}

	o.members = newMembers
	o.updatedAt = time.Now()
	o.version++
	return nil
}

// UpdateMemberRole updates a member's role
func (o *Organization) UpdateMemberRole(userID string, newRole Role) error {
	if userID == o.createdBy && newRole != RoleOwner {
		return errors.New("cannot change owner role")
	}

	found := false
	for i, member := range o.members {
		if member.UserID == userID {
			o.members[i].Role = newRole
			found = true
			break
		}
	}

	if !found {
		return errors.New("user is not a member")
	}

	o.updatedAt = time.Now()
	o.version++
	return nil
}

// GetMember gets a member by user ID
func (o *Organization) GetMember(userID string) (*Member, error) {
	for _, member := range o.members {
		if member.UserID == userID {
			return &member, nil
		}
	}
	return nil, errors.New("member not found")
}

// HasMember checks if a user is a member
func (o *Organization) HasMember(userID string) bool {
	for _, member := range o.members {
		if member.UserID == userID {
			return true
		}
	}
	return false
}

// Suspend suspends the organization
func (o *Organization) Suspend() error {
	if o.status == OrganizationStatusSuspended {
		return errors.New("organization is already suspended")
	}

	o.status = OrganizationStatusSuspended
	o.updatedAt = time.Now()
	o.version++
	return nil
}

// Activate activates the organization
func (o *Organization) Activate() error {
	if o.status == OrganizationStatusActive {
		return errors.New("organization is already active")
	}

	o.status = OrganizationStatusActive
	o.updatedAt = time.Now()
	o.version++
	return nil
}

// UpdateSettings updates organization settings
func (o *Organization) UpdateSettings(settings map[string]interface{}) {
	for key, value := range settings {
		o.settings[key] = value
	}
	o.updatedAt = time.Now()
	o.version++
}

// Helper functions

func generateSlug(name string) string {
	// Simple slug generation - in production, use a proper slug library
	slug := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				r = r + 32 // Convert to lowercase
			}
			slug += string(r)
		} else if r == ' ' || r == '-' || r == '_' {
			if slug != "" && slug[len(slug)-1] != '-' {
				slug += "-"
			}
		}
	}
	
	// Add random suffix to ensure uniqueness
	slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
	return slug
}

// ReconstructOrganization reconstructs an organization from persisted state
func ReconstructOrganization(
	id OrganizationID,
	name string,
	slug string,
	description string,
	logoURL string,
	website string,
	status OrganizationStatus,
	createdBy string,
	members []Member,
	settings map[string]interface{},
	createdAt time.Time,
	updatedAt time.Time,
	version int,
) *Organization {
	return &Organization{
		id:          id,
		name:        name,
		slug:        slug,
		description: description,
		logoURL:     logoURL,
		website:     website,
		status:      status,
		createdBy:   createdBy,
		members:     members,
		settings:    settings,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}
}

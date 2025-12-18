// Package model defines workspace domain models
package model

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// WorkspaceID represents a unique workspace identifier
type WorkspaceID string

// NewWorkspaceID creates a new workspace ID
func NewWorkspaceID() WorkspaceID {
	return WorkspaceID(uuid.New().String())
}

func (id WorkspaceID) String() string {
	return string(id)
}

// Workspace represents a workspace/team
type Workspace struct {
	ID          WorkspaceID
	Name        string
	Slug        string
	Description string
	OwnerID     string
	Plan        Plan
	Settings    WorkspaceSettings
	Limits      WorkspaceLimits
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Plan represents subscription plan
type Plan string

const (
	PlanFree       Plan = "free"
	PlanStarter    Plan = "starter"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

// WorkspaceSettings holds workspace-specific settings
type WorkspaceSettings struct {
	DefaultTimezone    string `json:"defaultTimezone"`
	AllowPublicWorkflows bool `json:"allowPublicWorkflows"`
	RequireApproval    bool   `json:"requireApproval"`
	AuditLogEnabled    bool   `json:"auditLogEnabled"`
	SSOEnabled         bool   `json:"ssoEnabled"`
}

// WorkspaceLimits defines resource limits
type WorkspaceLimits struct {
	MaxMembers         int   `json:"maxMembers"`
	MaxWorkflows       int   `json:"maxWorkflows"`
	MaxExecutionsMonth int   `json:"maxExecutionsPerMonth"`
	MaxCredentials     int   `json:"maxCredentials"`
	MaxWebhooks        int   `json:"maxWebhooks"`
	RetentionDays      int   `json:"retentionDays"`
}

// GetPlanLimits returns limits for a plan
func GetPlanLimits(plan Plan) WorkspaceLimits {
	switch plan {
	case PlanFree:
		return WorkspaceLimits{
			MaxMembers:         3,
			MaxWorkflows:       5,
			MaxExecutionsMonth: 100,
			MaxCredentials:     5,
			MaxWebhooks:        2,
			RetentionDays:      7,
		}
	case PlanStarter:
		return WorkspaceLimits{
			MaxMembers:         10,
			MaxWorkflows:       25,
			MaxExecutionsMonth: 1000,
			MaxCredentials:     20,
			MaxWebhooks:        10,
			RetentionDays:      30,
		}
	case PlanPro:
		return WorkspaceLimits{
			MaxMembers:         50,
			MaxWorkflows:       100,
			MaxExecutionsMonth: 10000,
			MaxCredentials:     100,
			MaxWebhooks:        50,
			RetentionDays:      90,
		}
	case PlanEnterprise:
		return WorkspaceLimits{
			MaxMembers:         -1,
			MaxWorkflows:       -1,
			MaxExecutionsMonth: -1,
			MaxCredentials:     -1,
			MaxWebhooks:        -1,
			RetentionDays:      365,
		}
	default:
		return GetPlanLimits(PlanFree)
	}
}

// NewWorkspace creates a new workspace
func NewWorkspace(name, slug, ownerID string, plan Plan) (*Workspace, error) {
	if name == "" {
		return nil, errors.New("workspace name is required")
	}
	if !slugRegex.MatchString(slug) {
		return nil, errors.New("invalid slug format")
	}

	now := time.Now()
	return &Workspace{
		ID:          NewWorkspaceID(),
		Name:        name,
		Slug:        slug,
		OwnerID:     ownerID,
		Plan:        plan,
		Limits:      GetPlanLimits(plan),
		Settings: WorkspaceSettings{
			DefaultTimezone:    "UTC",
			AllowPublicWorkflows: false,
			AuditLogEnabled:    true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// MemberRole represents a member's role in a workspace
type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleAdmin  MemberRole = "admin"
	RoleMember MemberRole = "member"
	RoleViewer MemberRole = "viewer"
)

// WorkspaceMember represents a workspace member
type WorkspaceMember struct {
	ID          string
	WorkspaceID string
	UserID      string
	Role        MemberRole
	InvitedBy   string
	JoinedAt    time.Time
	UpdatedAt   time.Time
}

// NewWorkspaceMember creates a new workspace member
func NewWorkspaceMember(workspaceID, userID string, role MemberRole, invitedBy string) *WorkspaceMember {
	now := time.Now()
	return &WorkspaceMember{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		InvitedBy:   invitedBy,
		JoinedAt:    now,
		UpdatedAt:   now,
	}
}

// CanManageMembers checks if role can manage members
func (r MemberRole) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanManageWorkflows checks if role can manage workflows
func (r MemberRole) CanManageWorkflows() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}

// CanViewWorkflows checks if role can view workflows
func (r MemberRole) CanViewWorkflows() bool {
	return true
}

// CanManageSettings checks if role can manage settings
func (r MemberRole) CanManageSettings() bool {
	return r == RoleOwner || r == RoleAdmin
}

// InvitationStatus represents invitation status
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationDeclined InvitationStatus = "declined"
	InvitationExpired  InvitationStatus = "expired"
)

// WorkspaceInvitation represents an invitation to join a workspace
type WorkspaceInvitation struct {
	ID          string
	WorkspaceID string
	Email       string
	Role        MemberRole
	Token       string
	InvitedBy   string
	Status      InvitationStatus
	ExpiresAt   time.Time
	CreatedAt   time.Time
	AcceptedAt  *time.Time
}

// NewWorkspaceInvitation creates a new invitation
func NewWorkspaceInvitation(workspaceID, email string, role MemberRole, invitedBy string) (*WorkspaceInvitation, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &WorkspaceInvitation{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Email:       email,
		Role:        role,
		Token:       token,
		InvitedBy:   invitedBy,
		Status:      InvitationPending,
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
		CreatedAt:   now,
	}, nil
}

// IsValid checks if invitation is valid
func (i *WorkspaceInvitation) IsValid() bool {
	return i.Status == InvitationPending && time.Now().Before(i.ExpiresAt)
}

// Accept accepts the invitation
func (i *WorkspaceInvitation) Accept() {
	now := time.Now()
	i.Status = InvitationAccepted
	i.AcceptedAt = &now
}

// Decline declines the invitation
func (i *WorkspaceInvitation) Decline() {
	i.Status = InvitationDeclined
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID          string
	WorkspaceID string
	UserID      string
	Action      string
	Resource    string
	ResourceID  string
	Details     map[string]interface{}
	IPAddress   string
	UserAgent   string
	CreatedAt   time.Time
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(workspaceID, userID, action, resource, resourceID string, details map[string]interface{}) *AuditLog {
	return &AuditLog{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Details:     details,
		CreatedAt:   time.Now(),
	}
}

// Common audit actions
const (
	ActionCreate         = "create"
	ActionUpdate         = "update"
	ActionDelete         = "delete"
	ActionExecute        = "execute"
	ActionInvite         = "invite"
	ActionRemoveMember   = "remove_member"
	ActionChangeRole     = "change_role"
	ActionLogin          = "login"
	ActionLogout         = "logout"
	ActionAPIKeyCreated  = "api_key_created"
	ActionAPIKeyRevoked  = "api_key_revoked"
)

// Resource types
const (
	ResourceWorkflow    = "workflow"
	ResourceCredential  = "credential"
	ResourceWebhook     = "webhook"
	ResourceSchedule    = "schedule"
	ResourceMember      = "member"
	ResourceWorkspace   = "workspace"
	ResourceAPIKey      = "api_key"
)

// Errors
var (
	ErrWorkspaceNotFound    = errors.New("workspace not found")
	ErrMemberNotFound       = errors.New("member not found")
	ErrInvitationNotFound   = errors.New("invitation not found")
	ErrInvitationExpired    = errors.New("invitation has expired")
	ErrAlreadyMember        = errors.New("user is already a member")
	ErrCannotRemoveOwner    = errors.New("cannot remove workspace owner")
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrMemberLimitReached   = errors.New("member limit reached")
)

func generateToken(length int) (string, error) {
	// Implementation similar to auth model
	return uuid.New().String(), nil
}

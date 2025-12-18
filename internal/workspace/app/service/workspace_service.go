// Package service provides workspace business logic
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/workspace/domain/model"
)

// WorkspaceRepository defines workspace persistence operations
type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *model.Workspace) error
	FindByID(ctx context.Context, id string) (*model.Workspace, error)
	FindBySlug(ctx context.Context, slug string) (*model.Workspace, error)
	FindByOwner(ctx context.Context, ownerID string) ([]*model.Workspace, error)
	Update(ctx context.Context, workspace *model.Workspace) error
	Delete(ctx context.Context, id string) error
	SlugExists(ctx context.Context, slug string) (bool, error)
}

// MemberRepository defines member persistence operations
type MemberRepository interface {
	Add(ctx context.Context, member *model.WorkspaceMember) error
	FindByID(ctx context.Context, id string) (*model.WorkspaceMember, error)
	FindByWorkspaceAndUser(ctx context.Context, workspaceID, userID string) (*model.WorkspaceMember, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*model.WorkspaceMember, error)
	ListByUser(ctx context.Context, userID string) ([]*model.WorkspaceMember, error)
	UpdateRole(ctx context.Context, id string, role model.MemberRole) error
	Remove(ctx context.Context, id string) error
	CountByWorkspace(ctx context.Context, workspaceID string) (int, error)
}

// InvitationRepository defines invitation persistence operations
type InvitationRepository interface {
	Create(ctx context.Context, invitation *model.WorkspaceInvitation) error
	FindByToken(ctx context.Context, token string) (*model.WorkspaceInvitation, error)
	FindByEmail(ctx context.Context, workspaceID, email string) (*model.WorkspaceInvitation, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*model.WorkspaceInvitation, error)
	ListPendingByEmail(ctx context.Context, email string) ([]*model.WorkspaceInvitation, error)
	Update(ctx context.Context, invitation *model.WorkspaceInvitation) error
	Delete(ctx context.Context, id string) error
}

// AuditLogRepository defines audit log persistence operations
type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	ListByWorkspace(ctx context.Context, workspaceID string, offset, limit int) ([]*model.AuditLog, int64, error)
}

// EmailService defines email operations
type EmailService interface {
	SendWorkspaceInvitation(ctx context.Context, email, workspaceName, inviterName, token string) error
}

// WorkspaceService handles workspace business logic
type WorkspaceService struct {
	workspaceRepo  WorkspaceRepository
	memberRepo     MemberRepository
	invitationRepo InvitationRepository
	auditRepo      AuditLogRepository
	emailSvc       EmailService
}

// NewWorkspaceService creates a new workspace service
func NewWorkspaceService(
	workspaceRepo WorkspaceRepository,
	memberRepo MemberRepository,
	invitationRepo InvitationRepository,
	auditRepo AuditLogRepository,
	emailSvc EmailService,
) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepo:  workspaceRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		auditRepo:      auditRepo,
		emailSvc:       emailSvc,
	}
}

// CreateWorkspaceInput represents workspace creation input
type CreateWorkspaceInput struct {
	Name        string
	Slug        string
	Description string
	OwnerID     string
	Plan        model.Plan
}

// CreateWorkspace creates a new workspace
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (*model.Workspace, error) {
	// Check if slug exists
	exists, err := s.workspaceRepo.SlugExists(ctx, input.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("slug '%s' is already taken", input.Slug)
	}

	// Create workspace
	workspace, err := model.NewWorkspace(input.Name, input.Slug, input.OwnerID, input.Plan)
	if err != nil {
		return nil, err
	}
	workspace.Description = input.Description

	if err := s.workspaceRepo.Create(ctx, workspace); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Add owner as member
	member := model.NewWorkspaceMember(workspace.ID.String(), input.OwnerID, model.RoleOwner, input.OwnerID)
	if err := s.memberRepo.Add(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	// Audit log
	s.auditRepo.Create(ctx, model.NewAuditLog(
		workspace.ID.String(), input.OwnerID, model.ActionCreate,
		model.ResourceWorkspace, workspace.ID.String(), nil,
	))

	return workspace, nil
}

// GetWorkspace retrieves a workspace by ID
func (s *WorkspaceService) GetWorkspace(ctx context.Context, id string) (*model.Workspace, error) {
	return s.workspaceRepo.FindByID(ctx, id)
}

// GetWorkspaceBySlug retrieves a workspace by slug
func (s *WorkspaceService) GetWorkspaceBySlug(ctx context.Context, slug string) (*model.Workspace, error) {
	return s.workspaceRepo.FindBySlug(ctx, slug)
}

// ListUserWorkspaces lists workspaces for a user
func (s *WorkspaceService) ListUserWorkspaces(ctx context.Context, userID string) ([]*model.Workspace, error) {
	// Get all memberships
	memberships, err := s.memberRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get workspaces
	workspaces := make([]*model.Workspace, 0, len(memberships))
	for _, m := range memberships {
		ws, err := s.workspaceRepo.FindByID(ctx, m.WorkspaceID)
		if err != nil {
			continue
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// UpdateWorkspaceInput represents workspace update input
type UpdateWorkspaceInput struct {
	ID          string
	Name        string
	Description string
	Settings    *model.WorkspaceSettings
	ActorID     string
}

// UpdateWorkspace updates a workspace
func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, input UpdateWorkspaceInput) (*model.Workspace, error) {
	// Check permission
	member, err := s.memberRepo.FindByWorkspaceAndUser(ctx, input.ID, input.ActorID)
	if err != nil {
		return nil, model.ErrMemberNotFound
	}
	if !member.Role.CanManageSettings() {
		return nil, model.ErrInsufficientPermission
	}

	workspace, err := s.workspaceRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		workspace.Name = input.Name
	}
	if input.Description != "" {
		workspace.Description = input.Description
	}
	if input.Settings != nil {
		workspace.Settings = *input.Settings
	}
	workspace.UpdatedAt = time.Now()

	if err := s.workspaceRepo.Update(ctx, workspace); err != nil {
		return nil, err
	}

	s.auditRepo.Create(ctx, model.NewAuditLog(
		input.ID, input.ActorID, model.ActionUpdate,
		model.ResourceWorkspace, input.ID, nil,
	))

	return workspace, nil
}

// DeleteWorkspace deletes a workspace
func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID, actorID string) error {
	workspace, err := s.workspaceRepo.FindByID(ctx, workspaceID)
	if err != nil {
		return err
	}

	if workspace.OwnerID != actorID {
		return model.ErrInsufficientPermission
	}

	return s.workspaceRepo.Delete(ctx, workspaceID)
}

// Member Management

// InviteMemberInput represents invitation input
type InviteMemberInput struct {
	WorkspaceID string
	Email       string
	Role        model.MemberRole
	InviterID   string
	InviterName string
}

// InviteMember invites a user to a workspace
func (s *WorkspaceService) InviteMember(ctx context.Context, input InviteMemberInput) (*model.WorkspaceInvitation, error) {
	// Check permission
	member, err := s.memberRepo.FindByWorkspaceAndUser(ctx, input.WorkspaceID, input.InviterID)
	if err != nil {
		return nil, model.ErrMemberNotFound
	}
	if !member.Role.CanManageMembers() {
		return nil, model.ErrInsufficientPermission
	}

	// Check limits
	workspace, err := s.workspaceRepo.FindByID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, err
	}

	memberCount, err := s.memberRepo.CountByWorkspace(ctx, input.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if workspace.Limits.MaxMembers > 0 && memberCount >= workspace.Limits.MaxMembers {
		return nil, model.ErrMemberLimitReached
	}

	// Check if already invited
	existing, _ := s.invitationRepo.FindByEmail(ctx, input.WorkspaceID, input.Email)
	if existing != nil && existing.IsValid() {
		return existing, nil
	}

	// Create invitation
	invitation, err := model.NewWorkspaceInvitation(input.WorkspaceID, input.Email, input.Role, input.InviterID)
	if err != nil {
		return nil, err
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, err
	}

	// Send email
	if s.emailSvc != nil {
		s.emailSvc.SendWorkspaceInvitation(ctx, input.Email, workspace.Name, input.InviterName, invitation.Token)
	}

	s.auditRepo.Create(ctx, model.NewAuditLog(
		input.WorkspaceID, input.InviterID, model.ActionInvite,
		model.ResourceMember, invitation.ID, map[string]interface{}{
			"email": input.Email,
			"role":  input.Role,
		},
	))

	return invitation, nil
}

// AcceptInvitation accepts a workspace invitation
func (s *WorkspaceService) AcceptInvitation(ctx context.Context, token, userID string) (*model.WorkspaceMember, error) {
	invitation, err := s.invitationRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, model.ErrInvitationNotFound
	}

	if !invitation.IsValid() {
		return nil, model.ErrInvitationExpired
	}

	// Check if already a member
	existing, _ := s.memberRepo.FindByWorkspaceAndUser(ctx, invitation.WorkspaceID, userID)
	if existing != nil {
		return nil, model.ErrAlreadyMember
	}

	// Create member
	member := model.NewWorkspaceMember(invitation.WorkspaceID, userID, invitation.Role, invitation.InvitedBy)
	if err := s.memberRepo.Add(ctx, member); err != nil {
		return nil, err
	}

	// Update invitation
	invitation.Accept()
	s.invitationRepo.Update(ctx, invitation)

	return member, nil
}

// DeclineInvitation declines a workspace invitation
func (s *WorkspaceService) DeclineInvitation(ctx context.Context, token string) error {
	invitation, err := s.invitationRepo.FindByToken(ctx, token)
	if err != nil {
		return model.ErrInvitationNotFound
	}

	invitation.Decline()
	return s.invitationRepo.Update(ctx, invitation)
}

// GetPendingInvitations gets pending invitations for an email
func (s *WorkspaceService) GetPendingInvitations(ctx context.Context, email string) ([]*model.WorkspaceInvitation, error) {
	return s.invitationRepo.ListPendingByEmail(ctx, email)
}

// ListMembers lists workspace members
func (s *WorkspaceService) ListMembers(ctx context.Context, workspaceID string) ([]*model.WorkspaceMember, error) {
	return s.memberRepo.ListByWorkspace(ctx, workspaceID)
}

// UpdateMemberRoleInput represents role update input
type UpdateMemberRoleInput struct {
	WorkspaceID string
	MemberID    string
	NewRole     model.MemberRole
	ActorID     string
}

// UpdateMemberRole updates a member's role
func (s *WorkspaceService) UpdateMemberRole(ctx context.Context, input UpdateMemberRoleInput) error {
	// Check permission
	actor, err := s.memberRepo.FindByWorkspaceAndUser(ctx, input.WorkspaceID, input.ActorID)
	if err != nil {
		return model.ErrMemberNotFound
	}
	if !actor.Role.CanManageMembers() {
		return model.ErrInsufficientPermission
	}

	// Get target member
	member, err := s.memberRepo.FindByID(ctx, input.MemberID)
	if err != nil {
		return model.ErrMemberNotFound
	}

	// Cannot change owner role
	if member.Role == model.RoleOwner {
		return model.ErrCannotRemoveOwner
	}

	if err := s.memberRepo.UpdateRole(ctx, input.MemberID, input.NewRole); err != nil {
		return err
	}

	s.auditRepo.Create(ctx, model.NewAuditLog(
		input.WorkspaceID, input.ActorID, model.ActionChangeRole,
		model.ResourceMember, input.MemberID, map[string]interface{}{
			"newRole": input.NewRole,
		},
	))

	return nil
}

// RemoveMember removes a member from workspace
func (s *WorkspaceService) RemoveMember(ctx context.Context, workspaceID, memberID, actorID string) error {
	// Check permission
	actor, err := s.memberRepo.FindByWorkspaceAndUser(ctx, workspaceID, actorID)
	if err != nil {
		return model.ErrMemberNotFound
	}
	if !actor.Role.CanManageMembers() {
		return model.ErrInsufficientPermission
	}

	// Get target member
	member, err := s.memberRepo.FindByID(ctx, memberID)
	if err != nil {
		return model.ErrMemberNotFound
	}

	// Cannot remove owner
	if member.Role == model.RoleOwner {
		return model.ErrCannotRemoveOwner
	}

	if err := s.memberRepo.Remove(ctx, memberID); err != nil {
		return err
	}

	s.auditRepo.Create(ctx, model.NewAuditLog(
		workspaceID, actorID, model.ActionRemoveMember,
		model.ResourceMember, memberID, nil,
	))

	return nil
}

// LeaveWorkspace allows a member to leave a workspace
func (s *WorkspaceService) LeaveWorkspace(ctx context.Context, workspaceID, userID string) error {
	member, err := s.memberRepo.FindByWorkspaceAndUser(ctx, workspaceID, userID)
	if err != nil {
		return model.ErrMemberNotFound
	}

	if member.Role == model.RoleOwner {
		return model.ErrCannotRemoveOwner
	}

	return s.memberRepo.Remove(ctx, member.ID)
}

// GetMemberRole gets a user's role in a workspace
func (s *WorkspaceService) GetMemberRole(ctx context.Context, workspaceID, userID string) (model.MemberRole, error) {
	member, err := s.memberRepo.FindByWorkspaceAndUser(ctx, workspaceID, userID)
	if err != nil {
		return "", model.ErrMemberNotFound
	}
	return member.Role, nil
}

// Audit Log

// ListAuditLogs lists audit logs for a workspace
func (s *WorkspaceService) ListAuditLogs(ctx context.Context, workspaceID string, offset, limit int) ([]*model.AuditLog, int64, error) {
	return s.auditRepo.ListByWorkspace(ctx, workspaceID, offset, limit)
}

// LogAction logs an action to audit log
func (s *WorkspaceService) LogAction(ctx context.Context, workspaceID, userID, action, resource, resourceID string, details map[string]interface{}) {
	s.auditRepo.Create(ctx, model.NewAuditLog(workspaceID, userID, action, resource, resourceID, details))
}

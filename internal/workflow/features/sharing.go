// Package features provides workflow sharing capabilities
package features

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

var (
	ErrShareNotFound       = errors.New("share not found")
	ErrInvalidPermission   = errors.New("invalid permission")
	ErrShareExpired        = errors.New("share link has expired")
	ErrAccessDenied        = errors.New("access denied")
	ErrAlreadyShared       = errors.New("workflow already shared with this user")
	ErrCannotShareWithSelf = errors.New("cannot share workflow with yourself")
)

// SharePermission defines what actions a user can perform on a shared workflow
type SharePermission string

const (
	PermissionView    SharePermission = "view"
	PermissionExecute SharePermission = "execute"
	PermissionEdit    SharePermission = "edit"
	PermissionAdmin   SharePermission = "admin"
)

// ShareType defines how a workflow is shared
type ShareType string

const (
	ShareTypeUser      ShareType = "user"
	ShareTypeWorkspace ShareType = "workspace"
	ShareTypePublic    ShareType = "public"
	ShareTypeLink      ShareType = "link"
)

// WorkflowShare represents a sharing relationship
type WorkflowShare struct {
	ID           string          `json:"id"`
	WorkflowID   string          `json:"workflowId"`
	OwnerID      string          `json:"ownerId"`
	ShareType    ShareType       `json:"shareType"`
	TargetID     string          `json:"targetId,omitempty"`
	Permission   SharePermission `json:"permission"`
	LinkToken    string          `json:"linkToken,omitempty"`
	ExpiresAt    *time.Time      `json:"expiresAt,omitempty"`
	Password     string          `json:"-"`
	MaxUses      int             `json:"maxUses,omitempty"`
	UseCount     int             `json:"useCount"`
	AllowCopy    bool            `json:"allowCopy"`
	AllowExecute bool            `json:"allowExecute"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	CreatedBy    string          `json:"createdBy"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ShareInvitation represents a pending share invitation
type ShareInvitation struct {
	ID           string          `json:"id"`
	WorkflowID   string          `json:"workflowId"`
	InviterID    string          `json:"inviterId"`
	InviteeEmail string          `json:"inviteeEmail"`
	Permission   SharePermission `json:"permission"`
	Token        string          `json:"token"`
	ExpiresAt    time.Time       `json:"expiresAt"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"createdAt"`
}

// ShareRepository defines persistence for shares
type ShareRepository interface {
	Create(ctx context.Context, share *WorkflowShare) error
	FindByID(ctx context.Context, id string) (*WorkflowShare, error)
	FindByWorkflowID(ctx context.Context, workflowID string) ([]*WorkflowShare, error)
	FindByTargetID(ctx context.Context, targetID string, shareType ShareType) ([]*WorkflowShare, error)
	FindByLinkToken(ctx context.Context, token string) (*WorkflowShare, error)
	Update(ctx context.Context, share *WorkflowShare) error
	Delete(ctx context.Context, id string) error
	DeleteByWorkflowID(ctx context.Context, workflowID string) error
	IncrementUseCount(ctx context.Context, id string) error

	CreateInvitation(ctx context.Context, invitation *ShareInvitation) error
	FindInvitationByToken(ctx context.Context, token string) (*ShareInvitation, error)
	UpdateInvitation(ctx context.Context, invitation *ShareInvitation) error
	DeleteInvitation(ctx context.Context, id string) error
	FindPendingInvitations(ctx context.Context, email string) ([]*ShareInvitation, error)
}

// SharingService manages workflow sharing
type SharingService struct {
	shareRepo    ShareRepository
	workflowRepo interface {
		FindByID(ctx context.Context, id model.WorkflowID) (*model.Workflow, error)
	}
	mu sync.RWMutex
}

// NewSharingService creates a new sharing service
func NewSharingService(shareRepo ShareRepository, workflowRepo interface {
	FindByID(ctx context.Context, id model.WorkflowID) (*model.Workflow, error)
}) *SharingService {
	return &SharingService{
		shareRepo:    shareRepo,
		workflowRepo: workflowRepo,
	}
}

// ShareWithUser shares a workflow with a specific user
func (s *SharingService) ShareWithUser(ctx context.Context, workflowID, ownerID, targetUserID string, permission SharePermission) (*WorkflowShare, error) {
	if ownerID == targetUserID {
		return nil, ErrCannotShareWithSelf
	}

	if !isValidPermission(permission) {
		return nil, ErrInvalidPermission
	}

	existing, _ := s.shareRepo.FindByTargetID(ctx, targetUserID, ShareTypeUser)
	for _, share := range existing {
		if share.WorkflowID == workflowID {
			return nil, ErrAlreadyShared
		}
	}

	share := &WorkflowShare{
		ID:           uuid.New().String(),
		WorkflowID:   workflowID,
		OwnerID:      ownerID,
		ShareType:    ShareTypeUser,
		TargetID:     targetUserID,
		Permission:   permission,
		AllowCopy:    permission == PermissionEdit || permission == PermissionAdmin,
		AllowExecute: permission != PermissionView,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    ownerID,
		Metadata:     make(map[string]interface{}),
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("failed to create share: %w", err)
	}

	return share, nil
}

// ShareWithWorkspace shares a workflow with an entire workspace
func (s *SharingService) ShareWithWorkspace(ctx context.Context, workflowID, ownerID, workspaceID string, permission SharePermission) (*WorkflowShare, error) {
	if !isValidPermission(permission) {
		return nil, ErrInvalidPermission
	}

	share := &WorkflowShare{
		ID:           uuid.New().String(),
		WorkflowID:   workflowID,
		OwnerID:      ownerID,
		ShareType:    ShareTypeWorkspace,
		TargetID:     workspaceID,
		Permission:   permission,
		AllowCopy:    permission == PermissionEdit || permission == PermissionAdmin,
		AllowExecute: permission != PermissionView,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    ownerID,
		Metadata:     make(map[string]interface{}),
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("failed to create share: %w", err)
	}

	return share, nil
}

// CreateShareLink creates a shareable link for a workflow
func (s *SharingService) CreateShareLink(ctx context.Context, workflowID, ownerID string, opts ShareLinkOptions) (*WorkflowShare, error) {
	permission := opts.Permission
	if permission == "" {
		permission = PermissionView
	}

	if !isValidPermission(permission) {
		return nil, ErrInvalidPermission
	}

	linkToken := generateLinkToken()

	var expiresAt *time.Time
	if opts.ExpiresIn > 0 {
		t := time.Now().Add(opts.ExpiresIn)
		expiresAt = &t
	}

	share := &WorkflowShare{
		ID:           uuid.New().String(),
		WorkflowID:   workflowID,
		OwnerID:      ownerID,
		ShareType:    ShareTypeLink,
		Permission:   permission,
		LinkToken:    linkToken,
		ExpiresAt:    expiresAt,
		Password:     opts.Password,
		MaxUses:      opts.MaxUses,
		UseCount:     0,
		AllowCopy:    opts.AllowCopy,
		AllowExecute: opts.AllowExecute,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    ownerID,
		Metadata:     make(map[string]interface{}),
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("failed to create share link: %w", err)
	}

	return share, nil
}

// ShareLinkOptions configures share link creation
type ShareLinkOptions struct {
	Permission   SharePermission
	ExpiresIn    time.Duration
	Password     string
	MaxUses      int
	AllowCopy    bool
	AllowExecute bool
}

// MakePublic makes a workflow publicly accessible
func (s *SharingService) MakePublic(ctx context.Context, workflowID, ownerID string, allowCopy bool) (*WorkflowShare, error) {
	existing, _ := s.shareRepo.FindByWorkflowID(ctx, workflowID)
	for _, share := range existing {
		if share.ShareType == ShareTypePublic {
			share.AllowCopy = allowCopy
			share.UpdatedAt = time.Now()
			if err := s.shareRepo.Update(ctx, share); err != nil {
				return nil, err
			}
			return share, nil
		}
	}

	share := &WorkflowShare{
		ID:           uuid.New().String(),
		WorkflowID:   workflowID,
		OwnerID:      ownerID,
		ShareType:    ShareTypePublic,
		Permission:   PermissionView,
		AllowCopy:    allowCopy,
		AllowExecute: false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    ownerID,
		Metadata:     make(map[string]interface{}),
	}

	if err := s.shareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("failed to make public: %w", err)
	}

	return share, nil
}

// MakePrivate removes public access from a workflow
func (s *SharingService) MakePrivate(ctx context.Context, workflowID, ownerID string) error {
	existing, err := s.shareRepo.FindByWorkflowID(ctx, workflowID)
	if err != nil {
		return err
	}

	for _, share := range existing {
		if share.ShareType == ShareTypePublic && share.OwnerID == ownerID {
			return s.shareRepo.Delete(ctx, share.ID)
		}
	}

	return nil
}

// AccessShareLink validates and accesses a share link
func (s *SharingService) AccessShareLink(ctx context.Context, token, password string) (*WorkflowShare, error) {
	share, err := s.shareRepo.FindByLinkToken(ctx, token)
	if err != nil {
		return nil, ErrShareNotFound
	}

	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, ErrShareExpired
	}

	if share.MaxUses > 0 && share.UseCount >= share.MaxUses {
		return nil, ErrShareExpired
	}

	if share.Password != "" && share.Password != password {
		return nil, ErrAccessDenied
	}

	if err := s.shareRepo.IncrementUseCount(ctx, share.ID); err != nil {
		return nil, err
	}

	return share, nil
}

// GetSharesForWorkflow returns all shares for a workflow
func (s *SharingService) GetSharesForWorkflow(ctx context.Context, workflowID, requesterID string) ([]*WorkflowShare, error) {
	return s.shareRepo.FindByWorkflowID(ctx, workflowID)
}

// GetSharedWithMe returns workflows shared with a user
func (s *SharingService) GetSharedWithMe(ctx context.Context, userID string) ([]*WorkflowShare, error) {
	return s.shareRepo.FindByTargetID(ctx, userID, ShareTypeUser)
}

// UpdateShare updates share settings
func (s *SharingService) UpdateShare(ctx context.Context, shareID, requesterID string, opts UpdateShareOptions) (*WorkflowShare, error) {
	share, err := s.shareRepo.FindByID(ctx, shareID)
	if err != nil {
		return nil, ErrShareNotFound
	}

	if share.OwnerID != requesterID {
		return nil, ErrAccessDenied
	}

	if opts.Permission != nil && isValidPermission(*opts.Permission) {
		share.Permission = *opts.Permission
	}
	if opts.AllowCopy != nil {
		share.AllowCopy = *opts.AllowCopy
	}
	if opts.AllowExecute != nil {
		share.AllowExecute = *opts.AllowExecute
	}
	if opts.ExpiresAt != nil {
		share.ExpiresAt = opts.ExpiresAt
	}

	share.UpdatedAt = time.Now()

	if err := s.shareRepo.Update(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// UpdateShareOptions configures share update
type UpdateShareOptions struct {
	Permission   *SharePermission
	AllowCopy    *bool
	AllowExecute *bool
	ExpiresAt    *time.Time
}

// RevokeShare removes a share
func (s *SharingService) RevokeShare(ctx context.Context, shareID, requesterID string) error {
	share, err := s.shareRepo.FindByID(ctx, shareID)
	if err != nil {
		return ErrShareNotFound
	}

	if share.OwnerID != requesterID {
		return ErrAccessDenied
	}

	return s.shareRepo.Delete(ctx, shareID)
}

// RevokeAllShares removes all shares for a workflow
func (s *SharingService) RevokeAllShares(ctx context.Context, workflowID, ownerID string) error {
	return s.shareRepo.DeleteByWorkflowID(ctx, workflowID)
}

// CreateInvitation creates an invitation for a user who doesn't have an account yet
func (s *SharingService) CreateInvitation(ctx context.Context, workflowID, inviterID, email string, permission SharePermission) (*ShareInvitation, error) {
	if !isValidPermission(permission) {
		return nil, ErrInvalidPermission
	}

	invitation := &ShareInvitation{
		ID:           uuid.New().String(),
		WorkflowID:   workflowID,
		InviterID:    inviterID,
		InviteeEmail: email,
		Permission:   permission,
		Token:        uuid.New().String(),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	if err := s.shareRepo.CreateInvitation(ctx, invitation); err != nil {
		return nil, err
	}

	return invitation, nil
}

// AcceptInvitation accepts a share invitation
func (s *SharingService) AcceptInvitation(ctx context.Context, token, userID string) (*WorkflowShare, error) {
	invitation, err := s.shareRepo.FindInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrShareNotFound
	}

	if time.Now().After(invitation.ExpiresAt) {
		return nil, ErrShareExpired
	}

	if invitation.Status != "pending" {
		return nil, errors.New("invitation already processed")
	}

	share, err := s.ShareWithUser(ctx, invitation.WorkflowID, invitation.InviterID, userID, invitation.Permission)
	if err != nil {
		return nil, err
	}

	invitation.Status = "accepted"
	if err := s.shareRepo.UpdateInvitation(ctx, invitation); err != nil {
		return nil, err
	}

	return share, nil
}

// CheckAccess checks if a user has access to a workflow
func (s *SharingService) CheckAccess(ctx context.Context, workflowID, userID string, requiredPermission SharePermission) (bool, error) {
	shares, err := s.shareRepo.FindByWorkflowID(ctx, workflowID)
	if err != nil {
		return false, err
	}

	for _, share := range shares {
		if share.ShareType == ShareTypePublic && requiredPermission == PermissionView {
			return true, nil
		}

		if share.ShareType == ShareTypeUser && share.TargetID == userID {
			if hasPermission(share.Permission, requiredPermission) {
				return true, nil
			}
		}
	}

	return false, nil
}

// Helper functions

func isValidPermission(p SharePermission) bool {
	switch p {
	case PermissionView, PermissionExecute, PermissionEdit, PermissionAdmin:
		return true
	}
	return false
}

func hasPermission(granted, required SharePermission) bool {
	permissions := map[SharePermission]int{
		PermissionView:    1,
		PermissionExecute: 2,
		PermissionEdit:    3,
		PermissionAdmin:   4,
	}

	return permissions[granted] >= permissions[required]
}

func generateLinkToken() string {
	return uuid.New().String()[:8] + uuid.New().String()[:8]
}

// InMemoryShareRepository implements ShareRepository in memory
type InMemoryShareRepository struct {
	shares      map[string]*WorkflowShare
	invitations map[string]*ShareInvitation
	mu          sync.RWMutex
}

// NewInMemoryShareRepository creates a new in-memory share repository
func NewInMemoryShareRepository() *InMemoryShareRepository {
	return &InMemoryShareRepository{
		shares:      make(map[string]*WorkflowShare),
		invitations: make(map[string]*ShareInvitation),
	}
}

func (r *InMemoryShareRepository) Create(ctx context.Context, share *WorkflowShare) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shares[share.ID] = share
	return nil
}

func (r *InMemoryShareRepository) FindByID(ctx context.Context, id string) (*WorkflowShare, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	share, ok := r.shares[id]
	if !ok {
		return nil, ErrShareNotFound
	}
	return share, nil
}

func (r *InMemoryShareRepository) FindByWorkflowID(ctx context.Context, workflowID string) ([]*WorkflowShare, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*WorkflowShare
	for _, share := range r.shares {
		if share.WorkflowID == workflowID {
			result = append(result, share)
		}
	}
	return result, nil
}

func (r *InMemoryShareRepository) FindByTargetID(ctx context.Context, targetID string, shareType ShareType) ([]*WorkflowShare, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*WorkflowShare
	for _, share := range r.shares {
		if share.TargetID == targetID && share.ShareType == shareType {
			result = append(result, share)
		}
	}
	return result, nil
}

func (r *InMemoryShareRepository) FindByLinkToken(ctx context.Context, token string) (*WorkflowShare, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, share := range r.shares {
		if share.LinkToken == token {
			return share, nil
		}
	}
	return nil, ErrShareNotFound
}

func (r *InMemoryShareRepository) Update(ctx context.Context, share *WorkflowShare) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shares[share.ID] = share
	return nil
}

func (r *InMemoryShareRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.shares, id)
	return nil
}

func (r *InMemoryShareRepository) DeleteByWorkflowID(ctx context.Context, workflowID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, share := range r.shares {
		if share.WorkflowID == workflowID {
			delete(r.shares, id)
		}
	}
	return nil
}

func (r *InMemoryShareRepository) IncrementUseCount(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if share, ok := r.shares[id]; ok {
		share.UseCount++
	}
	return nil
}

func (r *InMemoryShareRepository) CreateInvitation(ctx context.Context, invitation *ShareInvitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invitations[invitation.ID] = invitation
	return nil
}

func (r *InMemoryShareRepository) FindInvitationByToken(ctx context.Context, token string) (*ShareInvitation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inv := range r.invitations {
		if inv.Token == token {
			return inv, nil
		}
	}
	return nil, ErrShareNotFound
}

func (r *InMemoryShareRepository) UpdateInvitation(ctx context.Context, invitation *ShareInvitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invitations[invitation.ID] = invitation
	return nil
}

func (r *InMemoryShareRepository) DeleteInvitation(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.invitations, id)
	return nil
}

func (r *InMemoryShareRepository) FindPendingInvitations(ctx context.Context, email string) ([]*ShareInvitation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*ShareInvitation
	for _, inv := range r.invitations {
		if inv.InviteeEmail == email && inv.Status == "pending" {
			result = append(result, inv)
		}
	}
	return result, nil
}

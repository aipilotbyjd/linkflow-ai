// Package postgres provides PostgreSQL repository implementations for workspace
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/workspace/domain/model"
)

// WorkspaceRepository implements workspace persistence
type WorkspaceRepository struct {
	db *sql.DB
}

// NewWorkspaceRepository creates a new workspace repository
func NewWorkspaceRepository(db *sql.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// Create creates a new workspace
func (r *WorkspaceRepository) Create(ctx context.Context, workspace *model.Workspace) error {
	settings, _ := json.Marshal(workspace.Settings)
	limits, _ := json.Marshal(workspace.Limits)
	
	query := `
		INSERT INTO workspaces (id, name, slug, description, owner_id, plan, settings, limits, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		workspace.ID,
		workspace.Name,
		workspace.Slug,
		workspace.Description,
		workspace.OwnerID,
		workspace.Plan,
		settings,
		limits,
		workspace.CreatedAt,
		workspace.UpdatedAt,
	)
	
	return err
}

// FindByID finds a workspace by ID
func (r *WorkspaceRepository) FindByID(ctx context.Context, id string) (*model.Workspace, error) {
	query := `
		SELECT id, name, slug, description, owner_id, plan, settings, limits, created_at, updated_at
		FROM workspaces
		WHERE id = $1 AND deleted_at IS NULL
	`
	
	var ws model.Workspace
	var settings, limits []byte
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ws.ID,
		&ws.Name,
		&ws.Slug,
		&ws.Description,
		&ws.OwnerID,
		&ws.Plan,
		&settings,
		&limits,
		&ws.CreatedAt,
		&ws.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrWorkspaceNotFound
	}
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(settings, &ws.Settings)
	json.Unmarshal(limits, &ws.Limits)
	
	return &ws, nil
}

// FindBySlug finds a workspace by slug
func (r *WorkspaceRepository) FindBySlug(ctx context.Context, slug string) (*model.Workspace, error) {
	query := `
		SELECT id, name, slug, description, owner_id, plan, settings, limits, created_at, updated_at
		FROM workspaces
		WHERE slug = $1 AND deleted_at IS NULL
	`
	
	var ws model.Workspace
	var settings, limits []byte
	
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&ws.ID,
		&ws.Name,
		&ws.Slug,
		&ws.Description,
		&ws.OwnerID,
		&ws.Plan,
		&settings,
		&limits,
		&ws.CreatedAt,
		&ws.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrWorkspaceNotFound
	}
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(settings, &ws.Settings)
	json.Unmarshal(limits, &ws.Limits)
	
	return &ws, nil
}

// Update updates a workspace
func (r *WorkspaceRepository) Update(ctx context.Context, workspace *model.Workspace) error {
	settings, _ := json.Marshal(workspace.Settings)
	limits, _ := json.Marshal(workspace.Limits)
	
	query := `
		UPDATE workspaces
		SET name = $1, description = $2, plan = $3, settings = $4, limits = $5, updated_at = $6
		WHERE id = $7 AND deleted_at IS NULL
	`
	
	_, err := r.db.ExecContext(ctx, query,
		workspace.Name,
		workspace.Description,
		workspace.Plan,
		settings,
		limits,
		time.Now(),
		workspace.ID,
	)
	
	return err
}

// Delete soft deletes a workspace
func (r *WorkspaceRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE workspaces SET deleted_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// ListByUserID lists workspaces for a user
func (r *WorkspaceRepository) ListByUserID(ctx context.Context, userID string) ([]*model.Workspace, error) {
	query := `
		SELECT w.id, w.name, w.slug, w.description, w.owner_id, w.plan, w.settings, w.limits, w.created_at, w.updated_at
		FROM workspaces w
		LEFT JOIN workspace_members m ON w.id = m.workspace_id
		WHERE (w.owner_id = $1 OR m.user_id = $1) AND w.deleted_at IS NULL
		GROUP BY w.id
		ORDER BY w.created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var workspaces []*model.Workspace
	for rows.Next() {
		var ws model.Workspace
		var settings, limits []byte
		
		err := rows.Scan(
			&ws.ID,
			&ws.Name,
			&ws.Slug,
			&ws.Description,
			&ws.OwnerID,
			&ws.Plan,
			&settings,
			&limits,
			&ws.CreatedAt,
			&ws.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal(settings, &ws.Settings)
		json.Unmarshal(limits, &ws.Limits)
		
		workspaces = append(workspaces, &ws)
	}
	
	return workspaces, rows.Err()
}

// MemberRepository implements workspace member persistence
type MemberRepository struct {
	db *sql.DB
}

// NewMemberRepository creates a new member repository
func NewMemberRepository(db *sql.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// Create creates a new member
func (r *MemberRepository) Create(ctx context.Context, member *model.WorkspaceMember) error {
	query := `
		INSERT INTO workspace_members (id, workspace_id, user_id, role, invited_by, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.WorkspaceID,
		member.UserID,
		member.Role,
		member.InvitedBy,
		member.JoinedAt,
	)
	
	return err
}

// FindByID finds a member by ID
func (r *MemberRepository) FindByID(ctx context.Context, id string) (*model.WorkspaceMember, error) {
	query := `
		SELECT id, workspace_id, user_id, role, invited_by, joined_at
		FROM workspace_members
		WHERE id = $1
	`
	
	var m model.WorkspaceMember
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID,
		&m.WorkspaceID,
		&m.UserID,
		&m.Role,
		&m.InvitedBy,
		&m.JoinedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	return &m, err
}

// FindByWorkspaceAndUser finds a member by workspace and user
func (r *MemberRepository) FindByWorkspaceAndUser(ctx context.Context, workspaceID, userID string) (*model.WorkspaceMember, error) {
	query := `
		SELECT id, workspace_id, user_id, role, invited_by, joined_at
		FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`
	
	var m model.WorkspaceMember
	err := r.db.QueryRowContext(ctx, query, workspaceID, userID).Scan(
		&m.ID,
		&m.WorkspaceID,
		&m.UserID,
		&m.Role,
		&m.InvitedBy,
		&m.JoinedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &m, err
}

// ListByWorkspace lists members for a workspace
func (r *MemberRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]*model.WorkspaceMember, error) {
	query := `
		SELECT id, workspace_id, user_id, role, invited_by, joined_at
		FROM workspace_members
		WHERE workspace_id = $1
		ORDER BY joined_at
	`
	
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var members []*model.WorkspaceMember
	for rows.Next() {
		var m model.WorkspaceMember
		err := rows.Scan(
			&m.ID,
			&m.WorkspaceID,
			&m.UserID,
			&m.Role,
			&m.InvitedBy,
			&m.JoinedAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	
	return members, rows.Err()
}

// Update updates a member
func (r *MemberRepository) Update(ctx context.Context, member *model.WorkspaceMember) error {
	query := `
		UPDATE workspace_members
		SET role = $1, updated_at = $2
		WHERE id = $3
	`
	
	_, err := r.db.ExecContext(ctx, query, member.Role, time.Now(), member.ID)
	return err
}

// Delete deletes a member
func (r *MemberRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workspace_members WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// CountByWorkspace counts members in a workspace
func (r *MemberRepository) CountByWorkspace(ctx context.Context, workspaceID string) (int, error) {
	query := `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, workspaceID).Scan(&count)
	return count, err
}

// InvitationRepository implements workspace invitation persistence
type InvitationRepository struct {
	db *sql.DB
}

// NewInvitationRepository creates a new invitation repository
func NewInvitationRepository(db *sql.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create creates a new invitation
func (r *InvitationRepository) Create(ctx context.Context, invitation *model.WorkspaceInvitation) error {
	query := `
		INSERT INTO workspace_invitations (id, workspace_id, email, role, token, invited_by, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		invitation.ID,
		invitation.WorkspaceID,
		invitation.Email,
		invitation.Role,
		invitation.Token,
		invitation.InvitedBy,
		invitation.Status,
		invitation.ExpiresAt,
		invitation.CreatedAt,
	)
	
	return err
}

// FindByToken finds an invitation by token
func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*model.WorkspaceInvitation, error) {
	query := `
		SELECT id, workspace_id, email, role, token, invited_by, status, expires_at, accepted_at, created_at
		FROM workspace_invitations
		WHERE token = $1
	`
	
	var inv model.WorkspaceInvitation
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID,
		&inv.WorkspaceID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&inv.InvitedBy,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, model.ErrInvitationNotFound
	}
	return &inv, err
}

// FindPendingByEmail finds pending invitations by email
func (r *InvitationRepository) FindPendingByEmail(ctx context.Context, email string) ([]*model.WorkspaceInvitation, error) {
	query := `
		SELECT id, workspace_id, email, role, token, invited_by, status, expires_at, created_at
		FROM workspace_invitations
		WHERE email = $1 AND status = 'pending' AND expires_at > $2
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query, email, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var invitations []*model.WorkspaceInvitation
	for rows.Next() {
		var inv model.WorkspaceInvitation
		err := rows.Scan(
			&inv.ID,
			&inv.WorkspaceID,
			&inv.Email,
			&inv.Role,
			&inv.Token,
			&inv.InvitedBy,
			&inv.Status,
			&inv.ExpiresAt,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, &inv)
	}
	
	return invitations, rows.Err()
}

// Update updates an invitation
func (r *InvitationRepository) Update(ctx context.Context, invitation *model.WorkspaceInvitation) error {
	query := `
		UPDATE workspace_invitations
		SET status = $1, accepted_at = $2
		WHERE id = $3
	`
	
	_, err := r.db.ExecContext(ctx, query, invitation.Status, invitation.AcceptedAt, invitation.ID)
	return err
}

// AuditLogRepository implements audit log persistence
type AuditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	details, _ := json.Marshal(log.Details)
	
	query := `
		INSERT INTO audit_logs (id, workspace_id, user_id, action, resource, resource_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		uuid.New().String(),
		log.WorkspaceID,
		log.UserID,
		log.Action,
		log.Resource,
		log.ResourceID,
		details,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)
	
	return err
}

// ListByWorkspace lists audit logs for a workspace
func (r *AuditLogRepository) ListByWorkspace(ctx context.Context, workspaceID string, offset, limit int) ([]*model.AuditLog, int64, error) {
	// Count total
	var total int64
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE workspace_id = $1`
	r.db.QueryRowContext(ctx, countQuery, workspaceID).Scan(&total)
	
	// Get logs
	query := `
		SELECT id, workspace_id, user_id, action, resource, resource_id, details, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var logs []*model.AuditLog
	for rows.Next() {
		var log model.AuditLog
		var details []byte
		
		err := rows.Scan(
			&log.ID,
			&log.WorkspaceID,
			&log.UserID,
			&log.Action,
			&log.Resource,
			&log.ResourceID,
			&details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		
		json.Unmarshal(details, &log.Details)
		logs = append(logs, &log)
	}
	
	return logs, total, rows.Err()
}

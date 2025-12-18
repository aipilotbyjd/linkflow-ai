package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/user/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/user/domain/repository"
)

// OrganizationRepository implements the organization repository interface for PostgreSQL
type OrganizationRepository struct {
	db *database.DB
}

// NewOrganizationRepository creates a new PostgreSQL organization repository
func NewOrganizationRepository(db *database.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Save saves a new organization
func (r *OrganizationRepository) Save(ctx context.Context, org *model.Organization) error {
	// Serialize settings
	settingsJSON, err := json.Marshal(org.Settings())
	if err != nil {
		return fmt.Errorf("failed to serialize settings: %w", err)
	}

	query := `
		INSERT INTO user_service.organizations (
			id, name, slug, description,
			logo_url, website, status,
			created_by, settings,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		org.ID().String(),
		org.Name(),
		org.Slug(),
		org.Description(),
		org.LogoURL(),
		org.Website(),
		string(org.Status()),
		org.CreatedBy(),
		string(settingsJSON),
		org.CreatedAt(),
		org.UpdatedAt(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to insert organization: %w", err)
	}

	// Save members
	for _, member := range org.Members() {
		err := r.saveMember(ctx, org.ID(), member)
		if err != nil {
			return fmt.Errorf("failed to save member: %w", err)
		}
	}

	return nil
}

// FindByID finds an organization by ID
func (r *OrganizationRepository) FindByID(ctx context.Context, id model.OrganizationID) (*model.Organization, error) {
	query := `
		SELECT 
			id, name, slug, description,
			logo_url, website, status,
			created_by, settings,
			created_at, updated_at
		FROM user_service.organizations
		WHERE id = $1
	`

	var (
		orgID       string
		name        string
		slug        string
		description sql.NullString
		logoURL     sql.NullString
		website     sql.NullString
		status      string
		createdBy   string
		settingsJSON []byte
		createdAt   time.Time
		updatedAt   time.Time
	)

	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&orgID,
		&name,
		&slug,
		&description,
		&logoURL,
		&website,
		&status,
		&createdBy,
		&settingsJSON,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to query organization: %w", err)
	}

	// Deserialize settings
	var settings map[string]interface{}
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("failed to deserialize settings: %w", err)
	}

	// Load members
	members, err := r.loadMembers(ctx, model.OrganizationID(orgID))
	if err != nil {
		return nil, fmt.Errorf("failed to load members: %w", err)
	}

	// Reconstruct organization
	org := model.ReconstructOrganization(
		model.OrganizationID(orgID),
		name,
		slug,
		description.String,
		logoURL.String,
		website.String,
		model.OrganizationStatus(status),
		createdBy,
		members,
		settings,
		createdAt,
		updatedAt,
		0, // version
	)

	return org, nil
}

// FindBySlug finds an organization by slug
func (r *OrganizationRepository) FindBySlug(ctx context.Context, slug string) (*model.Organization, error) {
	query := `
		SELECT id FROM user_service.organizations WHERE slug = $1
	`

	var orgID string
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to query organization by slug: %w", err)
	}

	return r.FindByID(ctx, model.OrganizationID(orgID))
}

// FindByUserID finds organizations where user is a member
func (r *OrganizationRepository) FindByUserID(ctx context.Context, userID string) ([]*model.Organization, error) {
	query := `
		SELECT DISTINCT o.id
		FROM user_service.organizations o
		JOIN user_service.organization_members m ON o.id = m.organization_id
		WHERE m.user_id = $1
		ORDER BY o.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query organizations: %w", err)
	}
	defer rows.Close()

	var organizations []*model.Organization
	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return nil, fmt.Errorf("failed to scan organization ID: %w", err)
		}

		org, err := r.FindByID(ctx, model.OrganizationID(orgID))
		if err != nil {
			continue
		}
		organizations = append(organizations, org)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating organization rows: %w", err)
	}

	return organizations, nil
}

// Update updates an existing organization
func (r *OrganizationRepository) Update(ctx context.Context, org *model.Organization) error {
	settingsJSON, _ := json.Marshal(org.Settings())

	query := `
		UPDATE user_service.organizations
		SET 
			name = $2,
			slug = $3,
			description = $4,
			logo_url = $5,
			website = $6,
			status = $7,
			settings = $8::jsonb,
			updated_at = $9
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		org.ID().String(),
		org.Name(),
		org.Slug(),
		org.Description(),
		org.LogoURL(),
		org.Website(),
		string(org.Status()),
		string(settingsJSON),
		org.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	// Update members
	// First, delete existing members
	_, err = r.db.ExecContext(ctx, 
		"DELETE FROM user_service.organization_members WHERE organization_id = $1",
		org.ID().String(),
	)
	if err != nil {
		return fmt.Errorf("failed to delete existing members: %w", err)
	}

	// Then save new members
	for _, member := range org.Members() {
		err := r.saveMember(ctx, org.ID(), member)
		if err != nil {
			return fmt.Errorf("failed to save member: %w", err)
		}
	}

	return nil
}

// Delete deletes an organization
func (r *OrganizationRepository) Delete(ctx context.Context, id model.OrganizationID) error {
	query := `
		DELETE FROM user_service.organizations WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}

// ExistsBySlug checks if an organization with slug exists
func (r *OrganizationRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_service.organizations WHERE slug = $1
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}

	return exists, nil
}

// Helper methods

func (r *OrganizationRepository) saveMember(ctx context.Context, orgID model.OrganizationID, member model.Member) error {
	query := `
		INSERT INTO user_service.organization_members (
			organization_id, user_id, role, joined_at
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (organization_id, user_id) DO UPDATE
		SET role = EXCLUDED.role
	`

	_, err := r.db.ExecContext(ctx, query,
		orgID.String(),
		member.UserID,
		string(member.Role),
		member.JoinedAt,
	)

	return err
}

func (r *OrganizationRepository) loadMembers(ctx context.Context, orgID model.OrganizationID) ([]model.Member, error) {
	query := `
		SELECT user_id, role, joined_at
		FROM user_service.organization_members
		WHERE organization_id = $1
		ORDER BY joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, orgID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query members: %w", err)
	}
	defer rows.Close()

	var members []model.Member
	for rows.Next() {
		var member model.Member
		var role string
		
		err := rows.Scan(&member.UserID, &role, &member.JoinedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		
		member.Role = model.Role(role)
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating member rows: %w", err)
	}

	return members, nil
}

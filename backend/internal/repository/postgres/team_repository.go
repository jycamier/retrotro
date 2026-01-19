package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// TeamRepository handles team database operations
type TeamRepository struct {
	pool *pgxpool.Pool
}

// NewTeamRepository creates a new team repository
func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

// FindByID finds a team by ID
func (r *TeamRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	query := `
		SELECT id, name, slug, description, oidc_group_id, is_oidc_managed,
		       created_by, created_at, updated_at
		FROM teams WHERE id = $1
	`

	var team models.Team
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&team.ID, &team.Name, &team.Slug, &team.Description,
		&team.OIDCGroupID, &team.IsOIDCManaged, &team.CreatedBy,
		&team.CreatedAt, &team.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &team, nil
}

// FindBySlug finds a team by slug
func (r *TeamRepository) FindBySlug(ctx context.Context, slug string) (*models.Team, error) {
	query := `
		SELECT id, name, slug, description, oidc_group_id, is_oidc_managed,
		       created_by, created_at, updated_at
		FROM teams WHERE slug = $1
	`

	var team models.Team
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&team.ID, &team.Name, &team.Slug, &team.Description,
		&team.OIDCGroupID, &team.IsOIDCManaged, &team.CreatedBy,
		&team.CreatedAt, &team.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &team, nil
}

// FindByOIDCGroupID finds a team by OIDC group ID
func (r *TeamRepository) FindByOIDCGroupID(ctx context.Context, groupID string) (*models.Team, error) {
	query := `
		SELECT id, name, slug, description, oidc_group_id, is_oidc_managed,
		       created_by, created_at, updated_at
		FROM teams WHERE oidc_group_id = $1
	`

	var team models.Team
	err := r.pool.QueryRow(ctx, query, groupID).Scan(
		&team.ID, &team.Name, &team.Slug, &team.Description,
		&team.OIDCGroupID, &team.IsOIDCManaged, &team.CreatedBy,
		&team.CreatedAt, &team.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &team, nil
}

// ListAll returns all teams
func (r *TeamRepository) ListAll(ctx context.Context) ([]*models.Team, error) {
	query := `
		SELECT id, name, slug, description, oidc_group_id, is_oidc_managed,
		       created_by, created_at, updated_at
		FROM teams
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		var team models.Team
		err := rows.Scan(
			&team.ID, &team.Name, &team.Slug, &team.Description,
			&team.OIDCGroupID, &team.IsOIDCManaged, &team.CreatedBy,
			&team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		teams = append(teams, &team)
	}

	if teams == nil {
		teams = []*models.Team{}
	}

	return teams, nil
}

// List returns all teams for a user
func (r *TeamRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.Team, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.description, t.oidc_group_id, t.is_oidc_managed,
		       t.created_by, t.created_at, t.updated_at
		FROM teams t
		INNER JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = $1
		ORDER BY t.name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		var team models.Team
		err := rows.Scan(
			&team.ID, &team.Name, &team.Slug, &team.Description,
			&team.OIDCGroupID, &team.IsOIDCManaged, &team.CreatedBy,
			&team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		teams = append(teams, &team)
	}

	return teams, nil
}

// Create creates a new team
func (r *TeamRepository) Create(ctx context.Context, team *models.Team) (*models.Team, error) {
	query := `
		INSERT INTO teams (id, name, slug, description, oidc_group_id, is_oidc_managed, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	if team.ID == uuid.Nil {
		team.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		team.ID, team.Name, team.Slug, team.Description,
		team.OIDCGroupID, team.IsOIDCManaged, team.CreatedBy,
	).Scan(&team.ID, &team.CreatedAt, &team.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return team, nil
}

// Update updates a team
func (r *TeamRepository) Update(ctx context.Context, team *models.Team) error {
	query := `
		UPDATE teams
		SET name = $2, slug = $3, description = $4, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, team.ID, team.Name, team.Slug, team.Description)
	return err
}

// Delete deletes a team
func (r *TeamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM teams WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// TeamMemberRepository handles team member database operations
type TeamMemberRepository struct {
	pool *pgxpool.Pool
}

// NewTeamMemberRepository creates a new team member repository
func NewTeamMemberRepository(pool *pgxpool.Pool) *TeamMemberRepository {
	return &TeamMemberRepository{pool: pool}
}

// Find finds a team member
func (r *TeamMemberRepository) Find(ctx context.Context, teamID, userID uuid.UUID) (*models.TeamMember, error) {
	query := `
		SELECT id, team_id, user_id, role, is_oidc_synced, last_synced_at, joined_at
		FROM team_members WHERE team_id = $1 AND user_id = $2
	`

	var member models.TeamMember
	err := r.pool.QueryRow(ctx, query, teamID, userID).Scan(
		&member.ID, &member.TeamID, &member.UserID, &member.Role,
		&member.IsOIDCSynced, &member.LastSyncedAt, &member.JoinedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &member, nil
}

// GetByTeamAndUser is an alias for Find - gets a team member by team and user ID
func (r *TeamMemberRepository) GetByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) (*models.TeamMember, error) {
	return r.Find(ctx, teamID, userID)
}

// ListByTeam lists all members of a team
func (r *TeamMemberRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.TeamMember, error) {
	query := `
		SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.is_oidc_synced, tm.last_synced_at, tm.joined_at,
		       u.id, u.email, u.display_name, u.avatar_url, u.is_admin
		FROM team_members tm
		INNER JOIN users u ON tm.user_id = u.id
		WHERE tm.team_id = $1
		ORDER BY u.display_name
	`

	rows, err := r.pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		var user models.User
		err := rows.Scan(
			&member.ID, &member.TeamID, &member.UserID, &member.Role,
			&member.IsOIDCSynced, &member.LastSyncedAt, &member.JoinedAt,
			&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.IsAdmin,
		)
		if err != nil {
			return nil, err
		}
		member.User = &user
		members = append(members, &member)
	}

	return members, nil
}

// Create creates a new team member
func (r *TeamMemberRepository) Create(ctx context.Context, member *models.TeamMember) (*models.TeamMember, error) {
	query := `
		INSERT INTO team_members (id, team_id, user_id, role, is_oidc_synced, last_synced_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, joined_at
	`

	if member.ID == uuid.Nil {
		member.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		member.ID, member.TeamID, member.UserID, member.Role,
		member.IsOIDCSynced, member.LastSyncedAt,
	).Scan(&member.ID, &member.JoinedAt)

	if err != nil {
		return nil, err
	}

	return member, nil
}

// Update updates a team member
func (r *TeamMemberRepository) Update(ctx context.Context, member *models.TeamMember) error {
	query := `
		UPDATE team_members
		SET role = $3, is_oidc_synced = $4, last_synced_at = $5
		WHERE team_id = $1 AND user_id = $2
	`

	_, err := r.pool.Exec(ctx, query,
		member.TeamID, member.UserID, member.Role,
		member.IsOIDCSynced, member.LastSyncedAt,
	)
	return err
}

// Delete deletes a team member
func (r *TeamMemberRepository) Delete(ctx context.Context, teamID, userID uuid.UUID) error {
	query := `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, teamID, userID)
	return err
}

// DeleteOIDCSyncedExcept removes OIDC-synced memberships except for specified teams
func (r *TeamMemberRepository) DeleteOIDCSyncedExcept(ctx context.Context, userID uuid.UUID, keepTeamIDs []uuid.UUID) error {
	if len(keepTeamIDs) == 0 {
		query := `DELETE FROM team_members WHERE user_id = $1 AND is_oidc_synced = true`
		_, err := r.pool.Exec(ctx, query, userID)
		return err
	}

	query := `
		DELETE FROM team_members
		WHERE user_id = $1 AND is_oidc_synced = true AND team_id != ALL($2)
	`
	_, err := r.pool.Exec(ctx, query, userID, keepTeamIDs)
	return err
}

// GetUserRole gets a user's role in a team
func (r *TeamMemberRepository) GetUserRole(ctx context.Context, teamID, userID uuid.UUID) (models.Role, error) {
	query := `SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2`

	var role models.Role
	err := r.pool.QueryRow(ctx, query, teamID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	return role, nil
}

// UpdateRole updates a member's role
func (r *TeamMemberRepository) UpdateRole(ctx context.Context, teamID, userID uuid.UUID, role models.Role) error {
	query := `UPDATE team_members SET role = $3 WHERE team_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, teamID, userID, role)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// IsMember checks if a user is a member of a team
func (r *TeamMemberRepository) IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, teamID, userID).Scan(&exists)
	return exists, err
}

// CountMembers counts the number of members in a team
func (r *TeamMemberRepository) CountMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM team_members WHERE team_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, teamID).Scan(&count)
	return count, err
}

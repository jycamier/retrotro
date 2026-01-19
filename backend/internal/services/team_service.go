package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

var (
	ErrTeamNotFound     = errors.New("team not found")
	ErrNotTeamMember    = errors.New("not a team member")
	ErrNotAuthorized    = errors.New("not authorized")
	ErrCannotLeaveTeam  = errors.New("cannot leave team as last admin")
)

// TeamService handles team operations
type TeamService struct {
	teamRepo       *postgres.TeamRepository
	memberRepo     *postgres.TeamMemberRepository
	userRepo       UserRepository
}

// NewTeamService creates a new team service
func NewTeamService(teamRepo *postgres.TeamRepository, memberRepo *postgres.TeamMemberRepository, userRepo UserRepository) *TeamService {
	return &TeamService{
		teamRepo:   teamRepo,
		memberRepo: memberRepo,
		userRepo:   userRepo,
	}
}

// CreateTeamInput represents input for creating a team
type CreateTeamInput struct {
	Name        string
	Slug        string
	Description *string
}

// Create creates a new team
func (s *TeamService) Create(ctx context.Context, userID uuid.UUID, input CreateTeamInput) (*models.Team, error) {
	team := &models.Team{
		ID:          uuid.New(),
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		CreatedBy:   &userID,
	}

	team, err := s.teamRepo.Create(ctx, team)
	if err != nil {
		return nil, err
	}

	// Add creator as admin
	member := &models.TeamMember{
		ID:     uuid.New(),
		TeamID: team.ID,
		UserID: userID,
		Role:   models.RoleAdmin,
	}
	_, err = s.memberRepo.Create(ctx, member)
	if err != nil {
		return nil, err
	}

	return team, nil
}

// GetByID gets a team by ID
func (s *TeamService) GetByID(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	team, err := s.teamRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return team, nil
}

// GetBySlug gets a team by slug
func (s *TeamService) GetBySlug(ctx context.Context, slug string) (*models.Team, error) {
	team, err := s.teamRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	return team, nil
}

// ListByUser lists all teams for a user
func (s *TeamService) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.Team, error) {
	return s.teamRepo.ListByUser(ctx, userID)
}

// UpdateTeamInput represents input for updating a team
type UpdateTeamInput struct {
	Name        *string
	Description *string
}

// Update updates a team
func (s *TeamService) Update(ctx context.Context, userID, teamID uuid.UUID, input UpdateTeamInput) (*models.Team, error) {
	// Check authorization
	if err := s.requireRole(ctx, teamID, userID, models.RoleAdmin); err != nil {
		return nil, err
	}

	team, err := s.teamRepo.FindByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		team.Name = *input.Name
	}
	if input.Description != nil {
		team.Description = input.Description
	}

	if err := s.teamRepo.Update(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

// Delete deletes a team
func (s *TeamService) Delete(ctx context.Context, userID, teamID uuid.UUID) error {
	// Check authorization
	if err := s.requireRole(ctx, teamID, userID, models.RoleAdmin); err != nil {
		return err
	}

	return s.teamRepo.Delete(ctx, teamID)
}

// ListMembers lists all members of a team
func (s *TeamService) ListMembers(ctx context.Context, userID, teamID uuid.UUID) ([]*models.TeamMember, error) {
	// Check if user is a member
	isMember, err := s.memberRepo.IsMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	return s.memberRepo.ListByTeam(ctx, teamID)
}

// AddMember adds a member to a team
func (s *TeamService) AddMember(ctx context.Context, userID, teamID uuid.UUID, memberUserID uuid.UUID, role models.Role) error {
	// Check authorization
	if err := s.requireRole(ctx, teamID, userID, models.RoleAdmin); err != nil {
		return err
	}

	// Check if already a member
	isMember, _ := s.memberRepo.IsMember(ctx, teamID, memberUserID)
	if isMember {
		return nil // Already a member, no-op
	}

	member := &models.TeamMember{
		ID:     uuid.New(),
		TeamID: teamID,
		UserID: memberUserID,
		Role:   role,
	}

	_, err := s.memberRepo.Create(ctx, member)
	return err
}

// RemoveMember removes a member from a team
func (s *TeamService) RemoveMember(ctx context.Context, userID, teamID, memberUserID uuid.UUID) error {
	// Check authorization (admin or self)
	if userID != memberUserID {
		if err := s.requireRole(ctx, teamID, userID, models.RoleAdmin); err != nil {
			return err
		}
	}

	// Don't allow removing the last admin
	role, err := s.memberRepo.GetUserRole(ctx, teamID, memberUserID)
	if err != nil {
		return err
	}

	if role == models.RoleAdmin {
		// Count admins
		members, err := s.memberRepo.ListByTeam(ctx, teamID)
		if err != nil {
			return err
		}
		adminCount := 0
		for _, m := range members {
			if m.Role == models.RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return ErrCannotLeaveTeam
		}
	}

	return s.memberRepo.Delete(ctx, teamID, memberUserID)
}

// UpdateMemberRole updates a member's role
func (s *TeamService) UpdateMemberRole(ctx context.Context, userID, teamID, memberUserID uuid.UUID, role models.Role) error {
	// Check authorization
	if err := s.requireRole(ctx, teamID, userID, models.RoleAdmin); err != nil {
		return err
	}

	// Don't allow demoting the last admin
	currentRole, err := s.memberRepo.GetUserRole(ctx, teamID, memberUserID)
	if err != nil {
		return err
	}

	if currentRole == models.RoleAdmin && role != models.RoleAdmin {
		members, err := s.memberRepo.ListByTeam(ctx, teamID)
		if err != nil {
			return err
		}
		adminCount := 0
		for _, m := range members {
			if m.Role == models.RoleAdmin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return ErrCannotLeaveTeam
		}
	}

	return s.memberRepo.UpdateRole(ctx, teamID, memberUserID, role)
}

// GetUserRole gets a user's role in a team
func (s *TeamService) GetUserRole(ctx context.Context, teamID, userID uuid.UUID) (models.Role, error) {
	return s.memberRepo.GetUserRole(ctx, teamID, userID)
}

// IsMember checks if a user is a member of a team
func (s *TeamService) IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error) {
	return s.memberRepo.IsMember(ctx, teamID, userID)
}

// requireRole checks if a user has the required role or higher
func (s *TeamService) requireRole(ctx context.Context, teamID, userID uuid.UUID, requiredRole models.Role) error {
	role, err := s.memberRepo.GetUserRole(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrNotTeamMember
		}
		return err
	}

	// Admin can do everything
	if role == models.RoleAdmin {
		return nil
	}

	// Member can only do member things
	if role == models.RoleMember && requiredRole == models.RoleMember {
		return nil
	}

	return ErrNotAuthorized
}

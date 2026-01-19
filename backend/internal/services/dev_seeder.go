package services

import (
	"context"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

// DevUserInfo contains user info with team membership
type DevUserInfo struct {
	ID          uuid.UUID   `json:"id"`
	Email       string      `json:"email"`
	DisplayName string      `json:"displayName"`
	IsAdmin     bool        `json:"isAdmin"`
	TeamRole    models.Role `json:"teamRole"`
}

// DevTeamInfo contains team info
type DevTeamInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// DevUsersResponse is the response for GetDevUsers
type DevUsersResponse struct {
	Users []DevUserInfo `json:"users"`
	Team  DevTeamInfo   `json:"team"`
}

// DevSeeder provides dev users information (data is seeded via SQL migration)
type DevSeeder struct {
	teamRepo       *postgres.TeamRepository
	teamMemberRepo *postgres.TeamMemberRepository
}

// NewDevSeeder creates a new DevSeeder
func NewDevSeeder(
	teamRepo *postgres.TeamRepository,
	teamMemberRepo *postgres.TeamMemberRepository,
) *DevSeeder {
	return &DevSeeder{
		teamRepo:       teamRepo,
		teamMemberRepo: teamMemberRepo,
	}
}

// GetDevUsersInfo returns the list of dev users with their IDs
// Dev users are seeded via SQL migration (000004_dev_users_seed.up.sql)
func (s *DevSeeder) GetDevUsersInfo(ctx context.Context) (*DevUsersResponse, error) {
	// Get Dev Team
	team, err := s.teamRepo.FindBySlug(ctx, "dev-team")
	if err != nil {
		return nil, err
	}

	// Get team members with their roles
	members, err := s.teamMemberRepo.ListByTeam(ctx, team.ID)
	if err != nil {
		return nil, err
	}

	// Build user info list (only @retrotro.dev emails)
	var users []DevUserInfo
	for _, member := range members {
		if member.User != nil {
			users = append(users, DevUserInfo{
				ID:          member.User.ID,
				Email:       member.User.Email,
				DisplayName: member.User.DisplayName,
				IsAdmin:     member.User.IsAdmin,
				TeamRole:    member.Role,
			})
		}
	}

	return &DevUsersResponse{
		Users: users,
		Team: DevTeamInfo{
			ID:   team.ID,
			Name: team.Name,
			Slug: team.Slug,
		},
	}, nil
}

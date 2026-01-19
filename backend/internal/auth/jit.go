package auth

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/models"
)

// TeamRepository interface for JIT provisioning
type TeamRepository interface {
	FindByOIDCGroupID(ctx context.Context, groupID string) (*models.Team, error)
	Create(ctx context.Context, team *models.Team) (*models.Team, error)
}

// TeamMemberRepository interface for JIT provisioning
type TeamMemberRepository interface {
	Find(ctx context.Context, teamID, userID uuid.UUID) (*models.TeamMember, error)
	Create(ctx context.Context, member *models.TeamMember) (*models.TeamMember, error)
	Update(ctx context.Context, member *models.TeamMember) error
	DeleteOIDCSyncedExcept(ctx context.Context, userID uuid.UUID, keepTeamIDs []uuid.UUID) error
}

// JITProvisioner handles Just-In-Time provisioning of teams from OIDC groups
type JITProvisioner struct {
	config     config.JITConfig
	teamRepo   TeamRepository
	memberRepo TeamMemberRepository
}

// NewJITProvisioner creates a new JIT provisioner
func NewJITProvisioner(cfg config.JITConfig, teamRepo TeamRepository, memberRepo TeamMemberRepository) *JITProvisioner {
	return &JITProvisioner{
		config:     cfg,
		teamRepo:   teamRepo,
		memberRepo: memberRepo,
	}
}

// ProvisionUser provisions teams and memberships for a user based on OIDC claims
func (p *JITProvisioner) ProvisionUser(ctx context.Context, user *models.User, claims map[string]interface{}) error {
	if !p.config.Enabled {
		return nil
	}

	// Extract groups from claims
	groups := p.extractGroups(claims)
	if len(groups) == 0 {
		return nil
	}

	// Synchronize each group
	syncedTeamIDs := make([]uuid.UUID, 0)
	for _, groupID := range groups {
		team, err := p.ensureTeamExists(ctx, groupID)
		if err != nil {
			return err
		}

		role := p.determineRole(groupID)
		if err := p.ensureMembership(ctx, team.ID, user.ID, role); err != nil {
			return err
		}

		syncedTeamIDs = append(syncedTeamIDs, team.ID)
	}

	// Optional: remove from OIDC teams not listed
	if p.config.RemoveStaleMembers {
		if err := p.memberRepo.DeleteOIDCSyncedExcept(ctx, user.ID, syncedTeamIDs); err != nil {
			return err
		}
	}

	return nil
}

// extractGroups extracts group IDs from OIDC claims
func (p *JITProvisioner) extractGroups(claims map[string]interface{}) []string {
	raw, ok := claims[p.config.GroupsClaim]
	if !ok {
		return nil
	}

	var groups []string
	switch v := raw.(type) {
	case []interface{}:
		for _, g := range v {
			if s, ok := g.(string); ok {
				// Remove prefix if configured
				s = strings.TrimPrefix(s, p.config.GroupsPrefix)
				if s != "" {
					groups = append(groups, s)
				}
			}
		}
	case []string:
		for _, s := range v {
			s = strings.TrimPrefix(s, p.config.GroupsPrefix)
			if s != "" {
				groups = append(groups, s)
			}
		}
	case string:
		// Some providers send groups as a comma-separated string
		for _, s := range strings.Split(v, ",") {
			s = strings.TrimSpace(strings.TrimPrefix(s, p.config.GroupsPrefix))
			if s != "" {
				groups = append(groups, s)
			}
		}
	}

	return groups
}

// determineRole determines the role for a user based on group membership
func (p *JITProvisioner) determineRole(groupID string) models.Role {
	for _, adminGroup := range p.config.AdminGroups {
		if groupID == adminGroup {
			return models.RoleAdmin
		}
	}
	// FacilitatorGroups now map to member role (facilitator role removed)
	return models.Role(p.config.DefaultRole)
}

// ensureTeamExists ensures a team exists for the given OIDC group
func (p *JITProvisioner) ensureTeamExists(ctx context.Context, groupID string) (*models.Team, error) {
	// Look up by oidc_group_id
	team, err := p.teamRepo.FindByOIDCGroupID(ctx, groupID)
	if err == nil && team != nil {
		return team, nil
	}

	// Create the team
	team = &models.Team{
		ID:            uuid.New(),
		Name:          groupID, // Initial name = group ID
		Slug:          slugify(groupID),
		OIDCGroupID:   &groupID,
		IsOIDCManaged: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return p.teamRepo.Create(ctx, team)
}

// ensureMembership ensures a user is a member of a team with the correct role
func (p *JITProvisioner) ensureMembership(ctx context.Context, teamID, userID uuid.UUID, role models.Role) error {
	existing, _ := p.memberRepo.Find(ctx, teamID, userID)

	if existing != nil {
		// Update if OIDC managed
		if existing.IsOIDCSynced {
			existing.Role = role
			now := time.Now()
			existing.LastSyncedAt = &now
			return p.memberRepo.Update(ctx, existing)
		}
		return nil // Don't touch manual members
	}

	// Create the membership
	now := time.Now()
	member := &models.TeamMember{
		ID:           uuid.New(),
		TeamID:       teamID,
		UserID:       userID,
		Role:         role,
		IsOIDCSynced: true,
		LastSyncedAt: &now,
		JoinedAt:     now,
	}

	_, err := p.memberRepo.Create(ctx, member)
	return err
}

// slugify converts a string to a URL-friendly slug
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace non-alphanumeric characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	// Limit length
	if len(s) > 100 {
		s = s[:100]
	}

	return s
}

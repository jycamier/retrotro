package services

import (
	"context"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

// StatsService handles statistics operations
type StatsService struct {
	statsRepo  *postgres.StatsRepository
	memberRepo *postgres.TeamMemberRepository
}

// NewStatsService creates a new stats service
func NewStatsService(statsRepo *postgres.StatsRepository, memberRepo *postgres.TeamMemberRepository) *StatsService {
	return &StatsService{
		statsRepo:  statsRepo,
		memberRepo: memberRepo,
	}
}

// GetTeamRotiStats retrieves ROTI statistics for a team
func (s *StatsService) GetTeamRotiStats(ctx context.Context, userID, teamID uuid.UUID, filter *models.StatsFilter) (*models.TeamRotiStats, error) {
	// Check if user is a member of the team
	isMember, err := s.memberRepo.IsMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	return s.statsRepo.GetTeamRotiStats(ctx, teamID, filter)
}

// GetTeamMoodStats retrieves mood statistics for a team
func (s *StatsService) GetTeamMoodStats(ctx context.Context, userID, teamID uuid.UUID, filter *models.StatsFilter) (*models.TeamMoodStats, error) {
	// Check if user is a member of the team
	isMember, err := s.memberRepo.IsMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	return s.statsRepo.GetTeamMoodStats(ctx, teamID, filter)
}

// GetUserRotiStats retrieves ROTI statistics for a specific user within a team
func (s *StatsService) GetUserRotiStats(ctx context.Context, requestingUserID, teamID, targetUserID uuid.UUID, filter *models.StatsFilter) (*models.UserRotiStats, error) {
	// Check if requesting user is a member of the team
	isMember, err := s.memberRepo.IsMember(ctx, teamID, requestingUserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	// Check if target user is a member of the team
	isMember, err = s.memberRepo.IsMember(ctx, teamID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	return s.statsRepo.GetUserRotiStats(ctx, teamID, targetUserID, filter)
}

// GetUserMoodStats retrieves mood statistics for a specific user within a team
func (s *StatsService) GetUserMoodStats(ctx context.Context, requestingUserID, teamID, targetUserID uuid.UUID, filter *models.StatsFilter) (*models.UserMoodStats, error) {
	// Check if requesting user is a member of the team
	isMember, err := s.memberRepo.IsMember(ctx, teamID, requestingUserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	// Check if target user is a member of the team
	isMember, err = s.memberRepo.IsMember(ctx, teamID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	return s.statsRepo.GetUserMoodStats(ctx, teamID, targetUserID, filter)
}

// GetMyStats retrieves combined statistics for the requesting user
func (s *StatsService) GetMyStats(ctx context.Context, userID, teamID uuid.UUID, filter *models.StatsFilter) (*models.CombinedUserStats, error) {
	// Check if user is a member of the team
	isMember, err := s.memberRepo.IsMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	rotiStats, err := s.statsRepo.GetUserRotiStats(ctx, teamID, userID, filter)
	if err != nil {
		return nil, err
	}

	moodStats, err := s.statsRepo.GetUserMoodStats(ctx, teamID, userID, filter)
	if err != nil {
		return nil, err
	}

	return &models.CombinedUserStats{
		RotiStats: rotiStats,
		MoodStats: moodStats,
	}, nil
}

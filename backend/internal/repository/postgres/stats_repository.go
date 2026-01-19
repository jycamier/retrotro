package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// StatsRepository handles statistics database operations
type StatsRepository struct {
	pool *pgxpool.Pool
}

// NewStatsRepository creates a new statistics repository
func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{pool: pool}
}

// GetTeamRotiStats retrieves aggregated ROTI statistics for a team
func (r *StatsRepository) GetTeamRotiStats(ctx context.Context, teamID uuid.UUID, filter *models.StatsFilter) (*models.TeamRotiStats, error) {
	// Build the base query with optional limit
	limitClause := ""
	if filter != nil && filter.Limit > 0 {
		limitClause = "LIMIT $2"
	}

	// Get completed retrospectives for this team
	retrosQuery := `
		SELECT id, name, ended_at
		FROM retrospectives
		WHERE team_id = $1 AND status = 'completed'
		ORDER BY ended_at DESC
	` + limitClause

	var args []interface{}
	args = append(args, teamID)
	if filter != nil && filter.Limit > 0 {
		args = append(args, filter.Limit)
	}

	rows, err := r.pool.Query(ctx, retrosQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var retroIDs []uuid.UUID
	retroMap := make(map[uuid.UUID]struct {
		name    string
		endedAt interface{}
	})

	for rows.Next() {
		var id uuid.UUID
		var name string
		var endedAt interface{}
		if err := rows.Scan(&id, &name, &endedAt); err != nil {
			return nil, err
		}
		retroIDs = append(retroIDs, id)
		retroMap[id] = struct {
			name    string
			endedAt interface{}
		}{name: name, endedAt: endedAt}
	}

	if len(retroIDs) == 0 {
		return &models.TeamRotiStats{
			Distribution: make(map[int]int),
			Evolution:    []*models.RotiEvolutionPoint{},
		}, nil
	}

	// Get overall statistics
	statsQuery := `
		SELECT COALESCE(AVG(rv.rating), 0), COUNT(*)
		FROM roti_votes rv
		JOIN retrospectives r ON r.id = rv.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
	`

	var avg float64
	var totalVotes int
	err = r.pool.QueryRow(ctx, statsQuery, teamID, retroIDs).Scan(&avg, &totalVotes)
	if err != nil {
		return nil, err
	}

	// Get distribution
	distQuery := `
		SELECT rv.rating, COUNT(*) as count
		FROM roti_votes rv
		JOIN retrospectives r ON r.id = rv.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
		GROUP BY rv.rating
		ORDER BY rv.rating
	`

	distRows, err := r.pool.Query(ctx, distQuery, teamID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer distRows.Close()

	distribution := make(map[int]int)
	for distRows.Next() {
		var rating, cnt int
		if err := distRows.Scan(&rating, &cnt); err != nil {
			return nil, err
		}
		distribution[rating] = cnt
	}

	// Get participation rate (votes / potential votes based on participants)
	participationQuery := `
		SELECT
			COUNT(DISTINCT rv.user_id) as voters,
			COUNT(DISTINCT rp.user_id) as participants
		FROM retrospectives r
		LEFT JOIN roti_votes rv ON rv.retro_id = r.id
		LEFT JOIN retro_participants rp ON rp.retro_id = r.id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
	`

	var voters, participants int
	err = r.pool.QueryRow(ctx, participationQuery, teamID, retroIDs).Scan(&voters, &participants)
	if err != nil {
		return nil, err
	}

	participationRate := float64(0)
	if participants > 0 {
		participationRate = float64(voters) / float64(participants) * 100
	}

	// Get evolution data per retro (in chronological order)
	evolutionQuery := `
		SELECT r.id, r.name, r.ended_at, COALESCE(AVG(rv.rating), 0), COUNT(rv.id)
		FROM retrospectives r
		LEFT JOIN roti_votes rv ON rv.retro_id = r.id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
		GROUP BY r.id, r.name, r.ended_at
		ORDER BY r.ended_at ASC
	`

	evoRows, err := r.pool.Query(ctx, evolutionQuery, teamID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer evoRows.Close()

	var evolution []*models.RotiEvolutionPoint
	for evoRows.Next() {
		var point models.RotiEvolutionPoint
		if err := evoRows.Scan(&point.RetroID, &point.RetroName, &point.Date, &point.Average, &point.VoteCount); err != nil {
			return nil, err
		}
		evolution = append(evolution, &point)
	}

	if evolution == nil {
		evolution = []*models.RotiEvolutionPoint{}
	}

	return &models.TeamRotiStats{
		Average:           avg,
		TotalVotes:        totalVotes,
		TotalRetros:       len(retroIDs),
		Distribution:      distribution,
		ParticipationRate: participationRate,
		Evolution:         evolution,
	}, nil
}

// GetTeamMoodStats retrieves aggregated mood statistics for a team
func (r *StatsRepository) GetTeamMoodStats(ctx context.Context, teamID uuid.UUID, filter *models.StatsFilter) (*models.TeamMoodStats, error) {
	// Build the base query with optional limit
	limitClause := ""
	if filter != nil && filter.Limit > 0 {
		limitClause = "LIMIT $2"
	}

	// Get completed retrospectives for this team
	retrosQuery := `
		SELECT id
		FROM retrospectives
		WHERE team_id = $1 AND status = 'completed'
		ORDER BY ended_at DESC
	` + limitClause

	var args []interface{}
	args = append(args, teamID)
	if filter != nil && filter.Limit > 0 {
		args = append(args, filter.Limit)
	}

	rows, err := r.pool.Query(ctx, retrosQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var retroIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		retroIDs = append(retroIDs, id)
	}

	if len(retroIDs) == 0 {
		return &models.TeamMoodStats{
			Distribution: make(map[models.MoodWeather]int),
			Evolution:    []*models.MoodEvolutionPoint{},
		}, nil
	}

	// Get mood distribution
	distQuery := `
		SELECT im.mood, COUNT(*) as count
		FROM icebreaker_moods im
		JOIN retrospectives r ON r.id = im.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
		GROUP BY im.mood
	`

	distRows, err := r.pool.Query(ctx, distQuery, teamID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer distRows.Close()

	distribution := make(map[models.MoodWeather]int)
	totalMoods := 0
	for distRows.Next() {
		var mood models.MoodWeather
		var cnt int
		if err := distRows.Scan(&mood, &cnt); err != nil {
			return nil, err
		}
		distribution[mood] = cnt
		totalMoods += cnt
	}

	// Get participation rate
	participationQuery := `
		SELECT
			COUNT(DISTINCT im.user_id) as mood_submitters,
			COUNT(DISTINCT rp.user_id) as participants
		FROM retrospectives r
		LEFT JOIN icebreaker_moods im ON im.retro_id = r.id
		LEFT JOIN retro_participants rp ON rp.retro_id = r.id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
	`

	var submitters, participants int
	err = r.pool.QueryRow(ctx, participationQuery, teamID, retroIDs).Scan(&submitters, &participants)
	if err != nil {
		return nil, err
	}

	participationRate := float64(0)
	if participants > 0 {
		participationRate = float64(submitters) / float64(participants) * 100
	}

	// Get evolution data per retro
	evolutionQuery := `
		SELECT r.id, r.name, r.ended_at, im.mood, COUNT(im.id)
		FROM retrospectives r
		LEFT JOIN icebreaker_moods im ON im.retro_id = r.id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
		GROUP BY r.id, r.name, r.ended_at, im.mood
		ORDER BY r.ended_at ASC
	`

	evoRows, err := r.pool.Query(ctx, evolutionQuery, teamID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer evoRows.Close()

	evolutionMap := make(map[uuid.UUID]*models.MoodEvolutionPoint)
	for evoRows.Next() {
		var retroID uuid.UUID
		var retroName string
		var date interface{}
		var mood *models.MoodWeather
		var cnt int
		if err := evoRows.Scan(&retroID, &retroName, &date, &mood, &cnt); err != nil {
			return nil, err
		}

		if _, exists := evolutionMap[retroID]; !exists {
			point := &models.MoodEvolutionPoint{
				RetroID:      retroID,
				RetroName:    retroName,
				Distribution: make(map[models.MoodWeather]int),
			}
			if t, ok := date.(interface{ Time() (interface{}, interface{}) }); ok {
				// Handle pgx timestamp
				_ = t
			}
			evolutionMap[retroID] = point
		}

		if mood != nil {
			evolutionMap[retroID].Distribution[*mood] = cnt
			evolutionMap[retroID].MoodCount += cnt
		}
	}

	// Convert map to slice and get dates
	var evolution []*models.MoodEvolutionPoint
	for _, point := range evolutionMap {
		evolution = append(evolution, point)
	}

	// Re-query to get proper dates
	if len(evolution) > 0 {
		dateQuery := `
			SELECT id, COALESCE(ended_at, created_at) as date
			FROM retrospectives
			WHERE id = ANY($1)
		`
		dateRows, err := r.pool.Query(ctx, dateQuery, retroIDs)
		if err != nil {
			return nil, err
		}
		defer dateRows.Close()

		dateMap := make(map[uuid.UUID]interface{})
		for dateRows.Next() {
			var id uuid.UUID
			var date interface{}
			if err := dateRows.Scan(&id, &date); err != nil {
				return nil, err
			}
			dateMap[id] = date
		}

		for _, point := range evolution {
			if date, ok := dateMap[point.RetroID]; ok {
				if t, ok := date.(interface{ Time() (interface{}, interface{}) }); ok {
					_ = t
				}
			}
		}
	}

	if evolution == nil {
		evolution = []*models.MoodEvolutionPoint{}
	}

	return &models.TeamMoodStats{
		Distribution:      distribution,
		TotalMoods:        totalMoods,
		TotalRetros:       len(retroIDs),
		ParticipationRate: participationRate,
		Evolution:         evolution,
	}, nil
}

// GetUserRotiStats retrieves ROTI statistics for a specific user within a team
func (r *StatsRepository) GetUserRotiStats(ctx context.Context, teamID, userID uuid.UUID, filter *models.StatsFilter) (*models.UserRotiStats, error) {
	// Build the base query with optional limit
	limitClause := ""
	if filter != nil && filter.Limit > 0 {
		limitClause = "LIMIT $2"
	}

	// Get completed retrospectives for this team
	retrosQuery := `
		SELECT id
		FROM retrospectives
		WHERE team_id = $1 AND status = 'completed'
		ORDER BY ended_at DESC
	` + limitClause

	var args []interface{}
	args = append(args, teamID)
	if filter != nil && filter.Limit > 0 {
		args = append(args, filter.Limit)
	}

	rows, err := r.pool.Query(ctx, retrosQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var retroIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		retroIDs = append(retroIDs, id)
	}

	if len(retroIDs) == 0 {
		return &models.UserRotiStats{
			UserID:       userID,
			Distribution: make(map[int]int),
			Evolution:    []*models.RotiEvolutionPoint{},
		}, nil
	}

	// Get user's ROTI statistics
	userStatsQuery := `
		SELECT COALESCE(AVG(rv.rating), 0), COUNT(*)
		FROM roti_votes rv
		JOIN retrospectives r ON r.id = rv.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND rv.user_id = $2
		AND r.id = ANY($3)
	`

	var userAvg float64
	var userVotes int
	err = r.pool.QueryRow(ctx, userStatsQuery, teamID, userID, retroIDs).Scan(&userAvg, &userVotes)
	if err != nil {
		return nil, err
	}

	// Get team average for comparison
	teamStatsQuery := `
		SELECT COALESCE(AVG(rv.rating), 0)
		FROM roti_votes rv
		JOIN retrospectives r ON r.id = rv.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND r.id = ANY($2)
	`

	var teamAvg float64
	err = r.pool.QueryRow(ctx, teamStatsQuery, teamID, retroIDs).Scan(&teamAvg)
	if err != nil {
		return nil, err
	}

	// Get user's vote distribution
	distQuery := `
		SELECT rv.rating, COUNT(*) as count
		FROM roti_votes rv
		JOIN retrospectives r ON r.id = rv.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND rv.user_id = $2
		AND r.id = ANY($3)
		GROUP BY rv.rating
		ORDER BY rv.rating
	`

	distRows, err := r.pool.Query(ctx, distQuery, teamID, userID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer distRows.Close()

	distribution := make(map[int]int)
	for distRows.Next() {
		var rating, cnt int
		if err := distRows.Scan(&rating, &cnt); err != nil {
			return nil, err
		}
		distribution[rating] = cnt
	}

	// Get retros attended by user
	attendedQuery := `
		SELECT COUNT(DISTINCT rp.retro_id)
		FROM retro_participants rp
		JOIN retrospectives r ON r.id = rp.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND rp.user_id = $2
		AND r.id = ANY($3)
	`

	var retrosAttended int
	err = r.pool.QueryRow(ctx, attendedQuery, teamID, userID, retroIDs).Scan(&retrosAttended)
	if err != nil {
		return nil, err
	}

	participationRate := float64(0)
	if retrosAttended > 0 {
		participationRate = float64(userVotes) / float64(retrosAttended) * 100
	}

	// Get user's evolution data
	evolutionQuery := `
		SELECT r.id, r.name, COALESCE(r.ended_at, r.created_at), rv.rating
		FROM retrospectives r
		JOIN roti_votes rv ON rv.retro_id = r.id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND rv.user_id = $2
		AND r.id = ANY($3)
		ORDER BY r.ended_at ASC
	`

	evoRows, err := r.pool.Query(ctx, evolutionQuery, teamID, userID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer evoRows.Close()

	var evolution []*models.RotiEvolutionPoint
	for evoRows.Next() {
		var point models.RotiEvolutionPoint
		var rating int
		if err := evoRows.Scan(&point.RetroID, &point.RetroName, &point.Date, &rating); err != nil {
			return nil, err
		}
		point.Average = float64(rating)
		point.VoteCount = 1
		evolution = append(evolution, &point)
	}

	if evolution == nil {
		evolution = []*models.RotiEvolutionPoint{}
	}

	return &models.UserRotiStats{
		UserID:            userID,
		Average:           userAvg,
		TotalVotes:        userVotes,
		RetrosAttended:    retrosAttended,
		ParticipationRate: participationRate,
		TeamAverage:       teamAvg,
		Distribution:      distribution,
		Evolution:         evolution,
	}, nil
}

// GetUserMoodStats retrieves mood statistics for a specific user within a team
func (r *StatsRepository) GetUserMoodStats(ctx context.Context, teamID, userID uuid.UUID, filter *models.StatsFilter) (*models.UserMoodStats, error) {
	// Build the base query with optional limit
	limitClause := ""
	if filter != nil && filter.Limit > 0 {
		limitClause = "LIMIT $2"
	}

	// Get completed retrospectives for this team
	retrosQuery := `
		SELECT id
		FROM retrospectives
		WHERE team_id = $1 AND status = 'completed'
		ORDER BY ended_at DESC
	` + limitClause

	var args []interface{}
	args = append(args, teamID)
	if filter != nil && filter.Limit > 0 {
		args = append(args, filter.Limit)
	}

	rows, err := r.pool.Query(ctx, retrosQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var retroIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		retroIDs = append(retroIDs, id)
	}

	if len(retroIDs) == 0 {
		return &models.UserMoodStats{
			UserID:       userID,
			Distribution: make(map[models.MoodWeather]int),
			Evolution:    []*models.MoodEvolutionPoint{},
		}, nil
	}

	// Get user's mood distribution
	distQuery := `
		SELECT im.mood, COUNT(*) as count
		FROM icebreaker_moods im
		JOIN retrospectives r ON r.id = im.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND im.user_id = $2
		AND r.id = ANY($3)
		GROUP BY im.mood
	`

	distRows, err := r.pool.Query(ctx, distQuery, teamID, userID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer distRows.Close()

	distribution := make(map[models.MoodWeather]int)
	totalMoods := 0
	var mostCommonMood models.MoodWeather
	maxCount := 0
	for distRows.Next() {
		var mood models.MoodWeather
		var cnt int
		if err := distRows.Scan(&mood, &cnt); err != nil {
			return nil, err
		}
		distribution[mood] = cnt
		totalMoods += cnt
		if cnt > maxCount {
			maxCount = cnt
			mostCommonMood = mood
		}
	}

	// Get retros attended by user
	attendedQuery := `
		SELECT COUNT(DISTINCT rp.retro_id)
		FROM retro_participants rp
		JOIN retrospectives r ON r.id = rp.retro_id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND rp.user_id = $2
		AND r.id = ANY($3)
	`

	var retrosAttended int
	err = r.pool.QueryRow(ctx, attendedQuery, teamID, userID, retroIDs).Scan(&retrosAttended)
	if err != nil {
		return nil, err
	}

	participationRate := float64(0)
	if retrosAttended > 0 {
		participationRate = float64(totalMoods) / float64(retrosAttended) * 100
	}

	// Get user's evolution data
	evolutionQuery := `
		SELECT r.id, r.name, COALESCE(r.ended_at, r.created_at), im.mood
		FROM retrospectives r
		JOIN icebreaker_moods im ON im.retro_id = r.id
		WHERE r.team_id = $1 AND r.status = 'completed'
		AND im.user_id = $2
		AND r.id = ANY($3)
		ORDER BY r.ended_at ASC
	`

	evoRows, err := r.pool.Query(ctx, evolutionQuery, teamID, userID, retroIDs)
	if err != nil {
		return nil, err
	}
	defer evoRows.Close()

	var evolution []*models.MoodEvolutionPoint
	for evoRows.Next() {
		var point models.MoodEvolutionPoint
		var mood models.MoodWeather
		if err := evoRows.Scan(&point.RetroID, &point.RetroName, &point.Date, &mood); err != nil {
			return nil, err
		}
		point.Distribution = map[models.MoodWeather]int{mood: 1}
		point.MoodCount = 1
		evolution = append(evolution, &point)
	}

	if evolution == nil {
		evolution = []*models.MoodEvolutionPoint{}
	}

	return &models.UserMoodStats{
		UserID:            userID,
		Distribution:      distribution,
		MostCommonMood:    mostCommonMood,
		TotalMoods:        totalMoods,
		RetrosAttended:    retrosAttended,
		ParticipationRate: participationRate,
		Evolution:         evolution,
	}, nil
}

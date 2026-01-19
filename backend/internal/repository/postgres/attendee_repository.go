package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// AttendeeRepository handles retro attendance database operations
type AttendeeRepository struct {
	pool *pgxpool.Pool
}

// NewAttendeeRepository creates a new attendee repository
func NewAttendeeRepository(pool *pgxpool.Pool) *AttendeeRepository {
	return &AttendeeRepository{pool: pool}
}

// Record records a user's attendance for a retrospective
func (r *AttendeeRepository) Record(ctx context.Context, retroID, userID uuid.UUID, attended bool) error {
	query := `
		INSERT INTO retro_attendees (retrospective_id, user_id, attended)
		VALUES ($1, $2, $3)
		ON CONFLICT (retrospective_id, user_id)
		DO UPDATE SET attended = $3, recorded_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query, retroID, userID, attended)
	return err
}

// RecordBatch records attendance for multiple users at once
func (r *AttendeeRepository) RecordBatch(ctx context.Context, retroID uuid.UUID, attendees map[uuid.UUID]bool) error {
	for userID, attended := range attendees {
		if err := r.Record(ctx, retroID, userID, attended); err != nil {
			return err
		}
	}
	return nil
}

// GetByRetro gets all attendance records for a retrospective
func (r *AttendeeRepository) GetByRetro(ctx context.Context, retroID uuid.UUID) ([]*models.RetroAttendee, error) {
	query := `
		SELECT ra.id, ra.retrospective_id, ra.user_id, ra.attended, ra.recorded_at,
		       u.id, u.display_name, u.avatar_url
		FROM retro_attendees ra
		JOIN users u ON u.id = ra.user_id
		WHERE ra.retrospective_id = $1
		ORDER BY u.display_name
	`

	rows, err := r.pool.Query(ctx, query, retroID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attendees []*models.RetroAttendee
	for rows.Next() {
		var a models.RetroAttendee
		var user models.User
		err := rows.Scan(
			&a.ID, &a.RetrospectiveID, &a.UserID, &a.Attended, &a.RecordedAt,
			&user.ID, &user.DisplayName, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		a.User = &user
		attendees = append(attendees, &a)
	}

	if attendees == nil {
		attendees = []*models.RetroAttendee{}
	}

	return attendees, nil
}

// GetAttendanceRate calculates the attendance rate for a retrospective
func (r *AttendeeRepository) GetAttendanceRate(ctx context.Context, retroID uuid.UUID) (float64, error) {
	query := `
		SELECT
			CASE WHEN COUNT(*) > 0
			THEN CAST(SUM(CASE WHEN attended THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*)
			ELSE 0 END
		FROM retro_attendees
		WHERE retrospective_id = $1
	`

	var rate float64
	err := r.pool.QueryRow(ctx, query, retroID).Scan(&rate)
	if err != nil {
		return 0, err
	}

	return rate, nil
}

// GetUserAttendanceStats gets attendance statistics for a user within a team
func (r *AttendeeRepository) GetUserAttendanceStats(ctx context.Context, userID, teamID uuid.UUID) (attended int, total int, err error) {
	query := `
		SELECT
			COUNT(CASE WHEN ra.attended THEN 1 END) as attended,
			COUNT(*) as total
		FROM retro_attendees ra
		JOIN retrospectives r ON r.id = ra.retrospective_id
		WHERE ra.user_id = $1 AND r.team_id = $2
	`

	err = r.pool.QueryRow(ctx, query, userID, teamID).Scan(&attended, &total)
	return
}

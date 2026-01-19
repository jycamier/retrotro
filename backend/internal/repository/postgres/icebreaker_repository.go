package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// IcebreakerRepository handles icebreaker mood database operations
type IcebreakerRepository struct {
	pool *pgxpool.Pool
}

// NewIcebreakerRepository creates a new icebreaker repository
func NewIcebreakerRepository(pool *pgxpool.Pool) *IcebreakerRepository {
	return &IcebreakerRepository{pool: pool}
}

// SetMood sets or updates a user's mood for a retrospective
func (r *IcebreakerRepository) SetMood(ctx context.Context, retroID, userID uuid.UUID, mood models.MoodWeather) (*models.IcebreakerMood, error) {
	query := `
		INSERT INTO icebreaker_moods (id, retro_id, user_id, mood)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (retro_id, user_id)
		DO UPDATE SET mood = $4
		RETURNING id, retro_id, user_id, mood, created_at
	`

	var m models.IcebreakerMood
	err := r.pool.QueryRow(ctx, query, uuid.New(), retroID, userID, mood).Scan(
		&m.ID, &m.RetroID, &m.UserID, &m.Mood, &m.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &m, nil
}

// ListMoods lists all moods for a retrospective
func (r *IcebreakerRepository) ListMoods(ctx context.Context, retroID uuid.UUID) ([]*models.IcebreakerMood, error) {
	query := `
		SELECT im.id, im.retro_id, im.user_id, im.mood, im.created_at,
		       u.id, u.display_name, u.avatar_url
		FROM icebreaker_moods im
		JOIN users u ON u.id = im.user_id
		WHERE im.retro_id = $1
		ORDER BY im.created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, retroID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var moods []*models.IcebreakerMood
	for rows.Next() {
		var m models.IcebreakerMood
		var user models.User
		err := rows.Scan(
			&m.ID, &m.RetroID, &m.UserID, &m.Mood, &m.CreatedAt,
			&user.ID, &user.DisplayName, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		m.User = &user
		moods = append(moods, &m)
	}

	if moods == nil {
		moods = []*models.IcebreakerMood{}
	}

	return moods, nil
}

// GetMood gets a specific user's mood for a retrospective
func (r *IcebreakerRepository) GetMood(ctx context.Context, retroID, userID uuid.UUID) (*models.IcebreakerMood, error) {
	query := `
		SELECT id, retro_id, user_id, mood, created_at
		FROM icebreaker_moods
		WHERE retro_id = $1 AND user_id = $2
	`

	var m models.IcebreakerMood
	err := r.pool.QueryRow(ctx, query, retroID, userID).Scan(
		&m.ID, &m.RetroID, &m.UserID, &m.Mood, &m.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &m, nil
}

// CountMoods counts the number of moods submitted for a retrospective
func (r *IcebreakerRepository) CountMoods(ctx context.Context, retroID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM icebreaker_moods WHERE retro_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, retroID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// RotiRepository handles ROTI vote database operations
type RotiRepository struct {
	pool *pgxpool.Pool
}

// NewRotiRepository creates a new ROTI repository
func NewRotiRepository(pool *pgxpool.Pool) *RotiRepository {
	return &RotiRepository{pool: pool}
}

// SetVote sets or updates a user's ROTI vote for a retrospective
func (r *RotiRepository) SetVote(ctx context.Context, retroID, userID uuid.UUID, rating int) (*models.RotiVote, error) {
	query := `
		INSERT INTO roti_votes (id, retro_id, user_id, rating)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (retro_id, user_id)
		DO UPDATE SET rating = $4
		RETURNING id, retro_id, user_id, rating, created_at
	`

	var v models.RotiVote
	err := r.pool.QueryRow(ctx, query, uuid.New(), retroID, userID, rating).Scan(
		&v.ID, &v.RetroID, &v.UserID, &v.Rating, &v.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &v, nil
}

// GetResults gets the aggregated ROTI results for a retrospective
func (r *RotiRepository) GetResults(ctx context.Context, retroID uuid.UUID) (*models.RotiResults, error) {
	// Get average and count
	statsQuery := `
		SELECT COALESCE(AVG(rating), 0), COUNT(*)
		FROM roti_votes
		WHERE retro_id = $1
	`

	var avg float64
	var count int
	err := r.pool.QueryRow(ctx, statsQuery, retroID).Scan(&avg, &count)
	if err != nil {
		return nil, err
	}

	// Get distribution
	distQuery := `
		SELECT rating, COUNT(*) as count
		FROM roti_votes
		WHERE retro_id = $1
		GROUP BY rating
		ORDER BY rating
	`

	rows, err := r.pool.Query(ctx, distQuery, retroID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := make(map[int]int)
	for rows.Next() {
		var rating, cnt int
		if err := rows.Scan(&rating, &cnt); err != nil {
			return nil, err
		}
		distribution[rating] = cnt
	}

	// Get revealed status from retrospective
	var revealed bool
	revealedQuery := `SELECT roti_revealed FROM retrospectives WHERE id = $1`
	err = r.pool.QueryRow(ctx, revealedQuery, retroID).Scan(&revealed)
	if err != nil {
		return nil, err
	}

	return &models.RotiResults{
		Average:      avg,
		TotalVotes:   count,
		Distribution: distribution,
		Revealed:     revealed,
	}, nil
}

// ListVotes lists all ROTI votes for a retrospective
func (r *RotiRepository) ListVotes(ctx context.Context, retroID uuid.UUID) ([]*models.RotiVote, error) {
	query := `
		SELECT rv.id, rv.retro_id, rv.user_id, rv.rating, rv.created_at,
		       u.id, u.display_name, u.avatar_url
		FROM roti_votes rv
		JOIN users u ON u.id = rv.user_id
		WHERE rv.retro_id = $1
		ORDER BY rv.created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, retroID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []*models.RotiVote
	for rows.Next() {
		var v models.RotiVote
		var user models.User
		err := rows.Scan(
			&v.ID, &v.RetroID, &v.UserID, &v.Rating, &v.CreatedAt,
			&user.ID, &user.DisplayName, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		v.User = &user
		votes = append(votes, &v)
	}

	if votes == nil {
		votes = []*models.RotiVote{}
	}

	return votes, nil
}

// CountVotes counts the number of ROTI votes for a retrospective
func (r *RotiRepository) CountVotes(ctx context.Context, retroID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM roti_votes WHERE retro_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, retroID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetVote gets a specific user's ROTI vote for a retrospective
func (r *RotiRepository) GetVote(ctx context.Context, retroID, userID uuid.UUID) (*models.RotiVote, error) {
	query := `
		SELECT id, retro_id, user_id, rating, created_at
		FROM roti_votes
		WHERE retro_id = $1 AND user_id = $2
	`

	var v models.RotiVote
	err := r.pool.QueryRow(ctx, query, retroID, userID).Scan(
		&v.ID, &v.RetroID, &v.UserID, &v.Rating, &v.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &v, nil
}

// RevealResults sets the roti_revealed flag to true
func (r *RotiRepository) RevealResults(ctx context.Context, retroID uuid.UUID) error {
	query := `UPDATE retrospectives SET roti_revealed = true WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, retroID)
	return err
}

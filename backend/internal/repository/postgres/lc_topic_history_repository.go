package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// LCTopicHistoryRepository handles Lean Coffee topic history database operations
type LCTopicHistoryRepository struct {
	pool *pgxpool.Pool
}

// NewLCTopicHistoryRepository creates a new LC topic history repository
func NewLCTopicHistoryRepository(pool *pgxpool.Pool) *LCTopicHistoryRepository {
	return &LCTopicHistoryRepository{pool: pool}
}

// Create creates a new topic history entry
func (r *LCTopicHistoryRepository) Create(ctx context.Context, history *models.LCTopicHistory) (*models.LCTopicHistory, error) {
	query := `
		INSERT INTO lc_topic_history (id, retro_id, topic_id, discussion_order,
		                              total_discussion_seconds, extension_count, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	if history.ID == uuid.Nil {
		history.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		history.ID, history.RetroID, history.TopicID, history.DiscussionOrder,
		history.TotalDiscussionSeconds, history.ExtensionCount, history.StartedAt,
	).Scan(&history.ID)

	if err != nil {
		return nil, err
	}

	return history, nil
}

// Update updates a topic history entry
func (r *LCTopicHistoryRepository) Update(ctx context.Context, history *models.LCTopicHistory) error {
	query := `
		UPDATE lc_topic_history
		SET total_discussion_seconds = $2, extension_count = $3, ended_at = $4
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		history.ID, history.TotalDiscussionSeconds, history.ExtensionCount, history.EndedAt,
	)
	return err
}

// FindByTopic finds a topic history entry by retro and topic ID
func (r *LCTopicHistoryRepository) FindByTopic(ctx context.Context, retroID, topicID uuid.UUID) (*models.LCTopicHistory, error) {
	query := `
		SELECT id, retro_id, topic_id, discussion_order, total_discussion_seconds,
		       extension_count, started_at, ended_at
		FROM lc_topic_history
		WHERE retro_id = $1 AND topic_id = $2
	`

	var history models.LCTopicHistory
	err := r.pool.QueryRow(ctx, query, retroID, topicID).Scan(
		&history.ID, &history.RetroID, &history.TopicID, &history.DiscussionOrder,
		&history.TotalDiscussionSeconds, &history.ExtensionCount,
		&history.StartedAt, &history.EndedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &history, nil
}

// ListByRetro lists all topic history entries for a retrospective
func (r *LCTopicHistoryRepository) ListByRetro(ctx context.Context, retroID uuid.UUID) ([]*models.LCTopicHistory, error) {
	query := `
		SELECT id, retro_id, topic_id, discussion_order, total_discussion_seconds,
		       extension_count, started_at, ended_at
		FROM lc_topic_history
		WHERE retro_id = $1
		ORDER BY discussion_order
	`

	rows, err := r.pool.Query(ctx, query, retroID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []*models.LCTopicHistory
	for rows.Next() {
		var h models.LCTopicHistory
		err := rows.Scan(
			&h.ID, &h.RetroID, &h.TopicID, &h.DiscussionOrder,
			&h.TotalDiscussionSeconds, &h.ExtensionCount,
			&h.StartedAt, &h.EndedAt,
		)
		if err != nil {
			return nil, err
		}
		histories = append(histories, &h)
	}

	return histories, nil
}

// GetNextOrder returns the next discussion order for a retro
func (r *LCTopicHistoryRepository) GetNextOrder(ctx context.Context, retroID uuid.UUID) (int, error) {
	query := `SELECT COALESCE(MAX(discussion_order), 0) + 1 FROM lc_topic_history WHERE retro_id = $1`
	var order int
	err := r.pool.QueryRow(ctx, query, retroID).Scan(&order)
	return order, err
}

// ListByTeam lists all discussed topics for a team's completed Lean Coffee sessions
func (r *LCTopicHistoryRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.DiscussedTopic, error) {
	query := `
		SELECT lth.id, i.content, i.author_id, COALESCE(u.display_name, '') as author_name,
		       r.id as session_id, r.name as session_name,
		       lth.started_at as discussed_at,
		       lth.total_discussion_seconds, lth.extension_count
		FROM lc_topic_history lth
		JOIN items i ON i.id = lth.topic_id
		JOIN retrospectives r ON r.id = lth.retro_id
		LEFT JOIN users u ON u.id = i.author_id
		WHERE r.team_id = $1 AND r.status = 'completed' AND r.session_type = 'lean_coffee'
		ORDER BY lth.started_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []*models.DiscussedTopic
	for rows.Next() {
		var t models.DiscussedTopic
		var authorName sql.NullString
		err := rows.Scan(
			&t.ID, &t.Content, &t.AuthorID, &authorName,
			&t.SessionID, &t.SessionName,
			&t.DiscussedAt,
			&t.TotalDiscussionSeconds, &t.ExtensionCount,
		)
		if err != nil {
			return nil, err
		}
		if authorName.Valid {
			t.AuthorName = authorName.String
		}
		topics = append(topics, &t)
	}

	return topics, nil
}

// FindCurrentByRetro finds the currently active (non-ended) topic history
func (r *LCTopicHistoryRepository) FindCurrentByRetro(ctx context.Context, retroID uuid.UUID) (*models.LCTopicHistory, error) {
	query := `
		SELECT id, retro_id, topic_id, discussion_order, total_discussion_seconds,
		       extension_count, started_at, ended_at
		FROM lc_topic_history
		WHERE retro_id = $1 AND ended_at IS NULL
		ORDER BY discussion_order DESC
		LIMIT 1
	`

	var history models.LCTopicHistory
	err := r.pool.QueryRow(ctx, query, retroID).Scan(
		&history.ID, &history.RetroID, &history.TopicID, &history.DiscussionOrder,
		&history.TotalDiscussionSeconds, &history.ExtensionCount,
		&history.StartedAt, &history.EndedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &history, nil
}

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// TemplateRepository handles template database operations
type TemplateRepository struct {
	pool *pgxpool.Pool
}

// NewTemplateRepository creates a new template repository
func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

// FindByID finds a template by ID
func (r *TemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Template, error) {
	query := `
		SELECT id, name, description, columns, is_built_in, team_id, created_by, created_at
		FROM templates WHERE id = $1
	`

	var template models.Template
	var columnsJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&template.ID, &template.Name, &template.Description, &columnsJSON,
		&template.IsBuiltIn, &template.TeamID, &template.CreatedBy, &template.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(columnsJSON, &template.Columns); err != nil {
		return nil, err
	}

	// Load phase timers
	template.PhaseTimes, _ = r.GetPhaseTimers(ctx, id)

	return &template, nil
}

// ListBuiltIn lists all built-in templates
func (r *TemplateRepository) ListBuiltIn(ctx context.Context) ([]*models.Template, error) {
	query := `
		SELECT id, name, description, columns, is_built_in, team_id, created_by, created_at
		FROM templates WHERE is_built_in = true
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.Template
	for rows.Next() {
		var template models.Template
		var columnsJSON []byte
		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &columnsJSON,
			&template.IsBuiltIn, &template.TeamID, &template.CreatedBy, &template.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(columnsJSON, &template.Columns); err != nil {
			return nil, err
		}
		template.PhaseTimes, _ = r.GetPhaseTimers(ctx, template.ID)
		templates = append(templates, &template)
	}

	return templates, nil
}

// ListByTeam lists templates for a team (including built-in)
func (r *TemplateRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.Template, error) {
	query := `
		SELECT id, name, description, columns, is_built_in, team_id, created_by, created_at
		FROM templates WHERE is_built_in = true OR team_id = $1
		ORDER BY is_built_in DESC, name
	`

	rows, err := r.pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.Template
	for rows.Next() {
		var template models.Template
		var columnsJSON []byte
		err := rows.Scan(
			&template.ID, &template.Name, &template.Description, &columnsJSON,
			&template.IsBuiltIn, &template.TeamID, &template.CreatedBy, &template.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(columnsJSON, &template.Columns); err != nil {
			return nil, err
		}
		template.PhaseTimes, _ = r.GetPhaseTimers(ctx, template.ID)
		templates = append(templates, &template)
	}

	return templates, nil
}

// Create creates a new template
func (r *TemplateRepository) Create(ctx context.Context, template *models.Template) (*models.Template, error) {
	columnsJSON, err := json.Marshal(template.Columns)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO templates (id, name, description, columns, is_built_in, team_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}

	err = r.pool.QueryRow(ctx, query,
		template.ID, template.Name, template.Description, columnsJSON,
		template.IsBuiltIn, template.TeamID, template.CreatedBy,
	).Scan(&template.ID, &template.CreatedAt)

	if err != nil {
		return nil, err
	}

	return template, nil
}

// GetPhaseTimers gets the phase timers for a template
func (r *TemplateRepository) GetPhaseTimers(ctx context.Context, templateID uuid.UUID) (map[models.RetroPhase]int, error) {
	query := `
		SELECT phase, duration_seconds
		FROM template_phase_timers WHERE template_id = $1
	`

	rows, err := r.pool.Query(ctx, query, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timers := make(map[models.RetroPhase]int)
	for rows.Next() {
		var phase models.RetroPhase
		var duration int
		if err := rows.Scan(&phase, &duration); err != nil {
			return nil, err
		}
		timers[phase] = duration
	}

	return timers, nil
}

// RetrospectiveRepository handles retrospective database operations
type RetrospectiveRepository struct {
	pool *pgxpool.Pool
}

// NewRetrospectiveRepository creates a new retrospective repository
func NewRetrospectiveRepository(pool *pgxpool.Pool) *RetrospectiveRepository {
	return &RetrospectiveRepository{pool: pool}
}

// FindByID finds a retrospective by ID
func (r *RetrospectiveRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Retrospective, error) {
	query := `
		SELECT id, name, team_id, template_id, facilitator_id, status, current_phase,
		       max_votes_per_user, max_votes_per_item, anonymous_voting, anonymous_items,
		       allow_item_edit, allow_vote_change, phase_timer_overrides,
		       timer_started_at, timer_duration_seconds, timer_paused_at, timer_remaining_seconds,
		       scheduled_at, started_at, ended_at, created_at, updated_at
		FROM retrospectives WHERE id = $1
	`

	var retro models.Retrospective
	var phaseTimerOverrides []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&retro.ID, &retro.Name, &retro.TeamID, &retro.TemplateID, &retro.FacilitatorID,
		&retro.Status, &retro.CurrentPhase, &retro.MaxVotesPerUser, &retro.MaxVotesPerItem,
		&retro.AnonymousVoting, &retro.AnonymousItems, &retro.AllowItemEdit, &retro.AllowVoteChange,
		&phaseTimerOverrides, &retro.TimerStartedAt, &retro.TimerDurationSeconds, &retro.TimerPausedAt,
		&retro.TimerRemainingSeconds, &retro.ScheduledAt, &retro.StartedAt, &retro.EndedAt,
		&retro.CreatedAt, &retro.UpdatedAt,
	)

	if err == nil && phaseTimerOverrides != nil {
		json.Unmarshal(phaseTimerOverrides, &retro.PhaseTimerOverrides)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &retro, nil
}

// ListByTeam lists retrospectives for a team
func (r *RetrospectiveRepository) ListByTeam(ctx context.Context, teamID uuid.UUID, status *models.RetroStatus) ([]*models.Retrospective, error) {
	query := `
		SELECT id, name, team_id, template_id, facilitator_id, status, current_phase,
		       max_votes_per_user, max_votes_per_item, anonymous_voting, anonymous_items,
		       allow_item_edit, allow_vote_change, phase_timer_overrides,
		       timer_started_at, timer_duration_seconds, timer_paused_at, timer_remaining_seconds,
		       scheduled_at, started_at, ended_at, created_at, updated_at
		FROM retrospectives WHERE team_id = $1
	`
	args := []any{teamID}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var retros []*models.Retrospective
	for rows.Next() {
		var retro models.Retrospective
		var phaseTimerOverrides []byte
		err := rows.Scan(
			&retro.ID, &retro.Name, &retro.TeamID, &retro.TemplateID, &retro.FacilitatorID,
			&retro.Status, &retro.CurrentPhase, &retro.MaxVotesPerUser, &retro.MaxVotesPerItem,
			&retro.AnonymousVoting, &retro.AnonymousItems, &retro.AllowItemEdit, &retro.AllowVoteChange,
			&phaseTimerOverrides, &retro.TimerStartedAt, &retro.TimerDurationSeconds, &retro.TimerPausedAt,
			&retro.TimerRemainingSeconds, &retro.ScheduledAt, &retro.StartedAt, &retro.EndedAt,
			&retro.CreatedAt, &retro.UpdatedAt,
		)
		if err == nil && phaseTimerOverrides != nil {
			json.Unmarshal(phaseTimerOverrides, &retro.PhaseTimerOverrides)
		}
		if err != nil {
			return nil, err
		}
		retros = append(retros, &retro)
	}

	return retros, nil
}

// Create creates a new retrospective
func (r *RetrospectiveRepository) Create(ctx context.Context, retro *models.Retrospective) (*models.Retrospective, error) {
	query := `
		INSERT INTO retrospectives (id, name, team_id, template_id, facilitator_id, status,
		                            current_phase, max_votes_per_user, max_votes_per_item, anonymous_voting,
		                            anonymous_items, allow_item_edit, allow_vote_change, phase_timer_overrides,
		                            scheduled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`

	if retro.ID == uuid.Nil {
		retro.ID = uuid.New()
	}

	// Default max_votes_per_item to 3 if not set
	if retro.MaxVotesPerItem <= 0 {
		retro.MaxVotesPerItem = 3
	}

	var phaseTimerOverrides []byte
	if retro.PhaseTimerOverrides != nil {
		phaseTimerOverrides, _ = json.Marshal(retro.PhaseTimerOverrides)
	}

	err := r.pool.QueryRow(ctx, query,
		retro.ID, retro.Name, retro.TeamID, retro.TemplateID, retro.FacilitatorID,
		retro.Status, retro.CurrentPhase, retro.MaxVotesPerUser, retro.MaxVotesPerItem, retro.AnonymousVoting,
		retro.AnonymousItems, retro.AllowItemEdit, retro.AllowVoteChange, phaseTimerOverrides,
		retro.ScheduledAt,
	).Scan(&retro.ID, &retro.CreatedAt, &retro.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return retro, nil
}

// Update updates a retrospective
func (r *RetrospectiveRepository) Update(ctx context.Context, retro *models.Retrospective) error {
	query := `
		UPDATE retrospectives
		SET name = $2, status = $3, current_phase = $4, max_votes_per_user = $5,
		    max_votes_per_item = $6, anonymous_voting = $7, anonymous_items = $8,
		    allow_item_edit = $9, allow_vote_change = $10, phase_timer_overrides = $11,
		    facilitator_id = $12, started_at = $13, ended_at = $14, updated_at = NOW()
		WHERE id = $1
	`

	var phaseTimerOverrides []byte
	if retro.PhaseTimerOverrides != nil {
		phaseTimerOverrides, _ = json.Marshal(retro.PhaseTimerOverrides)
	}

	_, err := r.pool.Exec(ctx, query,
		retro.ID, retro.Name, retro.Status, retro.CurrentPhase,
		retro.MaxVotesPerUser, retro.MaxVotesPerItem, retro.AnonymousVoting, retro.AnonymousItems,
		retro.AllowItemEdit, retro.AllowVoteChange, phaseTimerOverrides, retro.FacilitatorID,
		retro.StartedAt, retro.EndedAt,
	)
	return err
}

// UpdateTimer updates timer fields
func (r *RetrospectiveRepository) UpdateTimer(ctx context.Context, retroID uuid.UUID, startedAt *time.Time, durationSeconds *int, pausedAt *time.Time, remainingSeconds *int) error {
	query := `
		UPDATE retrospectives
		SET timer_started_at = $2, timer_duration_seconds = $3,
		    timer_paused_at = $4, timer_remaining_seconds = $5, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, retroID, startedAt, durationSeconds, pausedAt, remainingSeconds)
	return err
}

// UpdatePhase updates the current phase
func (r *RetrospectiveRepository) UpdatePhase(ctx context.Context, retroID uuid.UUID, phase models.RetroPhase) error {
	query := `UPDATE retrospectives SET current_phase = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, retroID, phase)
	return err
}

// Delete deletes a retrospective
func (r *RetrospectiveRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM retrospectives WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ItemRepository handles item database operations
type ItemRepository struct {
	pool *pgxpool.Pool
}

// NewItemRepository creates a new item repository
func NewItemRepository(pool *pgxpool.Pool) *ItemRepository {
	return &ItemRepository{pool: pool}
}

// FindByID finds an item by ID
func (r *ItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Item, error) {
	query := `
		SELECT id, retro_id, column_id, content, author_id, group_id, position, created_at, updated_at
		FROM items WHERE id = $1
	`

	var item models.Item
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.RetroID, &item.ColumnID, &item.Content, &item.AuthorID,
		&item.GroupID, &item.Position, &item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &item, nil
}

// ListByRetro lists items for a retrospective
func (r *ItemRepository) ListByRetro(ctx context.Context, retroID uuid.UUID) ([]*models.Item, error) {
	query := `
		SELECT i.id, i.retro_id, i.column_id, i.content, i.author_id, i.group_id, i.position,
		       i.created_at, i.updated_at, COALESCE(COUNT(v.id), 0) as vote_count
		FROM items i
		LEFT JOIN votes v ON i.id = v.item_id
		WHERE i.retro_id = $1
		GROUP BY i.id
		ORDER BY i.column_id, i.position
	`

	rows, err := r.pool.Query(ctx, query, retroID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(
			&item.ID, &item.RetroID, &item.ColumnID, &item.Content, &item.AuthorID,
			&item.GroupID, &item.Position, &item.CreatedAt, &item.UpdatedAt, &item.VoteCount,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return items, nil
}

// Create creates a new item
func (r *ItemRepository) Create(ctx context.Context, item *models.Item) (*models.Item, error) {
	query := `
		INSERT INTO items (id, retro_id, column_id, content, author_id, position)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		item.ID, item.RetroID, item.ColumnID, item.Content, item.AuthorID, item.Position,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return item, nil
}

// Update updates an item
func (r *ItemRepository) Update(ctx context.Context, item *models.Item) error {
	query := `
		UPDATE items
		SET column_id = $2, content = $3, group_id = $4, position = $5, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, item.ID, item.ColumnID, item.Content, item.GroupID, item.Position)
	return err
}

// Delete deletes an item
func (r *ItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM items WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// GetNextPosition gets the next position for a new item in a column
func (r *ItemRepository) GetNextPosition(ctx context.Context, retroID uuid.UUID, columnID string) (int, error) {
	query := `SELECT COALESCE(MAX(position), -1) + 1 FROM items WHERE retro_id = $1 AND column_id = $2`
	var position int
	err := r.pool.QueryRow(ctx, query, retroID, columnID).Scan(&position)
	return position, err
}

// VoteRepository handles vote database operations
type VoteRepository struct {
	pool *pgxpool.Pool
}

// NewVoteRepository creates a new vote repository
func NewVoteRepository(pool *pgxpool.Pool) *VoteRepository {
	return &VoteRepository{pool: pool}
}

// Create creates a new vote
func (r *VoteRepository) Create(ctx context.Context, vote *models.Vote) (*models.Vote, error) {
	query := `
		INSERT INTO votes (id, item_id, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	if vote.ID == uuid.Nil {
		vote.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query, vote.ID, vote.ItemID, vote.UserID).Scan(&vote.ID, &vote.CreatedAt)
	if err != nil {
		return nil, err
	}

	return vote, nil
}

// Delete deletes a single vote from an item by a user
func (r *VoteRepository) Delete(ctx context.Context, itemID, userID uuid.UUID) error {
	// Delete only one vote (the oldest one) to support removing votes one at a time
	query := `
		DELETE FROM votes
		WHERE id = (
			SELECT id FROM votes
			WHERE item_id = $1 AND user_id = $2
			ORDER BY created_at ASC
			LIMIT 1
		)
	`
	_, err := r.pool.Exec(ctx, query, itemID, userID)
	return err
}

// CountByUser counts votes by a user in a retrospective
func (r *VoteRepository) CountByUser(ctx context.Context, retroID, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM votes v
		INNER JOIN items i ON v.item_id = i.id
		WHERE i.retro_id = $1 AND v.user_id = $2
	`
	var count int
	err := r.pool.QueryRow(ctx, query, retroID, userID).Scan(&count)
	return count, err
}

// CountByUserOnItem counts votes by a user on a specific item
func (r *VoteRepository) CountByUserOnItem(ctx context.Context, itemID, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM votes WHERE item_id = $1 AND user_id = $2`
	var count int
	err := r.pool.QueryRow(ctx, query, itemID, userID).Scan(&count)
	return count, err
}

// HasVoted checks if a user has voted on an item
func (r *VoteRepository) HasVoted(ctx context.Context, itemID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM votes WHERE item_id = $1 AND user_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, itemID, userID).Scan(&exists)
	return exists, err
}

// ActionItemRepository handles action item database operations
type ActionItemRepository struct {
	pool *pgxpool.Pool
}

// NewActionItemRepository creates a new action item repository
func NewActionItemRepository(pool *pgxpool.Pool) *ActionItemRepository {
	return &ActionItemRepository{pool: pool}
}

// FindByID finds an action item by ID
func (r *ActionItemRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ActionItem, error) {
	query := `
		SELECT id, retro_id, item_id, title, description, assignee_id, due_date,
		       is_completed, completed_at, priority, external_id, external_url,
		       created_by, created_at, updated_at
		FROM action_items WHERE id = $1
	`

	var action models.ActionItem
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&action.ID, &action.RetroID, &action.ItemID, &action.Title, &action.Description,
		&action.AssigneeID, &action.DueDate, &action.IsCompleted, &action.CompletedAt,
		&action.Priority, &action.ExternalID, &action.ExternalURL, &action.CreatedBy,
		&action.CreatedAt, &action.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &action, nil
}

// ListByRetro lists action items for a retrospective
func (r *ActionItemRepository) ListByRetro(ctx context.Context, retroID uuid.UUID) ([]*models.ActionItem, error) {
	query := `
		SELECT id, retro_id, item_id, title, description, assignee_id, due_date,
		       is_completed, completed_at, priority, external_id, external_url,
		       created_by, created_at, updated_at
		FROM action_items WHERE retro_id = $1
		ORDER BY priority DESC, created_at
	`

	rows, err := r.pool.Query(ctx, query, retroID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*models.ActionItem
	for rows.Next() {
		var action models.ActionItem
		err := rows.Scan(
			&action.ID, &action.RetroID, &action.ItemID, &action.Title, &action.Description,
			&action.AssigneeID, &action.DueDate, &action.IsCompleted, &action.CompletedAt,
			&action.Priority, &action.ExternalID, &action.ExternalURL, &action.CreatedBy,
			&action.CreatedAt, &action.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		actions = append(actions, &action)
	}

	return actions, nil
}

// Create creates a new action item
func (r *ActionItemRepository) Create(ctx context.Context, action *models.ActionItem) (*models.ActionItem, error) {
	query := `
		INSERT INTO action_items (id, retro_id, item_id, title, description, assignee_id,
		                          due_date, priority, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	if action.ID == uuid.Nil {
		action.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		action.ID, action.RetroID, action.ItemID, action.Title, action.Description,
		action.AssigneeID, action.DueDate, action.Priority, action.CreatedBy,
	).Scan(&action.ID, &action.CreatedAt, &action.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return action, nil
}

// Update updates an action item
func (r *ActionItemRepository) Update(ctx context.Context, action *models.ActionItem) error {
	query := `
		UPDATE action_items
		SET title = $2, description = $3, assignee_id = $4, due_date = $5,
		    is_completed = $6, completed_at = $7, priority = $8,
		    external_id = $9, external_url = $10, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		action.ID, action.Title, action.Description, action.AssigneeID, action.DueDate,
		action.IsCompleted, action.CompletedAt, action.Priority,
		action.ExternalID, action.ExternalURL,
	)
	return err
}

// Delete deletes an action item
func (r *ActionItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM action_items WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

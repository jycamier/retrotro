package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jycamier/retrotro/backend/internal/models"
)

// WebhookRepository handles webhook database operations
type WebhookRepository struct {
	pool *pgxpool.Pool
}

// NewWebhookRepository creates a new webhook repository
func NewWebhookRepository(pool *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{pool: pool}
}

// FindByID finds a webhook by ID
func (r *WebhookRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	query := `
		SELECT id, team_id, name, url, secret, events, is_enabled, created_by, created_at, updated_at
		FROM webhooks WHERE id = $1
	`

	var webhook models.Webhook
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&webhook.ID, &webhook.TeamID, &webhook.Name, &webhook.URL, &webhook.Secret,
		&webhook.Events, &webhook.IsEnabled, &webhook.CreatedBy,
		&webhook.CreatedAt, &webhook.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &webhook, nil
}

// ListByTeam lists all webhooks for a team
func (r *WebhookRepository) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.Webhook, error) {
	query := `
		SELECT id, team_id, name, url, secret, events, is_enabled, created_by, created_at, updated_at
		FROM webhooks WHERE team_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		err := rows.Scan(
			&webhook.ID, &webhook.TeamID, &webhook.Name, &webhook.URL, &webhook.Secret,
			&webhook.Events, &webhook.IsEnabled, &webhook.CreatedBy,
			&webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, &webhook)
	}

	if webhooks == nil {
		webhooks = []*models.Webhook{}
	}

	return webhooks, nil
}

// ListByTeamAndEvent lists enabled webhooks for a team subscribed to a specific event
func (r *WebhookRepository) ListByTeamAndEvent(ctx context.Context, teamID uuid.UUID, event string) ([]*models.Webhook, error) {
	query := `
		SELECT id, team_id, name, url, secret, events, is_enabled, created_by, created_at, updated_at
		FROM webhooks
		WHERE team_id = $1 AND is_enabled = true AND $2 = ANY(events)
		ORDER BY created_at
	`

	rows, err := r.pool.Query(ctx, query, teamID, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		err := rows.Scan(
			&webhook.ID, &webhook.TeamID, &webhook.Name, &webhook.URL, &webhook.Secret,
			&webhook.Events, &webhook.IsEnabled, &webhook.CreatedBy,
			&webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// Create creates a new webhook
func (r *WebhookRepository) Create(ctx context.Context, webhook *models.Webhook) (*models.Webhook, error) {
	query := `
		INSERT INTO webhooks (id, team_id, name, url, secret, events, is_enabled, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	if webhook.ID == uuid.Nil {
		webhook.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		webhook.ID, webhook.TeamID, webhook.Name, webhook.URL, webhook.Secret,
		webhook.Events, webhook.IsEnabled, webhook.CreatedBy,
	).Scan(&webhook.ID, &webhook.CreatedAt, &webhook.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return webhook, nil
}

// Update updates a webhook
func (r *WebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	query := `
		UPDATE webhooks
		SET name = $2, url = $3, secret = $4, events = $5, is_enabled = $6, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		webhook.ID, webhook.Name, webhook.URL, webhook.Secret, webhook.Events, webhook.IsEnabled,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete deletes a webhook
func (r *WebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// WebhookDeliveryRepository handles webhook delivery database operations
type WebhookDeliveryRepository struct {
	pool *pgxpool.Pool
}

// NewWebhookDeliveryRepository creates a new webhook delivery repository
func NewWebhookDeliveryRepository(pool *pgxpool.Pool) *WebhookDeliveryRepository {
	return &WebhookDeliveryRepository{pool: pool}
}

// Create creates a new webhook delivery record
func (r *WebhookDeliveryRepository) Create(ctx context.Context, delivery *models.WebhookDelivery) (*models.WebhookDelivery, error) {
	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event_type, payload, response_status, response_body, error_message, attempt_count, delivered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`

	if delivery.ID == uuid.Nil {
		delivery.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		delivery.ID, delivery.WebhookID, delivery.EventType, delivery.Payload,
		delivery.ResponseStatus, delivery.ResponseBody, delivery.ErrorMessage,
		delivery.AttemptCount, delivery.DeliveredAt,
	).Scan(&delivery.ID, &delivery.CreatedAt)

	if err != nil {
		return nil, err
	}

	return delivery, nil
}

// ListByWebhook lists deliveries for a webhook
func (r *WebhookDeliveryRepository) ListByWebhook(ctx context.Context, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_type, payload, response_status, response_body, error_message, attempt_count, delivered_at, created_at
		FROM webhook_deliveries WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	if limit <= 0 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx, query, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*models.WebhookDelivery
	for rows.Next() {
		var delivery models.WebhookDelivery
		err := rows.Scan(
			&delivery.ID, &delivery.WebhookID, &delivery.EventType, &delivery.Payload,
			&delivery.ResponseStatus, &delivery.ResponseBody, &delivery.ErrorMessage,
			&delivery.AttemptCount, &delivery.DeliveredAt, &delivery.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &delivery)
	}

	if deliveries == nil {
		deliveries = []*models.WebhookDelivery{}
	}

	return deliveries, nil
}

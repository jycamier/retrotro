package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

var (
	ErrWebhookNotFound = errors.New("webhook not found")
)

// WebhookService handles webhook operations
type WebhookService struct {
	webhookRepo  *postgres.WebhookRepository
	deliveryRepo *postgres.WebhookDeliveryRepository
	httpClient   *http.Client
}

// NewWebhookService creates a new webhook service
func NewWebhookService(
	webhookRepo *postgres.WebhookRepository,
	deliveryRepo *postgres.WebhookDeliveryRepository,
) *WebhookService {
	return &WebhookService{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateWebhookInput represents input for creating a webhook
type CreateWebhookInput struct {
	TeamID    uuid.UUID
	Name      string
	URL       string
	Secret    *string
	Events    []string
	IsEnabled bool
}

// Create creates a new webhook
func (s *WebhookService) Create(ctx context.Context, createdBy uuid.UUID, input CreateWebhookInput) (*models.Webhook, error) {
	webhook := &models.Webhook{
		ID:        uuid.New(),
		TeamID:    input.TeamID,
		Name:      input.Name,
		URL:       input.URL,
		Secret:    input.Secret,
		Events:    input.Events,
		IsEnabled: input.IsEnabled,
		CreatedBy: &createdBy,
	}

	return s.webhookRepo.Create(ctx, webhook)
}

// GetByID gets a webhook by ID
func (s *WebhookService) GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	webhook, err := s.webhookRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}
	return webhook, nil
}

// ListByTeam lists webhooks for a team
func (s *WebhookService) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*models.Webhook, error) {
	return s.webhookRepo.ListByTeam(ctx, teamID)
}

// UpdateWebhookInput represents input for updating a webhook
type UpdateWebhookInput struct {
	Name      *string
	URL       *string
	Secret    *string
	Events    []string
	IsEnabled *bool
}

// Update updates a webhook
func (s *WebhookService) Update(ctx context.Context, id uuid.UUID, input UpdateWebhookInput) (*models.Webhook, error) {
	webhook, err := s.webhookRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		webhook.Name = *input.Name
	}
	if input.URL != nil {
		webhook.URL = *input.URL
	}
	if input.Secret != nil {
		webhook.Secret = input.Secret
	}
	if input.Events != nil {
		webhook.Events = input.Events
	}
	if input.IsEnabled != nil {
		webhook.IsEnabled = *input.IsEnabled
	}

	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// Delete deletes a webhook
func (s *WebhookService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.webhookRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return ErrWebhookNotFound
		}
		return err
	}
	return nil
}

// ListDeliveries lists delivery history for a webhook
func (s *WebhookService) ListDeliveries(ctx context.Context, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	return s.deliveryRepo.ListByWebhook(ctx, webhookID, limit)
}

// DispatchRetroCompleted dispatches retro.completed webhooks
func (s *WebhookService) DispatchRetroCompleted(ctx context.Context, retro *models.Retrospective, data models.RetroCompletedData) {
	event := string(models.WebhookEventRetroCompleted)

	webhooks, err := s.webhookRepo.ListByTeamAndEvent(ctx, retro.TeamID, event)
	if err != nil {
		slog.Error("failed to list webhooks for retro.completed", "error", err, "teamId", retro.TeamID)
		return
	}

	if len(webhooks) == 0 {
		return
	}

	payload := models.WebhookPayload{
		Event:     models.WebhookEventRetroCompleted,
		Timestamp: time.Now().UTC(),
		RetroID:   retro.ID,
		TeamID:    retro.TeamID,
		Data:      data,
	}

	// Dispatch asynchronously
	for _, webhook := range webhooks {
		go s.dispatch(ctx, webhook, event, payload)
	}
}

// DispatchActionCreated dispatches action.created webhooks
func (s *WebhookService) DispatchActionCreated(ctx context.Context, action *models.ActionItem, teamID uuid.UUID, data models.ActionCreatedData) {
	event := string(models.WebhookEventActionCreated)

	webhooks, err := s.webhookRepo.ListByTeamAndEvent(ctx, teamID, event)
	if err != nil {
		slog.Error("failed to list webhooks for action.created", "error", err, "teamId", teamID)
		return
	}

	if len(webhooks) == 0 {
		return
	}

	payload := models.WebhookPayload{
		Event:     models.WebhookEventActionCreated,
		Timestamp: time.Now().UTC(),
		RetroID:   action.RetroID,
		TeamID:    teamID,
		Data:      data,
	}

	// Dispatch asynchronously
	for _, webhook := range webhooks {
		go s.dispatch(ctx, webhook, event, payload)
	}
}

// dispatch sends a webhook and records the delivery
func (s *WebhookService) dispatch(ctx context.Context, webhook *models.Webhook, eventType string, payload models.WebhookPayload) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal webhook payload", "error", err, "webhookId", webhook.ID)
		return
	}

	delivery := &models.WebhookDelivery{
		WebhookID:    webhook.ID,
		EventType:    eventType,
		Payload:      string(payloadBytes),
		AttemptCount: 1,
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		errMsg := err.Error()
		delivery.ErrorMessage = &errMsg
		_, _ = s.deliveryRepo.Create(ctx, delivery)
		slog.Error("failed to create webhook request", "error", err, "webhookId", webhook.ID)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Retrotro-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", eventType)
	req.Header.Set("X-Webhook-ID", webhook.ID.String())

	// Add HMAC signature if secret is set
	if webhook.Secret != nil && *webhook.Secret != "" {
		signature := s.computeSignature(payloadBytes, *webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		errMsg := err.Error()
		delivery.ErrorMessage = &errMsg
		_, _ = s.deliveryRepo.Create(ctx, delivery)
		slog.Error("failed to send webhook", "error", err, "webhookId", webhook.ID, "url", webhook.URL)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body (limit to 1KB)
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyStr := string(bodyBytes)

	delivery.ResponseStatus = &resp.StatusCode
	delivery.ResponseBody = &bodyStr
	now := time.Now()
	delivery.DeliveredAt = &now

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("webhook delivered successfully", "webhookId", webhook.ID, "status", resp.StatusCode)
	} else {
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
		delivery.ErrorMessage = &errMsg
		slog.Warn("webhook delivery failed", "webhookId", webhook.ID, "status", resp.StatusCode)
	}

	_, _ = s.deliveryRepo.Create(ctx, delivery)
}

// computeSignature computes HMAC-SHA256 signature for webhook payload
func (s *WebhookService) computeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

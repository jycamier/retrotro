package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/middleware"
	"github.com/jycamier/retrotro/backend/internal/services"
)

// WebhookHandler handles webhook endpoints
type WebhookHandler struct {
	webhookService *services.WebhookService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhookService *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// CreateWebhookRequest represents a create webhook request
type CreateWebhookRequest struct {
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Secret    *string  `json:"secret"`
	Events    []string `json:"events"`
	IsEnabled bool     `json:"isEnabled"`
}

// Create creates a new webhook
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.URL == "" || len(req.Events) == 0 {
		http.Error(w, `{"error": "name, url, and events are required"}`, http.StatusBadRequest)
		return
	}

	webhook, err := h.webhookService.Create(ctx, userID, services.CreateWebhookInput{
		TeamID:    teamID,
		Name:      req.Name,
		URL:       req.URL,
		Secret:    req.Secret,
		Events:    req.Events,
		IsEnabled: req.IsEnabled,
	})
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(webhook)
}

// List lists webhooks for a team
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teamID, err := uuid.Parse(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, `{"error": "invalid team ID"}`, http.StatusBadRequest)
		return
	}

	webhooks, err := h.webhookService.ListByTeam(ctx, teamID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(webhooks)
}

// Get gets a webhook by ID
func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookId"))
	if err != nil {
		http.Error(w, `{"error": "invalid webhook ID"}`, http.StatusBadRequest)
		return
	}

	webhook, err := h.webhookService.GetByID(ctx, webhookID)
	if err != nil {
		if err == services.ErrWebhookNotFound {
			http.Error(w, `{"error": "webhook not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(webhook)
}

// UpdateWebhookRequest represents an update webhook request
type UpdateWebhookRequest struct {
	Name      *string  `json:"name"`
	URL       *string  `json:"url"`
	Secret    *string  `json:"secret"`
	Events    []string `json:"events"`
	IsEnabled *bool    `json:"isEnabled"`
}

// Update updates a webhook
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookId"))
	if err != nil {
		http.Error(w, `{"error": "invalid webhook ID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	webhook, err := h.webhookService.Update(ctx, webhookID, services.UpdateWebhookInput{
		Name:      req.Name,
		URL:       req.URL,
		Secret:    req.Secret,
		Events:    req.Events,
		IsEnabled: req.IsEnabled,
	})
	if err != nil {
		if err == services.ErrWebhookNotFound {
			http.Error(w, `{"error": "webhook not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(webhook)
}

// Delete deletes a webhook
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookId"))
	if err != nil {
		http.Error(w, `{"error": "invalid webhook ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.webhookService.Delete(ctx, webhookID); err != nil {
		if err == services.ErrWebhookNotFound {
			http.Error(w, `{"error": "webhook not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListDeliveries lists delivery history for a webhook
func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookId"))
	if err != nil {
		http.Error(w, `{"error": "invalid webhook ID"}`, http.StatusBadRequest)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	deliveries, err := h.webhookService.ListDeliveries(ctx, webhookID, limit)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deliveries)
}

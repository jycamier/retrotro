package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// RetrotroClient is the HTTP client for the Retrotro API
type RetrotroClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// Webhook represents a webhook in the API
type Webhook struct {
	ID        string   `json:"id"`
	TeamID    string   `json:"teamId"`
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Secret    *string  `json:"secret,omitempty"`
	Events    []string `json:"events"`
	IsEnabled bool     `json:"isEnabled"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

// CreateWebhookRequest represents the request to create a webhook
type CreateWebhookRequest struct {
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Secret    *string  `json:"secret,omitempty"`
	Events    []string `json:"events"`
	IsEnabled bool     `json:"isEnabled"`
}

// UpdateWebhookRequest represents the request to update a webhook
type UpdateWebhookRequest struct {
	Name      *string  `json:"name,omitempty"`
	URL       *string  `json:"url,omitempty"`
	Secret    *string  `json:"secret,omitempty"`
	Events    []string `json:"events,omitempty"`
	IsEnabled *bool    `json:"isEnabled,omitempty"`
}

// Team represents a team in the API
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
}

// doRequest performs an authenticated HTTP request
func (c *RetrotroClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	return c.HTTPClient.Do(req)
}

// CreateWebhook creates a new webhook
func (c *RetrotroClient) CreateWebhook(ctx context.Context, teamID string, req CreateWebhookRequest) (*Webhook, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/teams/"+teamID+"/webhooks", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create webhook: %s - %s", resp.Status, string(body))
	}

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &webhook, nil
}

// GetWebhook gets a webhook by ID
func (c *RetrotroClient) GetWebhook(ctx context.Context, teamID, webhookID string) (*Webhook, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/teams/"+teamID+"/webhooks/"+webhookID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get webhook: %s - %s", resp.Status, string(body))
	}

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &webhook, nil
}

// UpdateWebhook updates a webhook
func (c *RetrotroClient) UpdateWebhook(ctx context.Context, teamID, webhookID string, req UpdateWebhookRequest) (*Webhook, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/api/v1/teams/"+teamID+"/webhooks/"+webhookID, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update webhook: %s - %s", resp.Status, string(body))
	}

	var webhook Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &webhook, nil
}

// DeleteWebhook deletes a webhook
func (c *RetrotroClient) DeleteWebhook(ctx context.Context, teamID, webhookID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "/api/v1/teams/"+teamID+"/webhooks/"+webhookID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete webhook: %s - %s", resp.Status, string(body))
	}

	return nil
}

// ListTeams lists all teams accessible by the current user
func (c *RetrotroClient) ListTeams(ctx context.Context) ([]Team, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/teams", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list teams: %s - %s", resp.Status, string(body))
	}

	var teams []Team
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return teams, nil
}

// GetTeamBySlug gets a team by its slug
func (c *RetrotroClient) GetTeamBySlug(ctx context.Context, slug string) (*Team, error) {
	teams, err := c.ListTeams(ctx)
	if err != nil {
		return nil, err
	}

	for _, team := range teams {
		if team.Slug == slug {
			return &team, nil
		}
	}

	return nil, nil
}

// ListWebhooks lists all webhooks for a team
func (c *RetrotroClient) ListWebhooks(ctx context.Context, teamID string) ([]Webhook, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/teams/"+teamID+"/webhooks", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list webhooks: %s - %s", resp.Status, string(body))
	}

	var webhooks []Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return webhooks, nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

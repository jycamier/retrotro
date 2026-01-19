package models

import (
	"time"

	"github.com/google/uuid"
)

// WebhookEvent represents the types of events that can trigger webhooks
type WebhookEvent string

const (
	WebhookEventRetroCompleted WebhookEvent = "retro.completed"
	WebhookEventActionCreated  WebhookEvent = "action.created"
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	TeamID    uuid.UUID      `json:"teamId" db:"team_id"`
	Name      string         `json:"name" db:"name"`
	URL       string         `json:"url" db:"url"`
	Secret    *string        `json:"-" db:"secret"` // Hidden from JSON responses
	Events    []string       `json:"events" db:"events"`
	IsEnabled bool           `json:"isEnabled" db:"is_enabled"`
	CreatedBy *uuid.UUID     `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time      `json:"updatedAt" db:"updated_at"`
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	WebhookID      uuid.UUID  `json:"webhookId" db:"webhook_id"`
	EventType      string     `json:"eventType" db:"event_type"`
	Payload        string     `json:"payload" db:"payload"`
	ResponseStatus *int       `json:"responseStatus,omitempty" db:"response_status"`
	ResponseBody   *string    `json:"responseBody,omitempty" db:"response_body"`
	ErrorMessage   *string    `json:"errorMessage,omitempty" db:"error_message"`
	AttemptCount   int        `json:"attemptCount" db:"attempt_count"`
	DeliveredAt    *time.Time `json:"deliveredAt,omitempty" db:"delivered_at"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
}

// WebhookPayload represents the base structure for all webhook payloads
type WebhookPayload struct {
	Event     WebhookEvent `json:"event"`
	Timestamp time.Time    `json:"timestamp"`
	RetroID   uuid.UUID    `json:"retroId"`
	TeamID    uuid.UUID    `json:"teamId"`
	Data      interface{}  `json:"data"`
}

// RetroCompletedData represents the data payload for retro.completed events
type RetroCompletedData struct {
	Name             string          `json:"name"`
	FacilitatorID    uuid.UUID       `json:"facilitatorId"`
	ParticipantCount int             `json:"participantCount"`
	ItemCount        int             `json:"itemCount"`
	ActionCount      int             `json:"actionCount"`
	AverageRoti      *float64        `json:"averageRoti,omitempty"`
	Moods            []MoodData      `json:"moods,omitempty"`
	RotiVotes        []RotiVoteData  `json:"rotiVotes,omitempty"`
}

// MoodData represents mood information in webhook payloads
type MoodData struct {
	UserID uuid.UUID   `json:"userId"`
	Mood   MoodWeather `json:"mood"`
}

// RotiVoteData represents ROTI vote information in webhook payloads
type RotiVoteData struct {
	UserID uuid.UUID `json:"userId"`
	Rating int       `json:"rating"`
}

// ActionCreatedData represents the data payload for action.created events
type ActionCreatedData struct {
	ActionID     uuid.UUID  `json:"actionId"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	AssigneeID   *uuid.UUID `json:"assigneeId,omitempty"`
	AssigneeName *string    `json:"assigneeName,omitempty"`
	DueDate      *time.Time `json:"dueDate,omitempty"`
	Priority     int        `json:"priority"`
	CreatedBy    uuid.UUID  `json:"createdBy"`
	SourceItemID *uuid.UUID `json:"sourceItemId,omitempty"`
}

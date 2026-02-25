package handlers

import (
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/bus"
	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
	"github.com/jycamier/retrotro/backend/internal/services"
	"github.com/jycamier/retrotro/backend/internal/websocket"
)

var Module = fx.Module("handler",
	fx.Provide(
		NewAuthHandlerFx,
		NewTeamHandler,
		NewRetrospectiveHandlerFx,
		NewWebSocketHandlerFx,
		NewStatsHandler,
		NewAdminHandlerFx,
		NewWebhookHandlerFx,
	),
)

// NewAuthHandlerFx creates the auth handler for fx
func NewAuthHandlerFx(authService *services.AuthService, cfg *config.Config, devSeeder *services.DevSeeder) *AuthHandler {
	return NewAuthHandler(authService, cfg.OIDC, cfg.DevMode, devSeeder, cfg.CORSOrigins)
}

// NewRetrospectiveHandlerFx creates the retrospective handler for fx
func NewRetrospectiveHandlerFx(retroService *services.RetrospectiveService, timerService *services.TimerService) *RetrospectiveHandler {
	return NewRetrospectiveHandler(retroService, timerService)
}

// NewWebSocketHandlerFx creates the WebSocket handler for fx
func NewWebSocketHandlerFx(
	hub *websocket.Hub,
	bridge bus.MessageBus,
	retroService *services.RetrospectiveService,
	timerService *services.TimerService,
	authService *services.AuthService,
	teamMemberRepo *postgres.TeamMemberRepository,
	attendeeRepo *postgres.AttendeeRepository,
) *WebSocketHandler {
	return NewWebSocketHandler(hub, bridge, retroService, timerService, authService, teamMemberRepo, attendeeRepo)
}

// NewAdminHandlerFx creates the admin handler for fx
func NewAdminHandlerFx(userRepo *postgres.UserRepository, teamRepo *postgres.TeamRepository, teamMemberRepo *postgres.TeamMemberRepository) *AdminHandler {
	return NewAdminHandler(userRepo, teamRepo, teamMemberRepo)
}

// NewWebhookHandlerFx creates the webhook handler for fx
func NewWebhookHandlerFx(webhookService *services.WebhookService) *WebhookHandler {
	return NewWebhookHandler(webhookService)
}

package services

import (
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/auth"
	"github.com/jycamier/retrotro/backend/internal/bus"
	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

var Module = fx.Module("service",
	fx.Provide(
		NewAuthServiceFx,
		NewTeamServiceFx,
		NewRetrospectiveServiceFx,
		NewTimerServiceFx,
		NewStatsServiceFx,
		NewDevSeederFx,
		NewWebhookServiceFx,
		NewLeanCoffeeServiceFx,
		NewAnalysisServiceFx,
	),
)

// NewAuthServiceFx creates the auth service for fx
func NewAuthServiceFx(oidc *auth.OIDCProvider, userRepo *postgres.UserRepository, jit *auth.JITProvisioner, cfg *config.Config) *AuthService {
	return NewAuthService(oidc, userRepo, jit, cfg.JWT)
}

// NewTeamServiceFx creates the team service for fx
func NewTeamServiceFx(teamRepo *postgres.TeamRepository, teamMemberRepo *postgres.TeamMemberRepository, userRepo *postgres.UserRepository) *TeamService {
	return NewTeamService(teamRepo, teamMemberRepo, userRepo)
}

// NewRetrospectiveServiceFx creates the retrospective service for fx
func NewRetrospectiveServiceFx(
	retroRepo *postgres.RetrospectiveRepository,
	templateRepo *postgres.TemplateRepository,
	itemRepo *postgres.ItemRepository,
	voteRepo *postgres.VoteRepository,
	actionRepo *postgres.ActionItemRepository,
	icebreakerRepo *postgres.IcebreakerRepository,
	rotiRepo *postgres.RotiRepository,
	webhookService *WebhookService,
) *RetrospectiveService {
	return NewRetrospectiveService(retroRepo, templateRepo, itemRepo, voteRepo, actionRepo, icebreakerRepo, rotiRepo, webhookService)
}

// NewTimerServiceFx creates the timer service for fx
func NewTimerServiceFx(bridge bus.MessageBus, retroRepo *postgres.RetrospectiveRepository, templateRepo *postgres.TemplateRepository) *TimerService {
	return NewTimerService(bridge, retroRepo, templateRepo)
}

// NewStatsServiceFx creates the stats service for fx
func NewStatsServiceFx(statsRepo *postgres.StatsRepository, teamMemberRepo *postgres.TeamMemberRepository) *StatsService {
	return NewStatsService(statsRepo, teamMemberRepo)
}

// NewDevSeederFx creates the dev seeder for fx (nil if not in dev mode)
func NewDevSeederFx(cfg *config.Config, teamRepo *postgres.TeamRepository, teamMemberRepo *postgres.TeamMemberRepository) *DevSeeder {
	if cfg.DevMode {
		return NewDevSeeder(teamRepo, teamMemberRepo)
	}
	return nil
}

// NewWebhookServiceFx creates the webhook service for fx
func NewWebhookServiceFx(webhookRepo *postgres.WebhookRepository, deliveryRepo *postgres.WebhookDeliveryRepository) *WebhookService {
	return NewWebhookService(webhookRepo, deliveryRepo)
}

// NewAnalysisServiceFx creates the analysis service for fx
func NewAnalysisServiceFx(lcService *LeanCoffeeService) *AnalysisService {
	return NewAnalysisService(lcService)
}

// NewLeanCoffeeServiceFx creates the lean coffee service for fx
func NewLeanCoffeeServiceFx(
	retroRepo *postgres.RetrospectiveRepository,
	itemRepo *postgres.ItemRepository,
	voteRepo *postgres.VoteRepository,
	topicHistoryRepo *postgres.LCTopicHistoryRepository,
) *LeanCoffeeService {
	return NewLeanCoffeeService(retroRepo, itemRepo, voteRepo, topicHistoryRepo)
}

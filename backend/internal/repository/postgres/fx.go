package postgres

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/config"
)

var Module = fx.Module("repository",
	fx.Provide(
		NewDatabasePool,
		NewUserRepository,
		NewTeamRepository,
		NewTeamMemberRepository,
		NewTemplateRepository,
		NewRetrospectiveRepository,
		NewItemRepository,
		NewVoteRepository,
		NewActionItemRepository,
		NewIcebreakerRepository,
		NewRotiRepository,
		NewStatsRepository,
		NewAttendeeRepository,
		NewWebhookRepository,
		NewWebhookDeliveryRepository,
		NewLCTopicHistoryRepository,
	),
)

// NewDatabasePool creates and configures the database connection pool
func NewDatabasePool(lc fx.Lifecycle, cfg *config.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return nil, errors.New("failed to connect to database")
	}

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		return nil, errors.New("failed to ping database")
	}

	slog.Info("connected to database")

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pool.Close()
			slog.Info("database connection closed")
			return nil
		},
	})

	return pool, nil
}

package migration

import (
	"context"
	"embed"
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/config"
)

//go:embed sql/*.sql
var migrationsFS embed.FS

var Module = fx.Module("migration",
	fx.Invoke(RunMigrations),
)

// RunMigrations runs all pending database migrations on startup
func RunMigrations(lc fx.Lifecycle, cfg *config.Config) error {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return runMigrations(cfg.DatabaseURL)
		},
	})
	return nil
}

func runMigrations(databaseURL string) error {
	slog.Info("running database migrations")

	source, err := iofs.New(migrationsFS, "sql")
	if err != nil {
		slog.Error("failed to create migration source", "error", err)
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		slog.Error("failed to create migrate instance", "error", err)
		return err
	}
	defer func() { _, _ = m.Close() }()

	// Run all pending migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database schema is up to date")
			return nil
		}
		slog.Error("migration failed", "error", err)
		return err
	}

	version, dirty, _ := m.Version()
	slog.Info("migrations completed successfully", "version", version, "dirty", dirty)

	return nil
}

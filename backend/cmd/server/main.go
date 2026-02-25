package main

import (
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/auth"
	"github.com/jycamier/retrotro/backend/internal/bus"
	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/handlers"
	"github.com/jycamier/retrotro/backend/internal/logger"
	"github.com/jycamier/retrotro/backend/internal/migration"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
	"github.com/jycamier/retrotro/backend/internal/services"
	"github.com/jycamier/retrotro/backend/internal/websocket"
)

func main() {
	// Load logger config early to configure fx logger
	logCfg := logger.LoadConfig()
	logger.Setup(logCfg)

	fx.New(
		// Use our slog-based logger for fx (or NopLogger if FX_LOGS=false)
		logger.FxLogger(logCfg),

		// Supply the already-loaded config
		fx.Supply(logCfg),

		// Modules
		///
		logger.Module,
		config.Module,
		migration.Module,
		postgres.Module,
		auth.Module,
		websocket.Module,
		bus.Module,
		services.Module,
		handlers.Module,
		handlers.RouterModule,
		handlers.ServerModule,
	).Run()
}

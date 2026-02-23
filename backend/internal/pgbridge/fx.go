package pgbridge

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/websocket"
)

var Module = fx.Module("pgbridge",
	fx.Provide(NewPGBridgeFx),
)

// NewPGBridgeFx creates the PGBridge with lifecycle management
func NewPGBridgeFx(lc fx.Lifecycle, hub *websocket.Hub, pool *pgxpool.Pool) *PGBridge {
	bridge := NewPGBridge(hub, pool)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := bridge.Start(ctx); err != nil {
				slog.Error("pgbridge: failed to start", "error", err)
				return err
			}
			slog.Info("pgbridge: started", "podId", bridge.podID)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			bridge.Stop()
			slog.Info("pgbridge: stopped")
			return nil
		},
	})

	return bridge
}

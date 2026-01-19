package websocket

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
)

var Module = fx.Module("websocket",
	fx.Provide(NewHubFx),
)

// NewHubFx creates the WebSocket hub with lifecycle management
func NewHubFx(lc fx.Lifecycle) *Hub {
	hub := NewHub()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go hub.Run()
			slog.Info("websocket hub started")
			return nil
		},
	})

	return hub
}

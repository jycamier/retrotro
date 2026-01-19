package config

import (
	"errors"
	"log/slog"

	"go.uber.org/fx"
)

var Module = fx.Module("config",
	fx.Provide(func() (*Config, error) {
		cfg, err := Load()
		if err != nil {
			slog.Error("failed to load configuration", "error", err)
			return nil, errors.New("failed to load configuration")
		}
		return cfg, nil
	}),
)

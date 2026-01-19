package auth

import (
	"context"
	"errors"
	"log/slog"

	"go.uber.org/fx"

	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

var Module = fx.Module("auth",
	fx.Provide(
		NewOIDCProviderFx,
		NewJITProvisionerFx,
	),
)

// NewOIDCProviderFx creates the OIDC provider for fx
func NewOIDCProviderFx(cfg *config.Config) (*OIDCProvider, error) {
	provider, err := NewOIDCProvider(context.Background(), cfg.OIDC)
	if err != nil {
		slog.Error("failed to initialize OIDC provider", "error", err)
		return nil, errors.New("failed to initialize OIDC provider")
	}
	return provider, nil
}

// NewJITProvisionerFx creates the JIT provisioner for fx
func NewJITProvisionerFx(cfg *config.Config, teamRepo *postgres.TeamRepository, teamMemberRepo *postgres.TeamMemberRepository) *JITProvisioner {
	return NewJITProvisioner(cfg.OIDC.JIT, teamRepo, teamMemberRepo)
}

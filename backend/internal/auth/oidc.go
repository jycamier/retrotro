package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/jycamier/retrotro/backend/internal/config"
)

// OIDCProvider handles OIDC authentication
type OIDCProvider struct {
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
	config      config.OIDCConfig
}

// OIDCClaims represents the claims from an OIDC token
type OIDCClaims struct {
	Subject       string                 `json:"sub"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Name          string                 `json:"name"`
	Picture       string                 `json:"picture"`
	Groups        []string               `json:"groups"`
	Raw           map[string]interface{} `json:"-"`
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(ctx context.Context, cfg config.OIDCConfig) (*OIDCProvider, error) {
	if cfg.IssuerURL == "" {
		// Return a mock provider for development without OIDC
		return &OIDCProvider{config: cfg}, nil
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}

	return &OIDCProvider{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
		config:       cfg,
	}, nil
}

// GetAuthURL returns the OAuth2 authorization URL
func (p *OIDCProvider) GetAuthURL(state string) string {
	if p.oauth2Config == nil {
		return ""
	}
	return p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for tokens
func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if p.oauth2Config == nil {
		return nil, fmt.Errorf("OIDC not configured")
	}
	return p.oauth2Config.Exchange(ctx, code)
}

// VerifyIDToken verifies an ID token and extracts claims
func (p *OIDCProvider) VerifyIDToken(ctx context.Context, rawIDToken string) (*OIDCClaims, error) {
	if p.verifier == nil {
		return nil, fmt.Errorf("OIDC not configured")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Also get raw claims for JIT provisioning
	var rawClaims map[string]interface{}
	if err := idToken.Claims(&rawClaims); err != nil {
		return nil, fmt.Errorf("failed to parse raw claims: %w", err)
	}
	claims.Raw = rawClaims

	return &claims, nil
}

// GetIssuer returns the OIDC issuer URL
func (p *OIDCProvider) GetIssuer() string {
	return p.config.IssuerURL
}

// IsConfigured returns true if OIDC is properly configured
func (p *OIDCProvider) IsConfigured() bool {
	return p.provider != nil
}

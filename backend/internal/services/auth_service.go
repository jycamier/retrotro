package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/auth"
	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/models"
	"github.com/jycamier/retrotro/backend/internal/repository/postgres"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

// UserRepository interface for auth service
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByOIDC(ctx context.Context, subject, issuer string) (*models.User, error)
	FindOrCreate(ctx context.Context, subject, issuer, email, name string, avatarURL *string) (*models.User, bool, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// AuthService handles authentication operations
type AuthService struct {
	oidcProvider   *auth.OIDCProvider
	userRepo       UserRepository
	jitProvisioner *auth.JITProvisioner
	jwtManager     *auth.JWTManager
}

// NewAuthService creates a new auth service
func NewAuthService(oidcProvider *auth.OIDCProvider, userRepo UserRepository, jitProvisioner *auth.JITProvisioner, jwtConfig config.JWTConfig) *AuthService {
	return &AuthService{
		oidcProvider:   oidcProvider,
		userRepo:       userRepo,
		jitProvisioner: jitProvisioner,
		jwtManager:     auth.NewJWTManager(jwtConfig.Secret, jwtConfig.AccessTokenTTL, jwtConfig.RefreshTokenTTL),
	}
}

// GetAuthURL returns the OIDC authorization URL
func (s *AuthService) GetAuthURL(state string) string {
	return s.oidcProvider.GetAuthURL(state)
}

// HandleCallback handles the OIDC callback
func (s *AuthService) HandleCallback(ctx context.Context, code string) (*models.User, *auth.TokenPair, error) {
	// Exchange code for tokens
	oauth2Token, err := s.oidcProvider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}

	// Get ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, nil, errors.New("no id_token in response")
	}

	// Verify and extract claims
	claims, err := s.oidcProvider.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return nil, nil, err
	}

	// Find or create user
	var avatarURL *string
	if claims.Picture != "" {
		avatarURL = &claims.Picture
	}

	user, isNew, err := s.userRepo.FindOrCreate(ctx, claims.Subject, s.oidcProvider.GetIssuer(), claims.Email, claims.Name, avatarURL)
	if err != nil {
		return nil, nil, err
	}

	// JIT provision teams if enabled
	slog.Info("OIDC claims received", "user", user.Email, "groups_claim", claims.Raw["groups"], "all_claims", claims.Raw)
	if err := s.jitProvisioner.ProvisionUser(ctx, user, claims.Raw); err != nil {
		slog.Error("JIT provisioning failed", "error", err, "user", user.Email)
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// Generate JWT tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.DisplayName, user.IsAdmin)
	if err != nil {
		return nil, nil, err
	}

	if isNew {
		// TODO: send welcome email or notification
	}

	return user, tokenPair, nil
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// Validate refresh token
	userID, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Generate new token pair
	return s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.DisplayName, user.IsAdmin)
}

// ValidateToken validates an access token and returns the claims
func (s *AuthService) ValidateToken(token string) (*auth.JWTClaims, error) {
	return s.jwtManager.ValidateAccessToken(token)
}

// GetUserByID gets a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// IsOIDCConfigured returns true if OIDC is properly configured
func (s *AuthService) IsOIDCConfigured() bool {
	return s.oidcProvider.IsConfigured()
}

// DevLogin handles development mode login (bypasses OIDC)
func (s *AuthService) DevLogin(ctx context.Context, email, displayName string) (*models.User, *auth.TokenPair, error) {
	// Use email as a pseudo subject/issuer for dev mode
	subject := "dev-" + email
	issuer := "dev-mode"

	// Find or create user
	user, _, err := s.userRepo.FindOrCreate(ctx, subject, issuer, email, displayName, nil)
	if err != nil {
		return nil, nil, err
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// Generate JWT tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.DisplayName, user.IsAdmin)
	if err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
}

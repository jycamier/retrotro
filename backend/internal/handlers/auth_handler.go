package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jycamier/retrotro/backend/internal/config"
	"github.com/jycamier/retrotro/backend/internal/middleware"
	"github.com/jycamier/retrotro/backend/internal/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
	oidcConfig  config.OIDCConfig
	devMode     bool
	devSeeder   *services.DevSeeder
	frontendURL string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, oidcConfig config.OIDCConfig, devMode bool, devSeeder *services.DevSeeder, corsOrigins []string) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		oidcConfig:  oidcConfig,
		devMode:     devMode,
		devSeeder:   devSeeder,
		frontendURL: corsOrigins[0],
	}
}

// GetLoginInfo returns information about available authentication methods
func (h *AuthHandler) GetLoginInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"oidcConfigured": h.authService.IsOIDCConfigured(),
		"devMode":        h.devMode,
	})
}

// Login initiates OIDC login flow
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.authService.IsOIDCConfigured() {
		// Return info about available auth methods
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"oidcConfigured": false,
			"devMode":        h.devMode,
			"message":        "OIDC not configured. Use dev login if in dev mode.",
		})
		return
	}

	// Generate state parameter
	state := generateState()

	// Store state in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   int(10 * time.Minute / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	// Redirect to OIDC provider
	authURL := h.authService.GetAuthURL(state)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// DevLogin handles development mode login (bypasses OIDC)
func (h *AuthHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		http.Error(w, `{"error": "dev login not available"}`, http.StatusForbidden)
		return
	}

	var body struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if body.Email == "" {
		http.Error(w, `{"error": "email is required"}`, http.StatusBadRequest)
		return
	}

	if body.DisplayName == "" {
		body.DisplayName = body.Email
	}

	ctx := r.Context()
	user, tokens, err := h.authService.DevLogin(ctx, body.Email, body.DisplayName)
	if err != nil {
		http.Error(w, `{"error": "login failed: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Set refresh token as HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":        user,
		"accessToken": tokens.AccessToken,
		"expiresAt":   tokens.ExpiresAt,
	})
}

// Callback handles OIDC callback
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, `{"error": "missing state cookie"}`, http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		http.Error(w, `{"error": "invalid state"}`, http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Check for error
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		http.Error(w, `{"error": "`+errParam+`", "description": "`+errDesc+`"}`, http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, `{"error": "missing authorization code"}`, http.StatusBadRequest)
		return
	}

	// Handle callback
	user, tokens, err := h.authService.HandleCallback(ctx, code)
	if err != nil {
		http.Error(w, `{"error": "authentication failed: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Set refresh token as HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	})

	// Redirect to frontend with access token
	// Use /auth/success which is a frontend-only route
	http.Redirect(w, r, h.frontendURL+"/auth/success?token="+tokens.AccessToken, http.StatusTemporaryRedirect)

	_ = user // User info could be included in response if needed
}

// Logout handles logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out"})
}

// RefreshToken refreshes the access token
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get refresh token from cookie or body
	var refreshToken string

	cookie, err := r.Cookie("refresh_token")
	if err == nil {
		refreshToken = cookie.Value
	} else {
		// Try body
		var body struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			refreshToken = body.RefreshToken
		}
	}

	if refreshToken == "" {
		http.Error(w, `{"error": "missing refresh token"}`, http.StatusBadRequest)
		return
	}

	// Refresh tokens
	tokens, err := h.authService.RefreshToken(ctx, refreshToken)
	if err != nil {
		http.Error(w, `{"error": "failed to refresh token"}`, http.StatusUnauthorized)
		return
	}

	// Update refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accessToken": tokens.AccessToken,
		"expiresAt":   tokens.ExpiresAt,
	})
}

// GetCurrentUser returns the current user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	user, err := h.authService.GetUserByID(ctx, userID)
	if err != nil {
		http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetDevUsers returns the list of dev users for quick switching
func (h *AuthHandler) GetDevUsers(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		http.Error(w, `{"error": "dev mode not enabled"}`, http.StatusForbidden)
		return
	}

	if h.devSeeder == nil {
		http.Error(w, `{"error": "dev seeder not initialized"}`, http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	response, err := h.devSeeder.GetDevUsersInfo(ctx)
	if err != nil {
		http.Error(w, `{"error": "failed to get dev users: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateState generates a random state string
func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

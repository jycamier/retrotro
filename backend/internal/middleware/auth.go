package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/jycamier/retrotro/backend/internal/auth"
)

// ContextKey is a custom type for context keys
type ContextKey string

const (
	UserIDKey    ContextKey = "userID"
	UserEmailKey ContextKey = "userEmail"
	UserNameKey  ContextKey = "userName"
	IsAdminKey   ContextKey = "isAdmin"
	ClaimsKey    ContextKey = "claims"
)

// JWTAuth is middleware that validates JWT tokens
func JWTAuth(secret string) func(http.Handler) http.Handler {
	jwtManager := auth.NewJWTManager(secret, 15, 168)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token
			claims, err := jwtManager.ValidateAccessToken(token)
			if err != nil {
				if err == auth.ErrExpiredToken {
					http.Error(w, `{"error": "token expired"}`, http.StatusUnauthorized)
					return
				}
				http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Parse user ID
			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				http.Error(w, `{"error": "invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, userID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, UserNameKey, claims.Name)
			ctx = context.WithValue(ctx, IsAdminKey, claims.IsAdmin)
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID gets the user ID from the context
func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetUserEmail gets the user email from the context
func GetUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

// GetUserName gets the user name from the context
func GetUserName(ctx context.Context) string {
	if name, ok := ctx.Value(UserNameKey).(string); ok {
		return name
	}
	return ""
}

// IsAdmin checks if the user is an admin
func IsAdmin(ctx context.Context) bool {
	if isAdmin, ok := ctx.Value(IsAdminKey).(bool); ok {
		return isAdmin
	}
	return false
}

// RequireAdmin is middleware that requires admin role
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r.Context()) {
			http.Error(w, `{"error": "admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

package middleware

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const (
	LoggerKey ContextKey = "logger"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush implements http.Flusher for streaming support
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// SlogLogger is middleware that adds structured logging with request context
func SlogLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get request ID from chi middleware
		requestID := chimiddleware.GetReqID(r.Context())

		// Create logger with request context
		logger := slog.With(
			"requestId", requestID,
			"method", r.Method,
			"path", r.URL.Path,
		)

		// Add logger to context
		ctx := context.WithValue(r.Context(), LoggerKey, logger)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// Log request completion
		duration := time.Since(start)
		logger.Info("request completed",
			"status", wrapped.statusCode,
			"duration", duration.String(),
			"durationMs", duration.Milliseconds(),
		)
	})
}

// GetLogger gets the logger from context, falls back to default slog if not found
func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// LoggerWithUser returns a logger enriched with user context
func LoggerWithUser(ctx context.Context) *slog.Logger {
	logger := GetLogger(ctx)

	userID := GetUserID(ctx)
	if userID.String() != "00000000-0000-0000-0000-000000000000" {
		logger = logger.With("userId", userID.String())
	}

	userName := GetUserName(ctx)
	if userName != "" {
		logger = logger.With("userName", userName)
	}

	return logger
}

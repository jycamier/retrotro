package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config holds logger configuration
type Config struct {
	Level   string // debug, info, warn, error
	Format  string // txt, json
	FxLogs  bool   // enable/disable fx framework logs
}

// LoadConfig loads logger configuration from environment variables
func LoadConfig() Config {
	return Config{
		Level:   getEnv("LOGGER_LEVEL", "info"),
		Format:  getEnv("LOGGER_FORMAT", "txt"),
		FxLogs:  getEnvBool("FX_LOGS", false),
	}
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// Setup initializes the global slog logger
func Setup(cfg Config) {
	level := parseLevel(cfg.Level)
	handler := createHandler(os.Stdout, cfg.Format, level)
	slog.SetDefault(slog.New(handler))
}

// SetupWithWriter initializes the logger with a custom writer (useful for testing)
func SetupWithWriter(cfg Config, w io.Writer) {
	level := parseLevel(cfg.Level)
	handler := createHandler(w, cfg.Format, level)
	slog.SetDefault(slog.New(handler))
}

// parseLevel converts a string level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// createHandler creates the appropriate slog handler based on format
func createHandler(w io.Writer, format string, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Add source file info in debug mode
	}

	switch strings.ToLower(format) {
	case "json":
		return slog.NewJSONHandler(w, opts)
	default:
		return slog.NewTextHandler(w, opts)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

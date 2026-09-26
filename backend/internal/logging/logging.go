package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a JSON structured logger with the given level and environment attribute.
func New(level string, appEnv string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler).With("environment", appEnv)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

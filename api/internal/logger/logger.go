package logger

import (
	"log/slog"
	"os"
)

// New returns a structured logger. When LOG_FORMAT=json output is JSON;
// otherwise human-readable. LOG_LEVEL sets minimum level (DEBUG, INFO, WARN, ERROR).
func New(service string) *slog.Logger {
	format := os.Getenv("LOG_FORMAT")
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "INFO"
	}
	var level slog.Level
	_ = level.UnmarshalText([]byte(levelStr))

	var h slog.Handler
	if format == "json" {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	return slog.New(h).With("service", service)
}

package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a *slog.Logger configured for the given environment.
//
// In production (env == "production") it writes JSON to stdout at the
// requested level. In all other environments it writes human-readable
// text at DEBUG level so every log line is visible during development.
//
// level accepts "debug", "info", "warn" or "error" (case-insensitive).
// An unrecognised value defaults to "info".
func New(env, level string) *slog.Logger {
	var lvl slog.Level

	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if strings.ToLower(env) == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// Development: text handler is easier to read in a terminal.
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

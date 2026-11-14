package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config configures the baseline logger.
type Config struct {
	Level  string
	Format string // "text" or "json"
	Output io.Writer
}

// DefaultConfig returns default logging configuration.
func DefaultConfig() Config {
	return Config{Level: "info", Format: "text", Output: os.Stderr}
}

// New creates a baseline slog.Logger with redaction enabled.
func New(cfg Config) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "json":
		h = slog.NewJSONHandler(cfg.Output, opts)
	default:
		h = slog.NewTextHandler(cfg.Output, opts)
	}
	return slog.New(Wrap(h))
}

// SetupDefault installs a baseline redacting logger as slog default.
func SetupDefault(cfg Config) *slog.Logger {
	l := New(cfg)
	slog.SetDefault(l)
	return l
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

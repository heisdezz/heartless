package config

import (
	"log/slog"

	"heartless/internal/logging"
)

// Logging holds logging-related configuration.
type Logging struct {
	Level  string
	Format string
}

// DefaultLogging returns baseline logging config.
func DefaultLogging() Logging {
	return Logging{Level: "info", Format: "text"}
}

// SetupLogging configures and installs the default slog logger with redaction.
func SetupLogging(cfg Logging) *slog.Logger {
	return logging.SetupDefault(logging.Config{Level: cfg.Level, Format: cfg.Format})
}

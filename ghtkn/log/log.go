// Package log provides structured logging functionality for ghtkn.
// It uses slog with tint handler for colored output to stderr.
package log

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// New creates a new structured logger with the specified version and log level.
// The logger outputs to stderr with colored formatting using tint handler.
// It includes "program" and "version" attributes in all log entries.
func New(version string, level slog.Level) *slog.Logger {
	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: level,
	})).With("program", "ghtkn", "version", version)
}

// ErrUnknownLogLevel is returned when an invalid log level string is provided to ParseLevel.
var ErrUnknownLogLevel = errors.New("unknown log level")

// ParseLevel converts a string log level to slog.Level.
// Supported levels are: "debug", "info", "warn", "error".
// Returns ErrUnknownLogLevel if the level string is not recognized.
func ParseLevel(lvl string) (slog.Level, error) {
	switch lvl {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, ErrUnknownLogLevel
	}
}

type Logger struct {
	Expire              func(logger *slog.Logger, exDate time.Time)
	FailedToOpenBrowser func(logger *slog.Logger, err error)
}

func NewLogger() *Logger {
	return &Logger{
		Expire: func(logger *slog.Logger, exDate time.Time) {
			logger.Debug("access token expires", "expiration_date", exDate)
		},
		FailedToOpenBrowser: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to open the browser")
		},
	}
}

// Package log provides structured logging functionality for ghtkn.
// It uses slog with tint handler for colored output to stderr.
package log

import (
	"log/slog"
	"time"

	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// NewLogger creates a new Logger instance with default logging functions.
// Each logging function is pre-configured with appropriate log levels and messages.
func NewLogger() *publog.Logger {
	return &publog.Logger{
		Expire: func(logger *slog.Logger, exDate time.Time) {
			logger.Debug("access token expires", "expiration_date", exDate.Format(time.RFC3339))
		},
		FailedToOpenBrowser: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to open the browser")
		},
		OpenedBrowser: func(logger *slog.Logger, url string) {
			logger.Info("opened the browser", "url", url)
		},
		AccessTokenIsNotFoundInBackend: func(logger *slog.Logger) {
			logger.Debug("access token is not found in backend")
		},
	}
}

// InitLogger initializes any nil logging functions in the provided Logger with default implementations.
// This function allows partial customization of logging behavior by only overriding specific
// log functions while falling back to defaults for unset functions.
func InitLogger(l *publog.Logger) {
	defaultLogger := NewLogger()
	if l.Expire == nil {
		l.Expire = defaultLogger.Expire
	}
	if l.FailedToOpenBrowser == nil {
		l.FailedToOpenBrowser = defaultLogger.FailedToOpenBrowser
	}
	if l.OpenedBrowser == nil {
		l.OpenedBrowser = defaultLogger.OpenedBrowser
	}
	if l.AccessTokenIsNotFoundInBackend == nil {
		l.AccessTokenIsNotFoundInBackend = defaultLogger.AccessTokenIsNotFoundInBackend
	}
}

// Package log provides structured logging functionality for ghtkn.
// It uses slog with tint handler for colored output to stderr.
package log

import (
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Logger provides structured logging functions for ghtkn operations.
// Each field is a function that logs specific events with appropriate log levels.
type Logger struct {
	// Expire logs when an access token expiration date is processed.
	Expire func(logger *slog.Logger, exDate time.Time)
	// FailedToOpenBrowser logs when the browser cannot be opened for authentication.
	FailedToOpenBrowser func(logger *slog.Logger, err error)
	// FailedToGetAccessTokenFromKeyring logs when access token retrieval from keyring fails.
	FailedToGetAccessTokenFromKeyring func(logger *slog.Logger, err error)
	// AccessTokenIsNotFoundInKeyring logs when no access token is found in the keyring.
	AccessTokenIsNotFoundInKeyring func(logger *slog.Logger)
	// FailedToGetAppFromKeyring logs when app retrieval from keyring fails.
	FailedToGetAppFromKeyring func(logger *slog.Logger, err error)
	// AppIsNotFoundInKeyring logs when no app is found in the keyring.
	AppIsNotFoundInKeyring func(logger *slog.Logger)
}

// NewLogger creates a new Logger instance with default logging functions.
// Each logging function is pre-configured with appropriate log levels and messages.
func NewLogger() *Logger {
	return &Logger{
		Expire: func(logger *slog.Logger, exDate time.Time) {
			logger.Debug("access token expires", "expiration_date", keyring.FormatDate(exDate))
		},
		FailedToOpenBrowser: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to open the browser")
		},
		FailedToGetAccessTokenFromKeyring: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to get access token from keyring")
		},
		AccessTokenIsNotFoundInKeyring: func(logger *slog.Logger) {
			logger.Debug("access token is not found in keyring")
		},
		FailedToGetAppFromKeyring: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to get app from keyring")
		},
		AppIsNotFoundInKeyring: func(logger *slog.Logger) {
			logger.Debug("app is not found in keyring")
		},
	}
}

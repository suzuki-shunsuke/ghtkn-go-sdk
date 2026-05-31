// Package log provides structured logging functionality for ghtkn.
// It uses slog with tint handler for colored output to stderr.
package log

import (
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// NewLogger creates a new Logger instance with default logging functions.
// Each logging function is pre-configured with appropriate log levels and messages.
func NewLogger() *publog.Logger {
	return &publog.Logger{
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
	if l.FailedToGetAccessTokenFromKeyring == nil {
		l.FailedToGetAccessTokenFromKeyring = defaultLogger.FailedToGetAccessTokenFromKeyring
	}
	if l.AccessTokenIsNotFoundInKeyring == nil {
		l.AccessTokenIsNotFoundInKeyring = defaultLogger.AccessTokenIsNotFoundInKeyring
	}
	if l.FailedToGetAppFromKeyring == nil {
		l.FailedToGetAppFromKeyring = defaultLogger.FailedToGetAppFromKeyring
	}
	if l.AppIsNotFoundInKeyring == nil {
		l.AppIsNotFoundInKeyring = defaultLogger.AppIsNotFoundInKeyring
	}
}

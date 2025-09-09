// Package log provides structured logging functionality for ghtkn.
// It uses slog with tint handler for colored output to stderr.
package log

import (
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type Logger struct {
	Expire                            func(logger *slog.Logger, exDate time.Time)
	FailedToOpenBrowser               func(logger *slog.Logger, err error)
	FailedToGetAccessTokenFromKeyring func(logger *slog.Logger, err error)
	AccessTokenIsNotFoundInKeyring    func(logger *slog.Logger)
}

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
			logger.Info("access token is not found in keyring")
		},
	}
}

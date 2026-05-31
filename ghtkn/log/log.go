// Package log provides the public logging hook type for ghtkn.
// Each field is a function that logs a specific event with an appropriate log level.
package log

import (
	"log/slog"
	"time"
)

// Logger provides structured logging functions for ghtkn operations.
// Each field is a function that logs specific events with appropriate log levels.
type Logger struct {
	// Expire logs when an access token expiration date is processed.
	Expire func(logger *slog.Logger, exDate time.Time)
	// FailedToOpenBrowser logs when the browser cannot be opened for authentication.
	FailedToOpenBrowser func(logger *slog.Logger, err error)
	// FailedToGetAccessTokenFromBackend logs when access token retrieval from backend fails.
	FailedToGetAccessTokenFromBackend func(logger *slog.Logger, err error)
	// AccessTokenIsNotFoundInBackend logs when no access token is found in the backend.
	AccessTokenIsNotFoundInBackend func(logger *slog.Logger)
}

// Package log provides the public logging hook type for ghtkn.
// Each field is a function that logs a specific event with an appropriate log level.
package log

import (
	"io"
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
	// FailedToCopyOnetimeCodeToClipboard logs when the one-time code cannot be copied to the clipboard.
	FailedToCopyOnetimeCodeToClipboard func(logger *slog.Logger, stderr io.Writer, err error)
	// OpenedBrowser logs when the browser has been opened for authentication.
	OpenedBrowser func(logger *slog.Logger, url string)
	// AccessTokenIsNotFoundInBackend logs when no access token is found in the backend.
	AccessTokenIsNotFoundInBackend func(logger *slog.Logger)
	// AgentWarning logs a security-relevant warning returned by the ghtkn agent
	// (e.g. a still-valid refresh token that failed to refresh, a possible leak).
	AgentWarning func(logger *slog.Logger, stderr io.Writer, message string)
}

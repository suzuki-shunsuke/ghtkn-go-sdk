// Package api provides the public request types for the ghtkn client.
package api

import (
	"errors"
	"time"
)

// InputGet contains the input parameters for token retrieval operations.
// It provides configuration options for specifying which app to use,
// where to find configuration, and token expiration requirements.
type InputGet struct {
	AppName        string        // Name of the app to use (defaults to GHTKN_APP environment variable)
	ConfigFilePath string        // Path to configuration file (auto-detected if empty)
	AppOwner       string        // GitHub App Owner
	MinExpiration  time.Duration // Minimum time before token expiration to trigger renewal
	// EnableDeviceFlow overrides whether the OAuth device flow may run to create a
	// new token. nil means "not specified", in which case the GHTKN_ENABLE_DEVICE_FLOW
	// environment variable decides (default enabled; set it to "false" to disable).
	EnableDeviceFlow *bool
	// SkipAccountPicker overrides whether the GitHub Device Flow account picker is
	// skipped. nil means "not specified", in which case GHTKN_SKIP_ACCOUNT_PICKER
	// decides (default disabled; set it to "true" to enable).
	SkipAccountPicker *bool
}

// InputRevoke contains the input parameters for revoking access tokens.
// The tokens to revoke are the tokens stored in the backend for each app in
// AppNames. When AppNames is empty, it falls back to the app selected by
// GHTKN_APP (or the default app).
type InputRevoke struct {
	// AppNames are the names of the apps whose stored tokens should be revoked.
	AppNames []string
	// ConfigFilePath is the path to the configuration file (auto-detected if empty).
	ConfigFilePath string
}

// ErrDisableDeviceFlow is returned when a new GitHub App access token is needed
// but the device flow is disabled (GHTKN_ENABLE_DEVICE_FLOW=false). The device flow
// is interactive (it waits for a one-time code), so it can't be completed by a
// background or non-interactive process such as a coding agent. Rather than
// blocking, the operation fails immediately. The message instructs a coding
// agent NOT to run `ghtkn get` itself (it would fail the same way) but to ask
// the user to run `ghtkn auth` in their own interactive terminal.
var ErrDisableDeviceFlow = errors.New("a GitHub App User access token can't be created via Device Flow because it's disabled by GHTKN_ENABLE_DEVICE_FLOW=false. The Device Flow is interactive and can't be completed by a background or non-interactive process. If you are a coding agent, do NOT run `ghtkn get` yourself because it would fail the same way; instead, ask the user to run `ghtkn auth` in their own interactive terminal to authenticate")

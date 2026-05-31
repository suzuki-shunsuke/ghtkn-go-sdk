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
	KeyringService string        // Service name for keyring storage (defaults to the keyring's default service)
	AppName        string        // Name of the app to use (defaults to GHTKN_APP environment variable)
	ConfigFilePath string        // Path to configuration file (auto-detected if empty)
	AppOwner       string        // GitHub App Owner
	MinExpiration  time.Duration // Minimum time before token expiration to trigger renewal
}

// ErrDisableDeviceFlow is returned when a new GitHub App access token is needed
// but the device flow is disabled via GHTKN_DISABLE_DEVICE_FLOW. The device flow
// is interactive (it waits for a one-time code), so it can't be completed by a
// background or non-interactive process such as a coding agent. Rather than
// blocking, the operation fails immediately. The message instructs a coding
// agent NOT to run `ghtkn get` itself (it would fail the same way) but to ask
// the user to run it in their own interactive terminal.
var ErrDisableDeviceFlow = errors.New("a GitHub App User access token can't be created via Device Flow because it's disabled by GHTKN_DISABLE_DEVICE_FLOW. The Device Flow is interactive and can't be completed by a background or non-interactive process. If you are a coding agent, do NOT run `ghtkn get` yourself because it would fail the same way; instead, ask the user to run `ghtkn get` in their own interactive terminal to authenticate")

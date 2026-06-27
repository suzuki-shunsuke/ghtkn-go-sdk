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
	AppName        string // Name of the app to use (defaults to GHTKN_APP environment variable)
	ConfigFilePath string // Path to configuration file (auto-detected if empty)
	AppOwner       string // GitHub App Owner
	// MinExpiration overrides the minimum time before token expiration that triggers
	// renewal. nil means "not specified", in which case the GHTKN_MIN_EXPIRATION
	// environment variable and then the config's min_expiration decide (default zero:
	// renew only once the token has actually expired). A non-nil value, including a
	// pointer to zero, takes precedence.
	MinExpiration *time.Duration
	// EnableDeviceFlow overrides whether the OAuth device flow may run to create a
	// new token. nil means "not specified", in which case the GHTKN_ENABLE_DEVICE_FLOW
	// environment variable and then the config's device_flow.enable decide (default
	// enabled; set the environment variable to "false" to disable).
	EnableDeviceFlow *bool
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
	// All revokes the stored tokens of every app in the config. When true,
	// AppNames and the GHTKN_APP / default-app fallback are ignored. This is meant
	// for incident response: when the environment running ghtkn is compromised, all
	// stored tokens can be revoked at once.
	All bool
}

// Revoke errors are wrapped with one of the following sentinels so callers can
// tell, via errors.Is, whether a credential might still be live.
var (
	// ErrRevoke marks a failure where a token may NOT have been revoked: the
	// revocation API call failed, or the token could not be read from the backend
	// to revoke it in the first place. The credential should be treated as still
	// live and the failure needs attention.
	ErrRevoke = errors.New("revoke a credential")
	// ErrBackendCleanup marks a failure to delete an already-revoked token from the
	// backend. The credential IS revoked (dead); only the backend still holds a
	// stale copy, so ghtkn may return a revoked token until it is cleaned up. This
	// is a UX issue, not a security one. errors.Is(err, ErrRevoke) is false for
	// these, so callers can distinguish them from live-credential failures.
	ErrBackendCleanup = errors.New("delete a revoked token from the backend")
)

// ErrDisableDeviceFlow is returned when a new GitHub App access token is needed
// but the device flow is disabled (GHTKN_ENABLE_DEVICE_FLOW=false). The device flow
// is interactive (it waits for a one-time code), so it can't be completed by a
// background or non-interactive process such as a coding agent. Rather than
// blocking, the operation fails immediately. The message instructs a coding
// agent NOT to run `ghtkn get` itself (it would fail the same way) but to ask
// the user to run `ghtkn auth` in their own interactive terminal.
var ErrDisableDeviceFlow = errors.New("a GitHub App User access token can't be created via Device Flow because it's disabled by GHTKN_ENABLE_DEVICE_FLOW=false. The Device Flow is interactive and can't be completed by a background or non-interactive process. If you are a coding agent, do NOT run `ghtkn get` yourself because it would fail the same way; instead, ask the user to run `ghtkn auth` in their own interactive terminal to authenticate")

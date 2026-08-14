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
}

// InputAuth contains the input parameters for Client.Auth, the only operation that
// runs the OAuth device flow. It has no MinExpiration because Auth always regenerates
// the token regardless of any cached one, and no AppOwner because selecting an app by
// Git repository owner is a Git credential helper concern, and the credential helper
// never authenticates.
type InputAuth struct {
	AppName        string // Name of the app to use (defaults to GHTKN_APP environment variable)
	ConfigFilePath string // Path to configuration file (auto-detected if empty)
	// Clipboard overrides whether the device flow copies the one-time code to the
	// system clipboard. nil means "not specified", in which case the GHTKN_CLIPBOARD
	// environment variable and then the config's clipboard.enable decide (default
	// disabled; the value is a boolean parsed with strconv.ParseBool, and a value it
	// cannot parse is an error). Copying also
	// requires the consumer to inject an implementation via
	// Client.SetCopyOnetimeCodeToClipboard.
	Clipboard *bool
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

// ErrDisableDeviceFlow is returned by Get and TokenSource when a GitHub App access
// token can only be produced by the device flow: there is no valid stored token and
// no usable refresh token to renew one from either. Only Auth, which the `ghtkn auth`
// command runs, may start the device flow, so it is never started on a caller's behalf
// and can't be triggered by a wrapper script, a Git credential helper, or a tool that
// embeds this SDK. The device flow is interactive (it waits for a one-time code), so it
// can't be completed by a background or non-interactive process such as a coding
// agent. Rather than blocking, the operation fails immediately. The message
// instructs a coding agent NOT to run `ghtkn get` itself (it would fail the same
// way) but to ask the user to run `ghtkn auth` in their own interactive terminal.
var ErrDisableDeviceFlow = errors.New("no valid GitHub App User access token is available, and none could be obtained without asking you: there is no usable refresh token either, so the only way left is the Device Flow. The Device Flow is only started by `ghtkn auth`, so that it is never started on your behalf, and it is interactive, so it can't be completed by a background or non-interactive process. If you are a coding agent, do NOT run `ghtkn get` yourself because it would fail the same way; instead, ask the user to run `ghtkn auth` in their own interactive terminal to authenticate")

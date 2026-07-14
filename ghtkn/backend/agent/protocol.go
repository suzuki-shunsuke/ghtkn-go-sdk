// Package agent defines the socket protocol and client used to talk to a running
// ghtkn agent. The agent is a long-running process that caches GitHub App access
// tokens and serves them over a Unix domain socket using newline-delimited JSON.
//
// Both the agent server (in the ghtkn CLI) and the SDK's agent backend client
// depend on this package so that the wire format and socket path stay in sync.
package agent

import (
	"encoding/json"
	"time"
)

// ProtocolVersion is the newest version of the agent socket protocol the agent
// speaks; the client stamps it on every request (see Send). The server accepts any
// version in the range [MinProtocolVersion, ProtocolVersion] and serves an older but
// still-supported client with that older version's behavior, so the two sides never
// speak past each other and old clients keep working after the agent is upgraded.
// A client newer than ProtocolVersion means the agent itself is out of date.
// Version history:
//
//	0: pre-versioning clients (no protocol_version field). The client owns the token
//	   lifecycle: it runs the device flow, mints tokens itself, and pushes them with
//	   the SET command. The agent never runs the device flow and never refreshes. The
//	   agent still serves these clients in a legacy compatibility mode (SET is kept).
//	1: the server owns the token lifecycle: it runs the device flow and mints tokens
//	   itself, checks expiration, refreshes with refresh tokens, and revokes tokens.
//	   A version-1 client never sends SET.
const ProtocolVersion = 1

// MinProtocolVersion is the oldest protocol version the agent still serves. A client
// older than this is rejected with RespObsoleteClient. It is currently 0 so that
// pre-versioning (SET-based) clients keep working; raise it when legacy support for
// an old version is dropped.
const MinProtocolVersion = 0

// ProtocolVersionServerLifecycle is the protocol version at which the server took
// over the token lifecycle (server-side device flow and refresh tokens). A request
// below this version is a legacy client: the agent must not run the device flow or
// refresh for it. See ProtocolVersion's version history.
const ProtocolVersionServerLifecycle = 1

// Command names and well-known response strings of the agent socket protocol.
const (
	CommandGet    = "GET"
	CommandDelete = "DELETE"
	CommandRevoke = "REVOKE"
	CommandStatus = "STATUS"
	CommandStop   = "STOP"
	CommandUnlock = "UNLOCK"
	// CommandSet stores a client-minted token (legacy, protocol version 0 only). The
	// agent keeps handling it so pre-versioning clients that mint tokens themselves
	// keep working; a version-1 client never sends it because the server owns the
	// token lifecycle.
	CommandSet = "SET"

	// RespNotFound is the Response.Error value returned by GET when no token is
	// cached for the client ID.
	RespNotFound = "not found"
	// RespLocked is the Response.Error value returned by GET when the agent is still
	// locked (its data key has not been loaded with a passphrase yet).
	RespLocked = "locked"
	// RespObsoleteClient is the Response.Error value returned when the client's
	// protocol version is older than the version the agent supports. Its value is a
	// full sentence because a pre-versioning client can only print it verbatim: it
	// tells the user to upgrade ghtkn (or the tool embedding the SDK).
	RespObsoleteClient = "the connecting client is too old for this ghtkn agent; upgrade ghtkn (or the tool that embeds the ghtkn SDK) to a version that supports the current agent protocol"
	// RespObsoleteAgent is the Response.Error value returned when the client's
	// protocol version is newer than the version the agent supports: the agent is out
	// of date. It tells the user to upgrade and restart the ghtkn agent.
	RespObsoleteAgent = "this ghtkn agent is older than the connecting client; upgrade ghtkn and restart the agent ('ghtkn agent') to a version that supports the client's protocol"
)

// SecretBytes holds a secret (the unlock passphrase) as a mutable byte slice rather
// than an immutable string, so it can be zeroed after use (see Zero) to shorten how
// long the plaintext lives in memory; a Go string cannot be reliably zeroed. On the
// wire it is a plain JSON string, so the format is unchanged (including for
// pre-versioning clients that send a string).
type SecretBytes []byte

// MarshalJSON encodes the secret as a plain JSON string.
func (s SecretBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s)) //nolint:wrapcheck
}

// UnmarshalJSON decodes a JSON string into the byte slice. JSON decoding materializes
// a transient string internally, but the retained value is a scrubbable []byte.
func (s *SecretBytes) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err //nolint:wrapcheck
	}
	*s = SecretBytes(str)
	return nil
}

// Zero overwrites the secret's bytes with zeros. It is a no-op on a nil slice.
func (s SecretBytes) Zero() {
	for i := range s {
		s[i] = 0
	}
}

// Request is a single request sent to the agent.
// The wire format is one JSON object per line (newline-delimited JSON).
type Request struct {
	// ProtocolVersion is the client's protocol version (see ProtocolVersion). Send
	// stamps it automatically. The server serves any version in the range
	// [MinProtocolVersion, ProtocolVersion]; an absent field decodes to 0 (a
	// pre-versioning, SET-based client served in legacy mode), a version above
	// ProtocolVersion means the agent is out of date.
	ProtocolVersion int `json:"protocol_version,omitempty"`
	// Command is one of CommandGet, CommandDelete, CommandRevoke, CommandStatus,
	// CommandStop, CommandUnlock, or (legacy, version 0 only) CommandSet.
	Command string `json:"command"`
	// ClientID identifies the GitHub App (used by GET, DELETE, and legacy SET).
	ClientID string `json:"client_id,omitempty"`
	// Token is the client-minted access token payload to store (legacy CommandSet,
	// protocol version 0 only). A version-1 client leaves it empty because the server
	// mints tokens itself.
	Token json.RawMessage `json:"token,omitempty"`
	// ClientIDs are the GitHub Apps whose stored tokens REVOKE should revoke and
	// delete in one batch.
	ClientIDs []string `json:"client_ids,omitempty"`
	// StartDeviceFlow lets a GET start (or join) the server-side device flow when
	// no valid token is cached. The client sets it only when its own device-flow gate
	// is enabled; a plain GET (false) is a pure probe that never starts a flow.
	StartDeviceFlow bool `json:"start_device_flow,omitempty"`
	// AwaitDeviceFlow marks a GET as polling for the result of a device flow the
	// client already started. The server reports Pending while the flow runs and then
	// returns the freshly minted token as is, WITHOUT the MinExpiration freshness
	// check, since it is the newest token obtainable even if short-lived.
	AwaitDeviceFlow bool `json:"await_device_flow,omitempty"`
	// MinExpiration is how long a cached token must still be valid for GET to return
	// it. The server treats a token expiring within MinExpiration as a miss, so the
	// freshness decision is made server-side (the agent owns the token lifecycle).
	MinExpiration time.Duration `json:"min_expiration,omitempty"`
	// Passphrase unlocks the agent (used by UNLOCK only). It is sent over the
	// 0600, same-user Unix socket and is never persisted. It is SecretBytes so the
	// client and server can zero it after use; on the wire it is a plain JSON string.
	Passphrase SecretBytes `json:"passphrase,omitempty"`
	// EnableRefreshToken enables refreshing an expiring access token with a stored
	// refresh token (used by UNLOCK only). It is bound to the passphrase moment on
	// purpose: the agent distrusts the ambient environment, so this security-relevant
	// setting is gated by the passphrase rather than an env var or config file.
	EnableRefreshToken bool `json:"enable_refresh_token,omitempty"`
	// RefreshTokenTTL is how long a stored token may sit unused before the agent
	// discards it (used by UNLOCK only, and only when EnableRefreshToken is set). The
	// agent periodically sweeps token files whose access token expired more than this
	// long ago, so an infrequently used refresh token does not linger indefinitely. A
	// zero value leaves the agent's default in place.
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl,omitempty"`
	// ConfirmRefreshTokenRemoval confirms that the user accepts dropping the stored
	// refresh tokens on an UNLOCK without EnableRefreshToken (used by UNLOCK only). The
	// first such unlock is answered with RefreshTokenRemovalPending when a still-valid
	// refresh token is stored; the client prompts the user and, on yes, re-sends the same
	// unlock with this set so the agent proceeds and strips the refresh tokens.
	ConfirmRefreshTokenRemoval bool `json:"confirm_refresh_token_removal,omitempty"`
}

// Response is a single response returned by the agent for a Request.
// The wire format is one JSON object per line (newline-delimited JSON).
type Response struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`
	// Token is the cached access token payload (returned by a successful GET).
	Token json.RawMessage `json:"token,omitempty"`
	// Count is the number of cached tokens (returned by STATUS).
	Count int `json:"count,omitempty"`
	// Locked reports whether the agent is locked (returned by STATUS).
	Locked bool `json:"locked,omitempty"`
	// Initialized reports whether an agent key already exists, i.e. whether unlock
	// asks for an existing passphrase rather than creating a new one (returned by
	// STATUS).
	Initialized bool `json:"initialized,omitempty"`
	// Error describes the failure when OK is false.
	Error string `json:"error,omitempty"`
	// Pending reports that the server-side device flow is in progress and no token
	// is cached yet. The client keeps polling GET while it is true.
	Pending bool `json:"pending,omitempty"`
	// UserCode is the device flow one-time code the user enters on GitHub (returned
	// while Pending, so the client can display it).
	UserCode string `json:"user_code,omitempty"`
	// VerificationURI is the GitHub URL where the user enters the one-time code
	// (returned while Pending).
	VerificationURI string `json:"verification_uri,omitempty"`
	// ExpiresIn is the number of seconds until the one-time code expires (returned
	// while Pending).
	ExpiresIn int `json:"expires_in,omitempty"`
	// RevokeFailed lists the client IDs whose credential REVOKE could not revoke, so
	// the credential may still be live. The client reports these as revoke failures.
	RevokeFailed []string `json:"revoke_failed,omitempty"`
	// CleanupFailed lists the client IDs whose credential REVOKE revoked but whose
	// stored copy it then could not delete. The client reports these as backend
	// cleanup failures (the credential is already revoked).
	CleanupFailed []string `json:"cleanup_failed,omitempty"`
	// RefreshTokenEnabled reports whether the agent will refresh expiring access
	// tokens with stored refresh tokens (returned by UNLOCK and STATUS) so the client
	// can surface the current state to the user.
	RefreshTokenEnabled bool `json:"refresh_token_enabled,omitempty"`
	// Warning carries a non-fatal but security-relevant message the client must show
	// the user (e.g. on GET). The agent sets it when a still-valid refresh token fails
	// to refresh, which suggests the refresh token may have been leaked or revoked. It
	// does not make OK false: the request may still succeed (or fall back to the device
	// flow) while the warning is surfaced.
	Warning string `json:"warning,omitempty"`
	// RefreshTokenRemovalPending reports that an UNLOCK without EnableRefreshToken was not
	// applied because a still-valid refresh token is stored and the removal was not yet
	// confirmed (OK is false and the agent stays locked). The client prompts the user and,
	// on yes, re-sends the same unlock with ConfirmRefreshTokenRemoval set.
	RefreshTokenRemovalPending bool `json:"refresh_token_removal_pending,omitempty"`
}

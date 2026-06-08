// Package agent defines the socket protocol and client used to talk to a running
// ghtkn agent. The agent is a long-running process that caches GitHub App access
// tokens and serves them over a Unix domain socket using newline-delimited JSON.
//
// Both the agent server (in the ghtkn CLI) and the SDK's agent backend client
// depend on this package so that the wire format and socket path stay in sync.
package agent

import "encoding/json"

// Command names and well-known response strings of the agent socket protocol.
const (
	CommandGet    = "GET"
	CommandSet    = "SET"
	CommandStatus = "STATUS"
	CommandStop   = "STOP"
	CommandUnlock = "UNLOCK"

	// RespNotFound is the Response.Error value returned by GET when no token is
	// cached for the client ID.
	RespNotFound = "not found"
	// RespLocked is the Response.Error value returned by GET and SET when the agent
	// is still locked (its data key has not been loaded with a passphrase yet).
	RespLocked = "locked"
)

// Request is a single request sent to the agent.
// The wire format is one JSON object per line (newline-delimited JSON).
type Request struct {
	// Command is one of CommandGet, CommandSet, CommandStatus, CommandStop, or
	// CommandUnlock.
	Command string `json:"command"`
	// ClientID identifies the GitHub App (used by GET and SET).
	ClientID string `json:"client_id,omitempty"`
	// Token is the opaque access token payload (used by SET).
	Token json.RawMessage `json:"token,omitempty"`
	// Passphrase unlocks the agent (used by UNLOCK only). It is sent over the
	// 0600, same-user Unix socket and is never persisted.
	Passphrase string `json:"passphrase,omitempty"`
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
}

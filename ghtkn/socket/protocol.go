// Package socket defines the wire protocol and client for the ghtkn daemon
// socket. The protocol is HTTP over a Unix domain socket; types in this package
// are exchanged as JSON request and response bodies.
//
// The package is split between two endpoints exposed by the daemon:
//
//   - Token socket: clients in containers/VMs call PathToken with a capability
//     token to retrieve a GitHub access token.
//   - Mgmt socket: host-side tooling calls PathSessions to issue and revoke
//     capability tokens.
package socket

import "time"

// Endpoint paths on the daemon HTTP API.
const (
	// PathToken is the token endpoint. POST with a TokenRequest body and an
	// "Authorization: Bearer <capability-token>" header to receive a
	// TokenResponse.
	PathToken = "/v1/token"
	// PathSessions is the management endpoint. POST with an IssueRequest body
	// to mint a capability token. DELETE PathSessions+"/{token}" to revoke.
	// GET to list active capabilities. Only exposed on the mgmt socket.
	PathSessions = "/v1/sessions"
)

// Environment variables read by the socket-mode client.
const (
	// EnvSock holds the path to the daemon token socket. When set, the SDK
	// switches to socket mode instead of accessing the keyring directly.
	EnvSock = "GHTKN_SOCK"
	// EnvCapToken holds the capability token presented to the daemon.
	EnvCapToken = "GHTKN_CAP_TOKEN"
)

// TokenRequest is the request body for POST PathToken.
type TokenRequest struct {
	// App is the configured app name to retrieve a token for.
	App string `json:"app"`
}

// TokenResponse is the response body for POST PathToken.
type TokenResponse struct {
	// AccessToken is the GitHub App user access token.
	AccessToken string `json:"token"`
	// Login is the GitHub login associated with the token. May be empty if
	// the daemon could not resolve it.
	Login string `json:"login,omitempty"`
	// ExpirationDate is when the access token expires.
	ExpirationDate time.Time `json:"expires_at"`
}

// IssueRequest is the request body for POST PathSessions on the mgmt socket.
type IssueRequest struct {
	// AppAllowlist limits which app names the issued capability token may
	// request tokens for. An empty list means no apps are allowed.
	AppAllowlist []string `json:"app_allowlist"`
	// TTL is how long the capability token remains valid.
	TTL time.Duration `json:"ttl"`
	// MaxRequests caps how many token requests the capability token may make.
	// Zero means unlimited.
	MaxRequests int `json:"max_requests,omitempty"`
	// Label is an optional human-readable label recorded in audit logs.
	Label string `json:"label,omitempty"`
}

// IssueResponse is the response body for POST PathSessions.
type IssueResponse struct {
	// CapabilityToken is the freshly minted capability token.
	CapabilityToken string `json:"capability_token"`
	// ExpiresAt is the absolute expiration time of the capability token.
	ExpiresAt time.Time `json:"expires_at"`
}

// ErrorResponse is returned by the daemon on non-2xx responses.
type ErrorResponse struct {
	// Code is a machine-readable identifier (e.g. "unauthorized", "forbidden").
	Code string `json:"code"`
	// Message is a human-readable description.
	Message string `json:"message"`
}

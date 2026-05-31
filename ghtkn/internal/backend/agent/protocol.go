package agent

import "encoding/json"

// Command names and well-known response strings of the agent socket protocol.
// They must match the agent server (see the ghtkn repository, pkg/controller/agent).
const (
	commandGet   = "GET"
	commandSet   = "SET"
	respNotFound = "not found"
)

// request is a single request sent to the agent.
// The wire format is one JSON object per line (newline-delimited JSON).
type request struct {
	Command  string          `json:"command"`
	ClientID string          `json:"client_id,omitempty"`
	Token    json.RawMessage `json:"token,omitempty"`
}

// response is a single response returned by the agent.
type response struct {
	OK    bool            `json:"ok"`
	Token json.RawMessage `json:"token,omitempty"`
	Error string          `json:"error,omitempty"`
}

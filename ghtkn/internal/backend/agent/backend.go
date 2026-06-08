// Package agent provides a backend that stores and retrieves GitHub access tokens
// through a running ghtkn agent. The agent is a long-running process that holds a
// passphrase-derived key in memory and persists encrypted tokens, exposed over a
// Unix domain socket. This backend is the client side of that socket protocol and
// targets environments where the OS keyring is unavailable, such as containers and VMs.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// Backend stores and retrieves access tokens through a running ghtkn agent over a
// Unix domain socket.
type Backend struct {
	socket string
}

// New creates an agent backend. It resolves the socket path (GHTKN_AGENT_SOCKET, then
// the XDG-based default) but does not connect; a missing agent is reported on the
// first Get or Set.
func New(getEnv func(string) string) (*Backend, error) {
	socket, err := agentapi.SocketPath(getEnv, runtime.GOOS)
	if err != nil {
		return nil, err //nolint:wrapcheck // SocketPath returns a descriptive error
	}
	return &Backend{
		socket: socket,
	}, nil
}

// Get retrieves the raw token stored for clientID from the agent.
// It returns (nil, nil) when the agent has no token for the client ID, and
// agentapi.ErrAgentNotRunning when no agent is listening.
func (b *Backend) Get(ctx context.Context, clientID string) ([]byte, error) {
	resp, err := agentapi.Send(ctx, b.socket, &agentapi.Request{Command: agentapi.CommandGet, ClientID: clientID})
	if err != nil {
		return nil, err //nolint:wrapcheck // Send returns a descriptive error; callers may use agentapi.IsNotRunning
	}
	if !resp.OK {
		if resp.Error == agentapi.RespNotFound {
			return nil, nil
		}
		if resp.Error == agentapi.RespLocked {
			return nil, agentapi.ErrAgentLocked
		}
		return nil, fmt.Errorf("get an access token through the agent: %s", resp.Error)
	}
	if len(resp.Token) == 0 {
		return nil, nil
	}
	return []byte(resp.Token), nil
}

// Set stores the raw token for clientID in the agent.
// The token is already a JSON document, so it is sent verbatim as the request token.
func (b *Backend) Set(ctx context.Context, clientID, token string) error {
	resp, err := agentapi.Send(ctx, b.socket, &agentapi.Request{
		Command:  agentapi.CommandSet,
		ClientID: clientID,
		Token:    json.RawMessage(token),
	})
	if err != nil {
		return err //nolint:wrapcheck // Send returns a descriptive error
	}
	if !resp.OK {
		if resp.Error == agentapi.RespLocked {
			return agentapi.ErrAgentLocked
		}
		return fmt.Errorf("set an access token through the agent: %s", resp.Error)
	}
	return nil
}

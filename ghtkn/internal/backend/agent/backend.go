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
	"os"
	"runtime"
)

// Backend stores and retrieves access tokens through a running ghtkn agent over a
// Unix domain socket.
type Backend struct {
	socket string
}

// New creates an agent backend. It resolves the socket path (GHTKN_AGENT_SOCKET, then
// the XDG-based default) but does not connect; a missing agent is reported on the
// first Get or Set.
func New() (*Backend, error) {
	socket, err := socketPath(os.Getenv, runtime.GOOS)
	if err != nil {
		return nil, err
	}
	return &Backend{
		socket: socket,
	}, nil
}

// Get retrieves the raw token stored for clientID from the agent.
// It returns (nil, nil) when the agent has no token for the client ID, and
// errAgentNotRunning when no agent is listening.
func (b *Backend) Get(ctx context.Context, clientID string) ([]byte, error) {
	resp, err := roundTrip(ctx, b.socket, &request{Command: commandGet, ClientID: clientID})
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		if resp.Error == respNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("the ghtkn agent failed to get the token: %s", resp.Error)
	}
	if len(resp.Token) == 0 {
		return nil, nil
	}
	return []byte(resp.Token), nil
}

// Set stores the raw token for clientID in the agent.
// The token is already a JSON document, so it is sent verbatim as the request token.
func (b *Backend) Set(ctx context.Context, clientID, token string) error {
	resp, err := roundTrip(ctx, b.socket, &request{
		Command:  commandSet,
		ClientID: clientID,
		Token:    json.RawMessage(token),
	})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("the ghtkn agent failed to set the token: %s", resp.Error)
	}
	return nil
}

// Package agent provides a backend that stores and retrieves GitHub access tokens
// through a running ghtkn agent. The agent is a long-running process that holds a
// passphrase-derived key in memory and persists encrypted tokens, exposed over a
// Unix domain socket. This backend is the client side of that socket protocol and
// targets environments where the OS keyring is unavailable, such as containers and VMs.
package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
)

// Backend stores and retrieves access tokens through a running ghtkn agent over a
// Unix domain socket.
type Backend struct {
	socket string
	// warn is where security-relevant agent warnings are written. It defaults to
	// os.Stderr when nil; tests set it to capture the output.
	warn io.Writer
}

// warnWriter returns where agent warnings should be written, defaulting to os.Stderr.
func (b *Backend) warnWriter() io.Writer {
	if b.warn != nil {
		return b.warn
	}
	return os.Stderr
}

// New creates an agent backend. It resolves the socket path (GHTKN_AGENT_SOCKET, then
// the XDG-based default) but does not connect; a missing agent is reported on the
// first Get.
func New(getEnv func(string) string) (*Backend, error) {
	socket, err := agentapi.SocketPath(getEnv, runtime.GOOS)
	if err != nil {
		return nil, err //nolint:wrapcheck // SocketPath returns a descriptive error
	}
	return &Backend{
		socket: socket,
	}, nil
}

// Get probes the agent for a cached token for clientID with no freshness requirement.
// It exists to satisfy the storage backend interface; the token-lifecycle paths use
// GetActive/Begin/Poll/Revoke instead. It returns (nil, nil) when the agent has no
// token for the client ID, and agentapi.ErrAgentNotRunning when no agent is listening.
func (b *Backend) Get(ctx context.Context, clientID string) ([]byte, error) {
	return b.GetActive(ctx, clientID, 0)
}

// GetActive probes the agent for a token for clientID that is still valid for at least
// minExpiration. The agent checks expiration server-side, so a token expiring within
// minExpiration is reported as a miss. It is a pure read: it never starts a device
// flow. It returns (nil, nil) on a miss and agentapi.ErrAgentNotRunning when no agent
// is listening.
func (b *Backend) GetActive(ctx context.Context, clientID string, minExpiration time.Duration) ([]byte, error) {
	resp, err := b.get(ctx, clientID, &agentapi.Request{MinExpiration: minExpiration})
	if err != nil {
		return nil, err
	}
	if len(resp.Token) != 0 {
		return []byte(resp.Token), nil
	}
	return nil, nil
}

// get sends a single GET built from the given request options (StartDeviceFlow,
// AwaitDeviceFlow, MinExpiration); the command and client ID are filled in here.
func (b *Backend) get(ctx context.Context, clientID string, req *agentapi.Request) (*agentapi.Response, error) {
	req.Command = agentapi.CommandGet
	req.ClientID = clientID
	resp, err := agentapi.Send(ctx, b.socket, req)
	if err != nil {
		return nil, err //nolint:wrapcheck // Send returns a descriptive error; callers may use agentapi.IsNotRunning
	}
	// A security-relevant warning (e.g. a still-valid refresh token that failed to
	// refresh, suggesting it may have leaked) must reach the human, not just the agent
	// log which a background agent's operator may never see. Write it straight to
	// stderr so `ghtkn get` and the git credential helper both surface it.
	if resp.Warning != "" {
		// A failed warning write to stderr is not actionable; ignore it.
		_, _ = fmt.Fprintf(b.warnWriter(), "WARNING: ghtkn agent: %s\n", resp.Warning)
	}
	if !resp.OK {
		if resp.Error == agentapi.RespNotFound {
			return resp, nil
		}
		if resp.Error == agentapi.RespLocked {
			return nil, agentapi.ErrAgentLocked
		}
		return nil, fmt.Errorf("get an access token through the agent: %s", resp.Error)
	}
	return resp, nil
}

// Begin asks the agent to start (or join) the server-side device flow for clientID.
// The server first checks its store: if a token valid for minExpiration is already
// there (e.g. minted concurrently by another client), Begin returns it directly (as
// raw bytes) and no flow is started. Otherwise it returns the one-time code for the
// started flow, which the client displays before polling with Poll. Exactly one of
// the returned token and device code is non-nil.
func (b *Backend) Begin(ctx context.Context, clientID string, minExpiration time.Duration) ([]byte, *pubdeviceflow.DeviceCodeResponse, error) {
	resp, err := b.get(ctx, clientID, &agentapi.Request{StartDeviceFlow: true, MinExpiration: minExpiration})
	if err != nil {
		return nil, nil, err
	}
	if len(resp.Token) != 0 {
		return []byte(resp.Token), nil, nil
	}
	if !resp.Pending {
		return nil, nil, fmt.Errorf("the agent did not start the device flow for %s", clientID)
	}
	return nil, &pubdeviceflow.DeviceCodeResponse{
		UserCode:        resp.UserCode,
		VerificationURI: resp.VerificationURI,
		ExpiresIn:       resp.ExpiresIn,
	}, nil
}

// Poll waits for the agent to finish the server-side device flow for clientID and
// returns the raw token bytes it minted and cached. It polls with AwaitDeviceFlow set,
// so the agent returns the freshly minted token as is (no freshness check) once the
// flow completes, and reports Pending while it runs.
func (b *Backend) Poll(ctx context.Context, clientID string, minExpiration time.Duration) ([]byte, error) {
	// Probe immediately so a token that is already available is returned without
	// waiting for the first tick.
	if token, err := b.pollOnce(ctx, clientID, minExpiration); err != nil || token != nil {
		return token, err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for the device flow to complete: %w", ctx.Err())
		case <-ticker.C:
			token, err := b.pollOnce(ctx, clientID, minExpiration)
			if err != nil || token != nil {
				return token, err
			}
		}
	}
}

// pollOnce sends one GET marked AwaitDeviceFlow while waiting for the device flow to
// finish. It returns the token bytes when ready, (nil, nil) while the flow is still
// pending, and an error when the agent reports the flow ended without a token.
func (b *Backend) pollOnce(ctx context.Context, clientID string, minExpiration time.Duration) ([]byte, error) {
	resp, err := b.get(ctx, clientID, &agentapi.Request{AwaitDeviceFlow: true, MinExpiration: minExpiration})
	if err != nil {
		return nil, err
	}
	if len(resp.Token) != 0 {
		return []byte(resp.Token), nil
	}
	if resp.Pending {
		return nil, nil
	}
	return nil, fmt.Errorf("the agent's device flow for %s ended without a token", clientID)
}

// RevokeTokens asks the agent to revoke the tokens stored for clientIDs in one batch
// and delete them. It returns the client IDs whose credential could not be revoked
// (it may still be live) and those revoked but not deleted (a cleanup issue), so the
// caller can classify each. A non-nil error means the request itself failed (e.g. the
// agent is not running or locked), not that a particular token could not be revoked.
func (b *Backend) RevokeTokens(ctx context.Context, clientIDs []string) (revokeFailed, cleanupFailed []string, err error) {
	resp, err := agentapi.Send(ctx, b.socket, &agentapi.Request{Command: agentapi.CommandRevoke, ClientIDs: clientIDs})
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // Send returns a descriptive error; callers may use agentapi.IsNotRunning
	}
	if !resp.OK {
		if resp.Error == agentapi.RespLocked {
			return nil, nil, agentapi.ErrAgentLocked
		}
		return nil, nil, fmt.Errorf("revoke access tokens through the agent: %s", resp.Error)
	}
	return resp.RevokeFailed, resp.CleanupFailed, nil
}

// Set exists only to satisfy the storage backend interface. The agent mints and
// stores tokens itself as part of the server-side device flow, so a client never
// pushes a token to it; this method is never called on the agent path (device-flow
// creation returns changed=false) and always reports that pushing is unsupported.
func (b *Backend) Set(_ context.Context, _, _ string) error {
	return errors.New("the ghtkn agent stores tokens itself; pushing a token to it is not supported")
}

// Delete removes the token stored for clientID from the agent.
// It is a no-op when the agent has no token for the client ID, and returns
// agentapi.ErrAgentLocked when the agent is running but still locked.
func (b *Backend) Delete(ctx context.Context, clientID string) error {
	resp, err := agentapi.Send(ctx, b.socket, &agentapi.Request{Command: agentapi.CommandDelete, ClientID: clientID})
	if err != nil {
		return err //nolint:wrapcheck // Send returns a descriptive error; callers may use agentapi.IsNotRunning
	}
	if !resp.OK {
		if resp.Error == agentapi.RespNotFound {
			return nil
		}
		if resp.Error == agentapi.RespLocked {
			return agentapi.ErrAgentLocked
		}
		return fmt.Errorf("delete an access token through the agent: %s", resp.Error)
	}
	return nil
}

// Package backend stores and retrieves GitHub App access tokens through a
// pluggable backend. The concrete backend is selected by the GHTKN_BACKEND
// environment variable, allowing users to switch from the default OS keyring
// to alternatives such as the agent or the plaintext text backend.
package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/backend/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/backend/text"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// Backend stores and retrieves access tokens through a pluggable inner backend.
// It handles JSON (un)marshaling and validation so that inner backends only deal
// with raw bytes.
type Backend struct {
	backend backend
}

// backend is the interface implemented by concrete storage backends (keyring, text, ...).
// Get returns (nil, nil) when no token is stored for the given client ID.
// Delete removes the token stored for the client ID and is a no-op when none is stored.
type backend interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, string) error
	Delete(context.Context, string) error
}

// deviceFlowBackend is implemented by backends that own the token lifecycle
// server-side (the agent): they check expiration, run the device flow, and revoke
// tokens themselves. The api layer detects it via SupportsDeviceFlow and drives these
// operations through the wrapper methods instead of the client-side equivalents.
type deviceFlowBackend interface {
	GetActive(ctx context.Context, clientID string, minExpiration time.Duration) ([]byte, error)
	Begin(ctx context.Context, clientID string, minExpiration time.Duration) ([]byte, *pubdeviceflow.DeviceCodeResponse, error)
	Poll(ctx context.Context, clientID string, minExpiration time.Duration) ([]byte, error)
	RevokeTokens(ctx context.Context, clientIDs []string) (revokeFailed, cleanupFailed []string, err error)
}

// New creates a Backend based on the GHTKN_BACKEND environment variable.
// An empty value or "keyring" selects the OS keyring (the default); "agent" selects
// the ghtkn agent; "text" selects the plaintext file backend. Any other value
// returns an error. logger and slogLogger are only used by the agent backend to
// surface its warnings; the keyring and text backends ignore them.
func New(s string, getEnv func(string) string, logger *publog.Logger, slogLogger *slog.Logger) (*Backend, error) {
	switch s {
	case "agent":
		a, err := agent.New(getEnv, logger, slogLogger)
		if err != nil {
			return nil, err
		}
		return &Backend{
			backend: a,
		}, nil
	case "text":
		t, err := text.New(getEnv)
		if err != nil {
			return nil, err
		}
		return &Backend{
			backend: t,
		}, nil
	case "", "keyring":
		return &Backend{
			backend: keyring.New(&keyring.Input{
				ServiceKey: keyring.DefaultServiceKey,
			}),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported backend: %s", s)
	}
}

// Get retrieves and validates the access token stored for clientID.
// It returns (nil, nil) when no token is stored.
func (b *Backend) Get(ctx context.Context, clientID string) (*api.AccessToken, error) {
	bt, err := b.backend.Get(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("get a token from the backend: %w", err)
	}
	if bt == nil {
		return nil, nil
	}
	token := &api.AccessToken{}
	if err := json.Unmarshal(bt, token); err != nil {
		return nil, fmt.Errorf("unmarshal the token as JSON: %w", err)
	}
	if err := token.Validate(); err != nil {
		return nil, fmt.Errorf("the token in the backend is invalid: %w", err)
	}
	return token, nil
}

// SupportsDeviceFlow reports whether the inner backend owns the token lifecycle
// server-side (the agent). When true, the api layer drives expiration-aware reads,
// device-flow token creation, and revocation through the backend instead of the
// client-side equivalents.
func (b *Backend) SupportsDeviceFlow() bool {
	_, ok := b.backend.(deviceFlowBackend)
	return ok
}

// GetActive returns the token stored for clientID that is still valid for at least
// minExpiration, or nil when there is no such token. The freshness check runs
// server-side. It is only valid on a backend where SupportsDeviceFlow reports true.
func (b *Backend) GetActive(ctx context.Context, clientID string, minExpiration time.Duration) (*api.AccessToken, error) {
	df, ok := b.backend.(deviceFlowBackend)
	if !ok {
		return nil, errors.New("the backend does not check expiration itself")
	}
	bt, err := df.GetActive(ctx, clientID, minExpiration)
	if err != nil {
		return nil, fmt.Errorf("get an active token from the backend: %w", err)
	}
	return decodeToken(bt)
}

// BeginDeviceFlow asks the backend to start the server-side device flow for clientID.
// If a token valid for minExpiration already exists it is returned directly and the
// returned device code is nil; otherwise the token is nil and the device code carries
// the one-time code to display. Exactly one of the two is non-nil.
func (b *Backend) BeginDeviceFlow(ctx context.Context, clientID string, minExpiration time.Duration) (*api.AccessToken, *pubdeviceflow.DeviceCodeResponse, error) {
	df, ok := b.backend.(deviceFlowBackend)
	if !ok {
		return nil, nil, errors.New("the backend does not run the device flow itself")
	}
	bt, dc, err := df.Begin(ctx, clientID, minExpiration)
	if err != nil {
		return nil, nil, fmt.Errorf("begin the device flow through the backend: %w", err)
	}
	if bt != nil {
		token, err := decodeToken(bt)
		return token, nil, err
	}
	return nil, dc, nil
}

// PollDeviceFlow waits for the backend to finish the server-side device flow for
// clientID and returns the validated token it minted.
func (b *Backend) PollDeviceFlow(ctx context.Context, clientID string, minExpiration time.Duration) (*api.AccessToken, error) {
	df, ok := b.backend.(deviceFlowBackend)
	if !ok {
		return nil, errors.New("the backend does not run the device flow itself")
	}
	bt, err := df.Poll(ctx, clientID, minExpiration)
	if err != nil {
		return nil, fmt.Errorf("wait for the device flow through the backend: %w", err)
	}
	return decodeToken(bt)
}

// RevokeTokens asks the backend to revoke the tokens stored for clientIDs in one batch
// and delete them. It returns the client IDs whose credential could not be revoked (it
// may be live) and those revoked but not deleted (a cleanup issue), so the caller can
// classify each. A non-nil error means the request itself failed. It is only valid on
// a backend where SupportsDeviceFlow reports true.
func (b *Backend) RevokeTokens(ctx context.Context, clientIDs []string) (revokeFailed, cleanupFailed []string, err error) {
	df, ok := b.backend.(deviceFlowBackend)
	if !ok {
		return nil, nil, errors.New("the backend does not revoke tokens itself")
	}
	revokeFailed, cleanupFailed, err = df.RevokeTokens(ctx, clientIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("revoke tokens through the backend: %w", err)
	}
	return revokeFailed, cleanupFailed, nil
}

// decodeToken unmarshals and validates raw token bytes, returning (nil, nil) when the
// bytes are empty.
func decodeToken(bt []byte) (*api.AccessToken, error) {
	if len(bt) == 0 {
		return nil, nil
	}
	token := &api.AccessToken{}
	if err := json.Unmarshal(bt, token); err != nil {
		return nil, fmt.Errorf("unmarshal the token as JSON: %w", err)
	}
	if err := token.Validate(); err != nil {
		return nil, fmt.Errorf("the token from the backend is invalid: %w", err)
	}
	return token, nil
}

// Delete removes the token stored for clientID. It is a no-op when no token is stored.
func (b *Backend) Delete(ctx context.Context, clientID string) error {
	if err := b.backend.Delete(ctx, clientID); err != nil {
		return fmt.Errorf("delete a token from the backend: %w", err)
	}
	return nil
}

// Set marshals token to JSON and stores it for clientID.
func (b *Backend) Set(ctx context.Context, clientID string, token *api.AccessToken) error {
	bts, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal the token as JSON: %w", err)
	}
	if err := b.backend.Set(ctx, clientID, string(bts)); err != nil {
		return fmt.Errorf("set a token to the backend: %w", err)
	}
	return nil
}

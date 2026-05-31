// Package backend stores and retrieves GitHub App access tokens through a
// pluggable backend. The concrete backend is selected by the GHTKN_BACKEND
// environment variable, allowing users to switch from the default OS keyring
// to alternatives such as the agent or the plaintext text backend.
package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/backend/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/backend/text"
)

// Backend stores and retrieves access tokens through a pluggable inner backend.
// It handles JSON (un)marshaling and validation so that inner backends only deal
// with raw bytes.
type Backend struct {
	backend backend
}

// backend is the interface implemented by concrete storage backends (keyring, text, ...).
// Get returns (nil, nil) when no token is stored for the given client ID.
type backend interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, string) error
}

// New creates a Backend based on the GHTKN_BACKEND environment variable.
// An empty value or "keyring" selects the OS keyring (the default); "agent" selects
// the ghtkn agent; "text" selects the plaintext file backend. Any other value
// returns an error.
func New() (*Backend, error) {
	switch s := os.Getenv("GHTKN_BACKEND"); s {
	case "agent":
		a, err := agent.New()
		if err != nil {
			return nil, err
		}
		return &Backend{
			backend: a,
		}, nil
	case "text":
		t, err := text.New()
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

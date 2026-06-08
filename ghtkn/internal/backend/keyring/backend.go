// Package keyring provides secure storage for GitHub access tokens.
// It wraps the zalando/go-keyring library to store and retrieve tokens from the system keychain.
package keyring

import (
	"context"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

type Backend struct {
	get     func(service, key string) (string, error)
	set     func(service, key, token string) error
	service string
}

// New creates a new Keyring instance with the specified service name.
// The keyService parameter is used as the service identifier in the system keychain.
func New(input *Input) *Backend {
	return &Backend{
		get:     keyring.Get,
		set:     keyring.Set,
		service: input.ServiceKey,
	}
}

type Input struct {
	ServiceKey string
}

// Get retrieves the raw token stored for clientID.
// It returns (nil, nil) when no token is stored in the keyring.
func (b *Backend) Get(_ context.Context, clientID string) ([]byte, error) {
	s, err := b.get(b.service, clientID)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get a secret from the keyring: %w", err)
	}
	return []byte(s), nil
}

// Set stores the raw token for clientID in the keyring.
func (b *Backend) Set(_ context.Context, clientID string, token string) error {
	if err := b.set(b.service, clientID, token); err != nil {
		return fmt.Errorf("set a secret to the keyring: %w", err)
	}
	return nil
}

// DefaultServiceKey is the default service identifier used in the system keychain.
// This key is used to namespace tokens in the keyring to avoid conflicts with other applications.
const DefaultServiceKey = "github.com/suzuki-shunsuke/ghtkn"

package keyring

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// backend is the real implementation of the API interface.
// It wraps the zalando/go-keyring library to interact with the system keyring.
type backend struct{}

// newAPI creates a new backend instance for accessing the system keyring.
// This is the production implementation that uses the actual OS keyring service.
func newAPI() *backend {
	return &backend{}
}

// Get retrieves a password from the system keyring.
// It delegates to the zalando/go-keyring library's Get function.
func (b *backend) Get(service, user string) (string, bool, error) {
	v, err := keyring.Get(service, user)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("get a secret from the keyring: %w", err)
	}
	return v, true, nil
}

// Set stores a password in the system keyring.
// It delegates to the zalando/go-keyring library's Set function.
func (b *backend) Set(service, user, password string) error {
	return keyring.Set(service, user, password) //nolint:wrapcheck
}

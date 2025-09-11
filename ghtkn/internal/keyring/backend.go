package keyring

import (
	"errors"

	"github.com/zalando/go-keyring"
)

// Backend is the real implementation of the API interface.
// It wraps the zalando/go-keyring library to interact with the system keyring.
type Backend struct{}

// NewAPI creates a new Backend instance for accessing the system keyring.
// This is the production implementation that uses the actual OS keyring service.
func NewAPI() *Backend {
	return &Backend{}
}

// Get retrieves a password from the system keyring.
// If the key does not exist, it returns ("", false, nil).
// Otherwise, it returns (value, true, nil) on success or ("", false, error) on failure.
func (b *Backend) Get(service, key string) (string, bool, error) {
	v, err := keyring.Get(service, key)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return v, true, nil
}

// Set stores a password in the system keyring.
// It delegates to the zalando/go-keyring library's Set function.
func (b *Backend) Set(service, key, value string) error {
	return keyring.Set(service, key, value) //nolint:wrapcheck
}

// Delete removes a password from the system keyring.
// If the key does not exist, it returns (false, nil).
// Otherwise, it returns (true, nil) on success or (false, error) on failure.
func (b *Backend) Delete(service, key string) (bool, error) {
	if err := keyring.Delete(service, key); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Package keyring provides secure storage for GitHub access tokens.
// It wraps the zalando/go-keyring library to store and retrieve tokens from the system keychain.
package keyring

import (
	"encoding/json"
	"fmt"
	"time"
)

// Keyring manages access tokens in the system keychain.
// It provides methods to get, set, and remove tokens securely.
type Keyring struct {
	input *Input
}

// Input contains the dependencies required by the Keyring.
// It allows for dependency injection and easier testing by providing
// a customizable API implementation.
type Input struct {
	API API // The keyring API implementation for storing and retrieving secrets
}

// DefaultServiceKey is the default service identifier used in the system keychain.
// This key is used to namespace tokens in the keyring to avoid conflicts with other applications.
const DefaultServiceKey = "github.com/suzuki-shunsuke/ghtkn"

// New creates a new Keyring instance with the specified service name.
// The keyService parameter is used as the service identifier in the system keychain.
func New(input *Input) *Keyring {
	return &Keyring{
		input: input,
	}
}

// API defines the interface for keyring operations.
// It abstracts the underlying keyring implementation to allow for different backends
// (system keychain, testing mocks, etc.).
type API interface {
	Get(service, user string) (string, bool, error) // Retrieves a secret from the keyring
	Set(service, user, password string) error        // Stores a secret in the keyring
}

// NewInput creates a new Input instance with the default API implementation.
// This provides the standard system keyring integration for production use.
func NewInput() *Input {
	return &Input{
		API: NewAPI(),
	}
}

// dateFormat defines the standard format for date strings in the keyring.
const dateFormat = time.RFC3339

// ParseDate parses a date string in RFC3339 format.
// It returns a time.Time value or an error if the string cannot be parsed.
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse a date string: %w", err)
	}
	return t, nil
}

// FormatDate formats a time value as an RFC3339 string.
// This is the standard format used for expiration dates in the keyring.
func FormatDate(t time.Time) string {
	return t.Format(dateFormat)
}

// AccessToken represents a GitHub App access token stored in the keyring.
// It contains the token value, expiration information, and user login details.
type AccessToken struct {
	AccessToken    string    `json:"access_token"`    // The OAuth access token for GitHub API authentication
	ExpirationDate time.Time `json:"expiration_date"` // RFC3339 formatted expiration timestamp
	Login          string    `json:"login"`           // The GitHub user login associated with the token
	// ClientID string `json:"client_id"`
}

// Get retrieves an access token from the keyring.
// The key parameter identifies the token to retrieve.
// Returns the token or an error if the token cannot be found or unmarshaled.
func (kr *Keyring) Get(service string, key string) (*AccessToken, error) {
	s, exist, err := kr.input.API.Get(service, key)
	if err != nil {
		return nil, fmt.Errorf("get a GitHub Access token in keyring: %w", err)
	}
	if !exist {
		return nil, nil
	}
	token := &AccessToken{}
	if err := json.Unmarshal([]byte(s), token); err != nil {
		return nil, fmt.Errorf("unmarshal the token as JSON: %w", err)
	}
	return token, nil
}

// Set stores an access token in the keyring.
// The key parameter identifies where to store the token.
// Returns an error if the token cannot be marshaled or stored.
func (kr *Keyring) Set(service, key string, token *AccessToken) error {
	s, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal the token as JSON: %w", err)
	}
	if err := kr.input.API.Set(service, key, string(s)); err != nil {
		return fmt.Errorf("set a GitHub Access token in keyring: %w", err)
	}
	return nil
}

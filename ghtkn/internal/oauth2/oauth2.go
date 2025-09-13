// Package oauth2 provides OAuth2 token source implementation for GitHub access tokens.
// It implements the oauth2.TokenSource interface with caching and thread-safe access.
package oauth2

import (
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

// TokenSource implements oauth2.TokenSource interface for GitHub access tokens.
// It provides thread-safe caching of tokens and retrieves them from a client when needed.
type TokenSource struct {
	token  *oauth2.Token // Cached OAuth2 token
	mutex  *sync.Mutex   // Mutex for thread-safe access to the token
	client Client        // Client interface for retrieving access tokens
}

// Client defines the interface for retrieving GitHub access tokens.
// Implementations should return the raw access token string.
type Client interface {
	Get() (string, error) // Returns the access token string or an error
}

// NewTokenSource creates a new TokenSource with the given client.
// The token source will use the client to retrieve access tokens when needed.
func NewTokenSource(client Client) *TokenSource {
	return &TokenSource{
		mutex:  &sync.Mutex{},
		client: client,
	}
}

// Token implements oauth2.TokenSource.Token() interface.
// It returns a cached token if available, otherwise retrieves a new one from the client.
// The token retrieval is thread-safe and caches the result for subsequent calls.
func (ks *TokenSource) Token() (*oauth2.Token, error) {
	// Check if we have a cached token (read lock)
	ks.mutex.Lock()
	defer ks.mutex.Unlock()
	token := ks.token
	if token != nil {
		return token, nil
	}

	// Get new token from client
	s, err := ks.client.Get()
	if err != nil {
		return nil, fmt.Errorf("get a GitHub Access token from keyring: %w", err)
	}

	// Create OAuth2 token and cache it (write lock)
	token = &oauth2.Token{
		AccessToken: s,
	}
	ks.token = token
	return token, nil
}

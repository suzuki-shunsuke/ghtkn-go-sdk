package api

import (
	"context"
	"log/slog"
	"sync"
	"time"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	pubkeyring "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"golang.org/x/oauth2"
)

// TokenSource creates an OAuth2 token source for the token manager.
// It returns a token source that can be used with OAuth2 clients to automatically
// handle token retrieval and caching through the internal token management system.
func (tm *TokenManager) TokenSource(logger *slog.Logger, input *pubapi.InputGet) *TokenSource {
	return &TokenSource{
		mutex:  &sync.Mutex{},
		tm:     tm,
		logger: logger,
		input:  input,
		now:    time.Now,
	}
}

// TokenSource implements oauth2.TokenSource interface for GitHub access tokens.
// It provides thread-safe caching of tokens and retrieves them from a client when needed.
type TokenSource struct {
	token  *oauth2.Token     // Cached OAuth2 token
	mutex  *sync.Mutex       // Mutex for thread-safe access to the token
	tm     TokenSourceClient // Token manager instance for retrieving tokens
	logger *slog.Logger      // Logger for debugging and error reporting
	input  *pubapi.InputGet  // Input parameters for token retrieval
	now    func() time.Time
}

type TokenSourceClient interface {
	Get(ctx context.Context, logger *slog.Logger, input *pubapi.InputGet) (*pubkeyring.AccessToken, *pubconfig.App, error)
}

// Token implements oauth2.TokenSource.Token() interface.
// It returns a cached token if available, otherwise retrieves a new one from the client.
// The token retrieval is thread-safe and caches the result for subsequent calls.
func (ks *TokenSource) Token() (*oauth2.Token, error) {
	// Check if we have a cached token (read lock)
	ks.mutex.Lock()
	defer ks.mutex.Unlock()
	token := ks.token
	if token != nil && !isExpired(token, ks.now()) {
		return token, nil
	}

	// Get new token from client

	t, _, err := ks.tm.Get(context.Background(), ks.logger, ks.input)
	if err != nil {
		return nil, err
	}
	ks.token = &oauth2.Token{
		AccessToken: t.AccessToken,
		Expiry:      t.ExpirationDate,
	}
	return ks.token, nil
}

func isExpired(token *oauth2.Token, now time.Time) bool {
	// TODO consider min expiration
	return !token.Expiry.IsZero() && token.Expiry.Before(now)
}

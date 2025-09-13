package api

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/oauth2"
)

// tokenSourceClient implements the oauth2.Client interface for the token manager.
// It provides a bridge between the OAuth2 token source and the internal token management system.
type tokenSourceClient struct {
	tm     *TokenManager // Token manager instance for retrieving tokens
	logger *slog.Logger  // Logger for debugging and error reporting
	input  *InputGet     // Input parameters for token retrieval
}

// Get implements the oauth2.Client interface.
// It retrieves a GitHub access token using the token manager and returns the raw token string.
func (c *tokenSourceClient) Get() (string, error) {
	token, _, err := c.tm.Get(context.Background(), c.logger, c.input)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

// TokenSource creates an OAuth2 token source for the token manager.
// It returns a token source that can be used with OAuth2 clients to automatically
// handle token retrieval and caching through the internal token management system.
func (tm *TokenManager) TokenSource(logger *slog.Logger, input *InputGet) *oauth2.TokenSource {
	client := &tokenSourceClient{
		tm:     tm,
		logger: logger,
		input:  input,
	}
	return oauth2.NewTokenSource(client)
}

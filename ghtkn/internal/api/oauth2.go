package api

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/oauth2"
)

type tokenSourceClient struct {
	tm     *TokenManager
	logger *slog.Logger
	input  *InputGet
}

func (c *tokenSourceClient) Get() (string, error) {
	token, _, err := c.tm.Get(context.Background(), c.logger, c.input)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func (tm *TokenManager) TokenSource(logger *slog.Logger, input *InputGet) *oauth2.TokenSource {
	client := &tokenSourceClient{
		tm:     tm,
		logger: logger,
		input:  input,
	}
	return oauth2.NewTokenSource(logger, input.KeyringService, client)
}

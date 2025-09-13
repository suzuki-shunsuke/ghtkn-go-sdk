package oauth2

import (
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/oauth2"
)

type TokenSource struct {
	token  *oauth2.Token
	logger *slog.Logger
	mutex  *sync.RWMutex
	client Client
}

type Client interface {
	Get() (string, error)
}

func NewTokenSource(logger *slog.Logger, keyService string, client Client) *TokenSource {
	return &TokenSource{
		logger: logger,
		mutex:  &sync.RWMutex{},
		client: client,
	}
}

func (ks *TokenSource) Token() (*oauth2.Token, error) {
	ks.mutex.RLock()
	token := ks.token
	ks.mutex.RUnlock()
	if token != nil {
		return token, nil
	}
	ks.logger.Debug("getting a GitHub Access token from keyring")
	s, err := ks.client.Get()
	if err != nil {
		return nil, fmt.Errorf("get a GitHub Access token from keyring: %w", err)
	}
	ks.logger.Debug("got a GitHub Access token from keyring")
	token = &oauth2.Token{
		AccessToken: s,
	}
	ks.mutex.Lock()
	ks.token = token
	ks.mutex.Unlock()
	return token, nil
}

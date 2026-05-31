// Package ghtkn provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package ghtkn

import (
	"context"
	"log/slog"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"golang.org/x/oauth2"
)

// Client retrieves GitHub App access tokens.
// It wraps the internal token manager so that the public API is decoupled from
// internal implementation types.
type Client struct {
	tm *api.TokenManager
}

// New creates a new Client instance with default production dependencies.
func New() *Client {
	return &Client{
		tm: api.New(api.NewInput()),
	}
}

// Get retrieves a GitHub App access token, creating or renewing it when needed.
// It returns the access token and the resolved app configuration.
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*AccessToken, *AppConfig, error) {
	token, app, err := c.tm.Get(ctx, logger, toAPIInputGet(input))
	return fromKeyringAccessToken(token), fromConfigApp(app), err //nolint:wrapcheck
}

// TokenSource returns an oauth2.TokenSource that retrieves and caches access tokens
// through this client. It can be used with OAuth2-aware HTTP clients.
func (c *Client) TokenSource(logger *slog.Logger, input *InputGet) oauth2.TokenSource {
	return c.tm.TokenSource(logger, toAPIInputGet(input))
}

// SetLogger sets the hook functions invoked by the SDK to report notable events.
func (c *Client) SetLogger(logger *Logger) {
	c.tm.SetLogger(toLogLogger(logger))
}

// SetDeviceCodeUI sets the UI implementation used to display device flow information.
func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	c.tm.SetDeviceCodeUI(&deviceCodeUIAdapter{ui: ui})
}

// SetBrowser sets the implementation used to open the GitHub verification URL.
func (c *Client) SetBrowser(b Browser) {
	c.tm.SetBrowser(b)
}

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return config.GetPath(os.Getenv, runtime.GOOS)
}

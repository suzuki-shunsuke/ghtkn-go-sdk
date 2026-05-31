// Package ghtkn provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package ghtkn

import (
	"context"
	"log/slog"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/browser"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	internalapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	intconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"golang.org/x/oauth2"
)

type (
	// Public data/contract types live in their own public packages.
	AccessToken        = keyring.AccessToken
	AppConfig          = config.App
	Logger             = log.Logger
	DeviceCodeUI       = deviceflow.DeviceCodeUI
	Browser            = deviceflow.Browser
	DeviceCodeResponse = deviceflow.DeviceCodeResponse
	DefaultBrowser     = browser.Browser
	ClientIDReader     = api.PasswordReader
	InputGet           = api.InputGet
)

// Client retrieves GitHub App access tokens.
// It wraps the internal token manager so that the public API is decoupled from
// the internal implementation: changing the internal manager's method signatures
// causes a compile error here rather than silently changing the public API.
type Client struct {
	tm *internalapi.TokenManager
}

// New creates a new Client instance with default production dependencies.
func New() *Client {
	return &Client{
		tm: internalapi.New(internalapi.NewInput()),
	}
}

// Get retrieves a GitHub App access token, creating or renewing it when needed.
// It returns the access token and the resolved app configuration.
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*AccessToken, *AppConfig, error) {
	return c.tm.Get(ctx, logger, input)
}

// TokenSource returns an oauth2.TokenSource that retrieves and caches access tokens
// through this client. It can be used with OAuth2-aware HTTP clients.
func (c *Client) TokenSource(logger *slog.Logger, input *InputGet) oauth2.TokenSource {
	return c.tm.TokenSource(logger, input)
}

// SetLogger sets the hook functions invoked by the SDK to report notable events.
func (c *Client) SetLogger(logger *Logger) {
	c.tm.SetLogger(logger)
}

// SetDeviceCodeUI sets the UI implementation used to display device flow information.
func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	c.tm.SetDeviceCodeUI(ui)
}

// SetBrowser sets the implementation used to open the GitHub verification URL.
func (c *Client) SetBrowser(b Browser) {
	c.tm.SetBrowser(b)
}

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return intconfig.GetPath(os.Getenv, runtime.GOOS)
}

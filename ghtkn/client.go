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
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"golang.org/x/oauth2"
)

type (
	// Public data/contract types live in their own public packages.
	AccessToken        = api.AccessToken
	AppConfig          = config.App
	Logger             = log.Logger
	OnetimeCodeUI      = deviceflow.OnetimeCodeUI
	Browser            = deviceflow.Browser
	DeviceCodeResponse = deviceflow.DeviceCodeResponse
	DefaultBrowser     = browser.Browser
	InputGet           = api.InputGet
)

// ErrDisableDeviceFlow is returned by Get when the device flow is disabled via
// the GHTKN_DISABLE_DEVICE_FLOW environment variable. Detect it with errors.Is.
var ErrDisableDeviceFlow = api.ErrDisableDeviceFlow

// Client retrieves GitHub App access tokens.
// It wraps the internal token manager so that the public API is decoupled from
// the internal implementation: changing the internal manager's method signatures
// causes a compile error here rather than silently changing the public API.
type Client struct {
	tm *internalapi.TokenManager
}

// New creates a new Client instance with default production dependencies.
func New() (*Client, error) {
	input, err := internalapi.NewInput()
	if err != nil {
		return nil, err
	}
	return &Client{
		tm: internalapi.New(input),
	}, nil
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

// SetOnetimeCodeUI sets the UI implementation used to display the one-time code (user code) during the device flow.
func (c *Client) SetOnetimeCodeUI(ui OnetimeCodeUI) {
	c.tm.SetOnetimeCodeUI(ui)
}

// SetBrowser sets the implementation used to open the GitHub verification URL.
func (c *Client) SetBrowser(b Browser) {
	c.tm.SetBrowser(b)
}

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return intconfig.GetPath(os.Getenv, runtime.GOOS)
}

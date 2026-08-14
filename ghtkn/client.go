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
	InputShow          = deviceflow.InputShow
	DefaultBrowser     = browser.Browser
	InputGet           = api.InputGet
	InputAuth          = api.InputAuth
	InputRevoke        = api.InputRevoke
)

// ErrDisableDeviceFlow is returned by Get and TokenSource when a new token is
// needed. Creating one runs the device flow, which only Auth may do, so that a
// token is never created on the caller's behalf. Detect it with errors.Is.
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
	input, err := internalapi.NewInput(os.Getenv)
	if err != nil {
		return nil, err
	}
	return &Client{
		tm: internalapi.New(input),
	}, nil
}

// Get retrieves a GitHub App access token from the backend.
// It returns the access token and the resolved app configuration. It never creates a
// token: when no valid one is stored it fails with ErrDisableDeviceFlow, because
// creating one runs the interactive device flow and only Auth may do that. Call it
// freely from a background or non-interactive process; it never blocks on a user.
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*AccessToken, *AppConfig, error) {
	return c.tm.Get(ctx, logger, input)
}

// Auth authenticates to GitHub and stores a GitHub App access token in the backend,
// running the OAuth device flow when it needs to. It returns the access token and the
// resolved app configuration.
//
// It is the only operation that runs the device flow, so a token is only ever created
// by an explicit authentication and never on a caller's behalf. The device flow is
// interactive: it displays a one-time code and waits for the user to enter it, so only
// call this from a foreground, interactive context. It also always regenerates,
// ignoring any cached token, so that running it refreshes the token before it expires.
func (c *Client) Auth(ctx context.Context, logger *slog.Logger, input *InputAuth) (*AccessToken, *AppConfig, error) {
	return c.tm.Auth(ctx, logger, input)
}

// Revoke revokes GitHub credentials and removes the revoked tokens from the backend.
// The tokens to revoke are the tokens stored for input.AppNames. When AppNames is
// empty, it falls back to the app selected by GHTKN_APP (or the default app). It is
// a no-op when there is nothing to revoke.
func (c *Client) Revoke(ctx context.Context, logger *slog.Logger, input *InputRevoke) error {
	return c.tm.Revoke(ctx, logger, input)
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

// SetCopyOnetimeCodeToClipboard sets the implementation used to copy the one-time code to the clipboard.
func (c *Client) SetCopyOnetimeCodeToClipboard(f deviceflow.CopyTextToClipboard) {
	c.tm.SetCopyOnetimeCodeToClipboard(f)
}

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return intconfig.GetPath(os.Getenv, runtime.GOOS)
}

// Package ghtkn provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package ghtkn

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/browser"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/socket"
)

type (
	AccessToken        = keyring.AccessToken
	AppConfig          = config.App
	Config             = config.Config
	Logger             = log.Logger
	DeviceCodeUI       = deviceflow.DeviceCodeUI
	Browser            = deviceflow.Browser
	DeviceCodeResponse = deviceflow.DeviceCodeResponse
	DefaultBrowser     = browser.Browser
	ClientIDReader     = api.PasswordReader
	InputGet           = api.InputGet
)

// Client retrieves GitHub App access tokens. It runs in one of two modes:
//
//   - keyring mode (default): tokens are obtained via the OAuth device flow
//     and cached in the system keyring.
//   - socket mode: token retrieval is delegated to a ghtkn daemon over a Unix
//     socket. Activated when the GHTKN_SOCK environment variable is set.
//
// The mode is chosen once when New is called.
type Client struct {
	keyring  *api.TokenManager
	sockPath string
	capToken string
}

// New creates a new Client. If GHTKN_SOCK is set, the Client runs in socket
// mode and forwards token requests to the daemon at that path; otherwise it
// runs in keyring mode using the local OAuth device flow.
func New() *Client {
	if sock := os.Getenv(socket.EnvSock); sock != "" {
		return &Client{
			sockPath: sock,
			capToken: os.Getenv(socket.EnvCapToken),
		}
	}
	return &Client{keyring: api.New(api.NewInput())}
}

// Get retrieves an access token. In keyring mode it returns the cached token
// or runs the device flow. In socket mode it forwards the request to the
// daemon, returning the token and a minimal AppConfig containing only the
// app name (the daemon owns the full configuration).
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*AccessToken, *AppConfig, error) {
	if c.sockPath != "" {
		return c.getViaSocket(ctx, input)
	}
	return c.keyring.Get(ctx, logger, input)
}

func (c *Client) getViaSocket(ctx context.Context, input *InputGet) (*AccessToken, *AppConfig, error) {
	if input == nil {
		input = &InputGet{}
	}
	appName := input.AppName
	if appName == "" {
		appName = os.Getenv("GHTKN_APP")
	}
	resp, err := socket.FetchToken(ctx, c.sockPath, c.capToken, &socket.TokenRequest{App: appName})
	if err != nil {
		return nil, nil, fmt.Errorf("fetch token from ghtkn daemon: %w", err)
	}
	token := &AccessToken{
		AccessToken:    resp.AccessToken,
		ExpirationDate: resp.ExpirationDate,
		Login:          resp.Login,
	}
	return token, &AppConfig{Name: appName}, nil
}

// SetLogger updates the logger used by the keyring backend. It is a no-op in
// socket mode because logging happens daemon-side.
func (c *Client) SetLogger(logger *Logger) {
	if c.keyring != nil {
		c.keyring.SetLogger(logger)
	}
}

// SetDeviceCodeUI updates the device code UI used by the keyring backend. It
// is a no-op in socket mode because the device flow runs on the daemon host.
func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	if c.keyring != nil {
		c.keyring.SetDeviceCodeUI(ui)
	}
}

// SetBrowser updates the browser used by the keyring backend. It is a no-op
// in socket mode because the device flow runs on the daemon host.
func (c *Client) SetBrowser(b Browser) {
	if c.keyring != nil {
		c.keyring.SetBrowser(b)
	}
}

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return config.GetPath(os.Getenv, runtime.GOOS)
}

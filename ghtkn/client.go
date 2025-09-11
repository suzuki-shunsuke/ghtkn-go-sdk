// Package get provides functionality to retrieve GitHub App access tokens.
// It serves both the standard 'get' command and the 'git-credential' helper command.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package ghtkn

import (
	"context"
	"log/slog"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/browser"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// Client manages the process of retrieving GitHub App access tokens.
// It coordinates between configuration reading, token caching, and token generation.
type Client struct {
	tm tokenManager
}

// New creates a new Client instance with the provided input configuration.
func New() *Client {
	return &Client{
		tm: api.New(api.NewInput()),
	}
}

type tokenManager interface {
	Get(ctx context.Context, logger *slog.Logger, input *api.InputGet) (*keyring.AccessToken, *config.App, error)
	SetLogger(logger *log.Logger)
	SetDeviceCodeUI(ui deviceflow.DeviceCodeUI)
	SetBrowser(browser deviceflow.Browser)
	SetClientIDReader(reader ClientIDReader)
	SetApp(ctx context.Context, logger *slog.Logger, input *InputSetApp) error
}

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
)

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return config.GetPath(os.Getenv, runtime.GOOS)
}

// Package ghtkn provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package ghtkn

import (
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/browser"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	intconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// New creates a new Client instance with the provided input configuration.
func New() *Client {
	return api.New(api.NewInput())
}

type (
	// Public data/contract types live in their own public packages.
	AccessToken        = keyring.AccessToken
	AppConfig          = config.App
	Config             = config.Config
	Logger             = log.Logger
	DeviceCodeUI       = deviceflow.DeviceCodeUI
	Browser            = deviceflow.Browser
	DeviceCodeResponse = deviceflow.DeviceCodeResponse
	DefaultBrowser     = browser.Browser
	// The token manager and its inputs remain internal implementation.
	ClientIDReader = api.PasswordReader
	InputGet       = api.InputGet
	Client         = api.TokenManager
)

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return intconfig.GetPath(os.Getenv, runtime.GOOS)
}

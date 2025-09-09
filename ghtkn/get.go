package ghtkn

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

type InputGet struct {
	ClientID       string
	UseKeyring     bool
	KeyringService string
	UseConfig      bool
	AppName        string
	ConfigFilePath string
	MinExpiration  time.Duration
}

type (
	AccessToken  = keyring.AccessToken
	AppConfig    = config.App
	Config       = config.Config
	Logger       = log.Logger
	DeviceCodeUI = deviceflow.DeviceCodeUI
	Browser      = deviceflow.Browser
)

// GetConfigPath returns the default configuration file path for ghtkn.
func GetConfigPath() (string, error) {
	return config.GetPath(os.Getenv, runtime.GOOS)
}

// Get executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary.
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*AccessToken, *AppConfig, error) {
	if input == nil {
		input = &InputGet{}
	}
	return c.tm.Get(ctx, logger, &api.InputGet{
		ClientID:       input.ClientID,
		UseKeyring:     input.UseKeyring,
		KeyringService: input.KeyringService,
		UseConfig:      input.UseConfig,
		AppName:        input.AppName,
		ConfigFilePath: input.ConfigFilePath,
		MinExpiration:  input.MinExpiration,
	})
}

func (c *Client) SetLogger(logger *Logger) {
	c.tm.SetLogger(logger)
}

func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	c.tm.SetDeviceCodeUI(ui)
}

func (c *Client) SetBrowser(ui Browser) {
	c.tm.SetBrowser(ui)
}

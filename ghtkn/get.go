package ghtkn

import (
	"context"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
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

// Get executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary.
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*keyring.AccessToken, *config.App, error) {
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

func (c *Client) SetLogger(logger *log.Logger) {
	c.tm.SetLogger(logger)
}

func (c *Client) SetDeviceCodeUI(ui deviceflow.DeviceCodeUI) {
	c.tm.SetDeviceCodeUI(ui)
}

package ghtkn

import (
	"context"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

type InputGet struct {
	KeyringService string
	AppName        string
	ConfigFilePath string
	User           string
	MinExpiration  time.Duration
}

// Get executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary.
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*AccessToken, *AppConfig, error) {
	if input == nil {
		input = &InputGet{}
	}
	i := &api.InputGet{
		KeyringService: input.KeyringService,
		AppName:        input.AppName,
		ConfigFilePath: input.ConfigFilePath,
		MinExpiration:  input.MinExpiration,
		User:           input.User,
	}
	return c.tm.Get(ctx, logger, i)
}

func (c *Client) SetLogger(logger *Logger) {
	log.InitLogger(logger)
	c.tm.SetLogger(logger)
}

func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	c.tm.SetDeviceCodeUI(ui)
}

func (c *Client) SetBrowser(ui Browser) {
	c.tm.SetBrowser(ui)
}

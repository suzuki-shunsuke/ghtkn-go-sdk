package ghtkn

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// Get executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary.
func (c *Client) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*AccessToken, *AppConfig, error) {
	if input == nil {
		input = &InputGet{}
	}
	return c.tm.Get(ctx, logger, input)
}

// SetLogger updates the logger instance used by the client.
// It initializes any nil logging functions with defaults and propagates the logger
// to the underlying token manager.
func (c *Client) SetLogger(logger *Logger) {
	log.InitLogger(logger)
	c.tm.SetLogger(logger)
}

// SetDeviceCodeUI updates the device code UI implementation used during OAuth device flow.
// This allows customization of how device flow information is presented to users.
func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	c.tm.SetDeviceCodeUI(ui)
}

// SetBrowser updates the browser implementation used to open verification URLs.
// This allows customization of how the GitHub verification page is opened during device flow.
func (c *Client) SetBrowser(ui Browser) {
	c.tm.SetBrowser(ui)
}

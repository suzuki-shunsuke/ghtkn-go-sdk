package deviceflow

import (
	"context"
	"log/slog"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow/ui"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// Client handles GitHub App authentication and access token generation using OAuth device flow.
// It manages the complete authentication flow including device code requests, user authorization,
// and access token polling.
type Client struct {
	input *Input // Configuration and dependencies for the client
}

// NewClient creates a new Client with the provided HTTP client.
// The client uses the provided HTTP client for all API requests.
func NewClient(input *Input) *Client {
	return &Client{
		input: input,
	}
}

// Show displays the one-time code (and opens the browser / copies to the clipboard)
// for a device flow whose device code was obtained elsewhere, such as by the agent
// server. It reuses the same UI the client-side device flow uses, so the presentation
// is identical regardless of which side minted the token. input carries the display
// options (AppName, SkipAccountPicker, OpenBrowser, Clipboard) and deviceCode carries
// the one-time code, verification URL, and expiry.
func (c *Client) Show(ctx context.Context, logger *slog.Logger, input *InputCreate, deviceCode *pubdeviceflow.DeviceCodeResponse) error {
	return c.input.OnetimeCodeUI.Show(ctx, logger, &ui.InputCreate{ //nolint:wrapcheck
		ClientID:          input.ClientID,
		AppName:           input.AppName,
		SkipAccountPicker: input.SkipAccountPicker,
		OpenBrowser:       input.OpenBrowser,
		Clipboard:         input.Clipboard,
	}, deviceCode)
}

// SetLogger updates the logger instance used by the client.
// This allows dynamic reconfiguration of logging behavior.
func (c *Client) SetLogger(logger *publog.Logger) {
	c.input.Logger = logger
}

// SetOnetimeCodeUI updates the one-time code UI implementation used by the client.
// This allows customization of how the one-time code (user code) is presented to users.
func (c *Client) SetOnetimeCodeUI(o pubdeviceflow.OnetimeCodeUI) {
	c.input.OnetimeCodeUI.SetOnetimeCodeUI(o)
}

// SetBrowser updates the browser implementation used by the client.
// This allows customization of how verification URLs are opened in the browser.
func (c *Client) SetBrowser(b pubdeviceflow.Browser) {
	c.input.OnetimeCodeUI.SetBrowser(b)
}

// SetCopyOnetimeCodeToClipboard updates the clipboard implementation used to copy the one-time code.
// This allows customization of how the one-time code is copied to the user's clipboard.
func (c *Client) SetCopyOnetimeCodeToClipboard(f pubdeviceflow.CopyTextToClipboard) {
	c.input.OnetimeCodeUI.SetCopyOnetimeCodeToClipboard(f)
}

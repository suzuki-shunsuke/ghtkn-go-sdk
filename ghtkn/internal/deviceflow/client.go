package deviceflow

import (
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
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

// SetLogger updates the logger instance used by the client.
// This allows dynamic reconfiguration of logging behavior.
func (c *Client) SetLogger(logger *publog.Logger) {
	c.input.Logger = logger
}

// SetOnetimeCodeUI updates the one-time code UI implementation used by the client.
// This allows customization of how the one-time code (user code) is presented to users.
func (c *Client) SetOnetimeCodeUI(ui pubdeviceflow.OnetimeCodeUI) {
	c.input.OnetimeCodeUI = ui
}

// SetBrowser updates the browser implementation used by the client.
// This allows customization of how verification URLs are opened in the browser.
func (c *Client) SetBrowser(b pubdeviceflow.Browser) {
	c.input.Browser = b
}

// SetCopyOnetimeCodeToClipboard updates the clipboard implementation used to copy the one-time code.
// This allows customization of how the one-time code is copied to the user's clipboard.
func (c *Client) SetCopyOnetimeCodeToClipboard(f pubdeviceflow.CopyTextToClipboard) {
	c.input.CopyOnetimeCodeToClipboard = f
}

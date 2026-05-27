package deviceflow

import "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"

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
func (c *Client) SetLogger(logger *log.Logger) {
	c.input.Logger = logger
}

// SetDeviceCodeUI updates the device code UI implementation used by the client.
// This allows customization of how device flow information is presented to users.
func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	c.input.DeviceCodeUI = ui
}

// SetBrowser updates the browser implementation used by the client.
// This allows customization of how verification URLs are opened in the browser.
func (c *Client) SetBrowser(b Browser) {
	c.input.Browser = b
}

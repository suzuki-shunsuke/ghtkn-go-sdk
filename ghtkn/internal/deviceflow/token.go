// Package deviceflow handles GitHub App access token generation using OAuth device flow.
// It provides functionality to authenticate GitHub Apps and obtain access tokens.
package deviceflow

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/browser"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// Client handles GitHub App authentication and access token generation using OAuth device flow.
// It manages the complete authentication flow including device code requests, user authorization,
// and access token polling.
type Client struct {
	input *Input // Configuration and dependencies for the client
}

// Browser provides an interface for opening URLs in a web browser.
// This is used to open the GitHub verification URL during device flow authentication.
type Browser interface {
	Open(ctx context.Context, logger *slog.Logger, url string) error
}

// Input contains all dependencies and configuration needed by the Client.
// It allows for dependency injection and makes testing easier by providing
// customizable implementations of external dependencies.
type Input struct {
	HTTPClient   *http.Client                       // HTTP client for API requests
	Now          func() time.Time                   // Function to get current time (for testing)
	Stderr       io.Writer                          // Writer for error output
	Browser      Browser                            // Interface for opening URLs in browser
	NewTicker    func(d time.Duration) *time.Ticker // Function to create tickers (for testing)
	Logger       *log.Logger                        // Logger for debugging and info messages
	DeviceCodeUI DeviceCodeUI                       // UI for displaying device flow information
}

// SetLogger updates the logger instance used by the client.
// This allows dynamic reconfiguration of logging behavior.
func (c *Client) SetLogger(logger *log.Logger) {
	c.input.Logger = logger
}

// NewInput creates a new Input instance with default dependencies.
// This provides sensible defaults for production use, including the default HTTP client,
// system stderr, real browser integration, and standard time functions.
func NewInput() *Input {
	return &Input{
		HTTPClient:   http.DefaultClient,
		Now:          time.Now,
		Stderr:       os.Stderr,
		Browser:      &browser.Browser{},
		NewTicker:    time.NewTicker,
		Logger:       log.NewLogger(),
		DeviceCodeUI: NewDeviceCodeUI(os.Stdin, os.Stderr, &SimpleWaiter{}),
	}
}

// NewClient creates a new Client with the provided HTTP client.
// The client uses the provided HTTP client for all API requests.
func NewClient(input *Input) *Client {
	return &Client{
		input: input,
	}
}

// DeviceCodeResponse represents the response from GitHub's device code endpoint.
// It contains the device code and user code needed for authentication.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// AccessTokenResponse represents the response from GitHub's access token endpoint.
// It contains either an access token or an error message.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`

	Error string `json:"error"`
}

// AccessToken represents a GitHub App access token with its metadata.
// It includes the token value, associated app, and expiration date.
type AccessToken struct {
	App            string    `json:"app"`
	AccessToken    string    `json:"access_token"`
	ExpirationDate time.Time `json:"expiration_date"`
}

// Package apptoken handles GitHub App access token generation using OAuth device flow.
// It provides functionality to authenticate GitHub Apps and obtain access tokens.
package apptoken

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// Client handles GitHub App authentication and access token generation.
type Client struct {
	input *Input
}

type Browser interface {
	Open(ctx context.Context, url string) error
}

type Input struct {
	HTTPClient   *http.Client
	Now          func() time.Time
	Stderr       io.Writer
	Browser      Browser
	NewTicker    func(d time.Duration) *time.Ticker
	Logger       *log.Logger
	DeviceCodeUI DeviceCodeUI
}

func (c *Client) SetLogger(logger *log.Logger) {
	c.input.Logger = logger
}

func NewInput() *Input {
	return &Input{
		HTTPClient:   http.DefaultClient,
		Now:          time.Now,
		Stderr:       os.Stderr,
		Browser:      NewBrowser(),
		NewTicker:    time.NewTicker,
		Logger:       log.NewLogger(),
		DeviceCodeUI: NewDeviceCodeUI(os.Stderr),
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

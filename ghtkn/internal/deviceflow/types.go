// Package deviceflow handles GitHub App access token generation using OAuth device flow.
// It provides functionality to authenticate GitHub Apps and obtain access tokens.
// The public contract types (OnetimeCodeUI, Browser, DeviceCodeResponse) live in the
// public ghtkn/deviceflow package.
package deviceflow

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/browser"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// Input contains all dependencies and configuration needed by the Client.
// It allows for dependency injection and makes testing easier by providing
// customizable implementations of external dependencies.
type Input struct {
	HTTPClient                 *http.Client                       // HTTP client for API requests
	Now                        func() time.Time                   // Function to get current time (for testing)
	Stderr                     io.Writer                          // Writer for error output
	Browser                    pubdeviceflow.Browser              // Interface for opening URLs in browser
	NewTicker                  func(d time.Duration) *time.Ticker // Function to create tickers (for testing)
	Logger                     *publog.Logger                     // Logger for debugging and info messages
	OnetimeCodeUI              pubdeviceflow.OnetimeCodeUI        // UI for displaying the one-time code (user code)
	CopyOnetimeCodeToClipboard pubdeviceflow.CopyTextToClipboard  // Function to copy one-time code to clipboard
}

// NewInput creates a new Input instance with default dependencies.
// This provides sensible defaults for production use, including the default HTTP client,
// system stderr, real browser integration, and standard time functions.
func NewInput() *Input {
	return &Input{
		HTTPClient:    http.DefaultClient,
		Now:           time.Now,
		Stderr:        os.Stderr,
		Browser:       &browser.Browser{},
		NewTicker:     time.NewTicker,
		Logger:        log.NewLogger(),
		OnetimeCodeUI: newOnetimeCodeUI(os.Stdin, os.Stderr, &simpleWaiter{}),
	}
}

// accessTokenResponse represents the response from GitHub's access token endpoint.
// It contains either an access token or an error message.
type accessTokenResponse struct {
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

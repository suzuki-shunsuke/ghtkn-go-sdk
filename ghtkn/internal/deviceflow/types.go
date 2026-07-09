// Package deviceflow handles GitHub App access token generation using OAuth device flow.
// It provides functionality to authenticate GitHub Apps and obtain access tokens.
// The public contract types (OnetimeCodeUI, Browser, DeviceCodeResponse) live in the
// public ghtkn/deviceflow package.
package deviceflow

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/browser"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
)

// Input contains all dependencies and configuration needed by the Client.
// It allows for dependency injection and makes testing easier by providing
// customizable implementations of external dependencies.
type Input struct {
	Now                        func() time.Time                  // Function to get current time (for testing)
	Stderr                     io.Writer                         // Writer for error output
	Browser                    pubdeviceflow.Browser             // Interface for opening URLs in browser
	Logger                     *publog.Logger                    // Logger for debugging and info messages
	OnetimeCodeUI              pubdeviceflow.OnetimeCodeUI       // UI for displaying the one-time code (user code)
	CopyOnetimeCodeToClipboard pubdeviceflow.CopyTextToClipboard // Function to copy one-time code to clipboard
	Client                     DeviceFlow                        // Device flow API client (wraps the go-github-device-flow library)
}

// DeviceFlow talks to GitHub's device flow endpoints. GetDeviceCode returns the
// SDK's own DeviceCodeResponse because it flows out to OnetimeCodeUI in the public
// API; the access token stays internal, so Poll returns the library type directly.
// The production implementation is libDeviceFlow; tests inject a fake.
type DeviceFlow interface {
	GetDeviceCode(ctx context.Context, clientID string) (*pubdeviceflow.DeviceCodeResponse, error)
	Poll(ctx context.Context, logger *slog.Logger, clientID string, deviceCode *pubdeviceflow.DeviceCodeResponse) (*deviceflow.AccessToken, error)
}

// NewInput creates a new Input instance with default dependencies.
// This provides sensible defaults for production use, including the default HTTP client,
// system stderr, real browser integration, and standard time functions.
func NewInput() *Input {
	return &Input{
		Now:           time.Now,
		Stderr:        os.Stderr,
		Browser:       &browser.Browser{},
		Logger:        log.NewLogger(),
		OnetimeCodeUI: newOnetimeCodeUI(os.Stdin, os.Stderr, &simpleWaiter{}),
		Client:        newLibDeviceFlow(http.DefaultClient, time.Now, time.NewTicker),
	}
}

// AccessToken represents a GitHub App access token with its metadata.
// It includes the token value, associated app, and expiration date.
type AccessToken struct {
	App            string    `json:"app"`
	AccessToken    string    `json:"access_token"`
	ExpirationDate time.Time `json:"expiration_date"`
}

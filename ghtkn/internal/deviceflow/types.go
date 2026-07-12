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
	"time"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow/ui"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
)

// Input contains all dependencies and configuration needed by the Client.
// It allows for dependency injection and makes testing easier by providing
// customizable implementations of external dependencies.
type Input struct {
	Stderr        io.Writer      // Writer for error output
	Logger        *publog.Logger // Logger for debugging and info messages
	OnetimeCodeUI OnetimeCodeUI  // UI for displaying the one-time code (user code)
	Client        DeviceFlow     // Device flow API client (wraps the go-github-device-flow library)
}

type OnetimeCodeUI interface {
	Show(ctx context.Context, logger *slog.Logger, input *ui.InputCreate, deviceCode *pubdeviceflow.DeviceCodeResponse) error
	SetBrowser(b pubdeviceflow.Browser)
	SetOnetimeCodeUI(o pubdeviceflow.OnetimeCodeUI)
	SetCopyOnetimeCodeToClipboard(f pubdeviceflow.CopyTextToClipboard)
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
		Logger:        log.NewLogger(),
		OnetimeCodeUI: ui.New(nil),
		Client:        newLibDeviceFlow(http.DefaultClient),
	}
}

// AccessToken represents a GitHub App access token with its metadata.
// It includes the token value, associated app, and expiration date.
type AccessToken struct {
	App            string    `json:"app"`
	AccessToken    string    `json:"access_token"`
	ExpirationDate time.Time `json:"expiration_date"`
}

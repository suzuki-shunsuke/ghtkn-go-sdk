package ghtkn

import (
	"context"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/browser"
)

// AccessToken represents a GitHub App access token returned by the SDK.
// It contains the token value, its expiration date, and the associated GitHub user login.
type AccessToken struct {
	AccessToken    string
	ExpirationDate time.Time
	Login          string
}

// AppConfig represents a GitHub App configuration.
// Each app must have a unique name and a client ID for authentication.
type AppConfig struct {
	Name     string `json:"name"`
	ClientID string `json:"client_id" yaml:"client_id"`
	GitOwner string `json:"git_owner,omitempty" yaml:"git_owner"`
}

// Config represents the main configuration structure for ghtkn.
// It contains a list of GitHub Apps.
type Config struct {
	Apps []*AppConfig `json:"apps"`
}

// InputGet contains the input parameters for token retrieval operations.
// It provides configuration options for specifying which app to use,
// where to find configuration, and token expiration requirements.
type InputGet struct {
	KeyringService string        // Service name for keyring storage (defaults to the SDK's default)
	AppName        string        // Name of the app to use (defaults to GHTKN_APP environment variable)
	ConfigFilePath string        // Path to configuration file (auto-detected if empty)
	AppOwner       string        // GitHub App Owner
	MinExpiration  time.Duration // Minimum time before token expiration to trigger renewal
}

// Logger holds hook functions invoked by the SDK to report notable events.
// Each field is optional; nil functions are not called.
type Logger struct {
	// Expire logs when an access token expiration date is processed.
	Expire func(logger *slog.Logger, exDate time.Time)
	// FailedToOpenBrowser logs when the browser cannot be opened for authentication.
	FailedToOpenBrowser func(logger *slog.Logger, err error)
	// FailedToGetAccessTokenFromKeyring logs when access token retrieval from keyring fails.
	FailedToGetAccessTokenFromKeyring func(logger *slog.Logger, err error)
	// AccessTokenIsNotFoundInKeyring logs when no access token is found in the keyring.
	AccessTokenIsNotFoundInKeyring func(logger *slog.Logger)
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

// DeviceCodeUI provides an interface for displaying device flow information to users.
type DeviceCodeUI interface {
	Show(ctx context.Context, logger *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error
}

// Browser provides an interface for opening URLs in a web browser.
// This is used to open the GitHub verification URL during device flow authentication.
type Browser interface {
	Open(ctx context.Context, logger *slog.Logger, url string) error
}

// ClientIDReader provides an interface for reading a client ID for an app.
type ClientIDReader interface {
	Read(ctx context.Context, logger *slog.Logger, app *AppConfig) (string, error)
}

// DefaultBrowser is the default Browser implementation that opens URLs using
// the operating system's standard browser-opening command.
type DefaultBrowser struct{}

// Open opens the given URL in the operating system's default web browser.
func (b *DefaultBrowser) Open(ctx context.Context, logger *slog.Logger, url string) error {
	return (&browser.Browser{}).Open(ctx, logger, url) //nolint:wrapcheck
}

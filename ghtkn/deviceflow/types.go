// Package deviceflow provides the public types for GitHub App OAuth device flow,
// used to display the one-time code (user code) and open verification URLs.
package deviceflow

import (
	"context"
	"log/slog"
	"time"
)

// OnetimeCodeUI provides an interface for displaying the one-time code (user code)
// and verification URL to users during the device flow.
type OnetimeCodeUI interface {
	Show(ctx context.Context, logger *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error
}

// Browser provides an interface for opening URLs in a web browser.
// This is used to open the GitHub verification URL during device flow authentication.
type Browser interface {
	Open(ctx context.Context, logger *slog.Logger, url string) error
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

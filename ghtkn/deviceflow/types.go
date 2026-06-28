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
	Show(ctx context.Context, logger *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time, input *InputShow) error
}

// InputShow carries the optional parameters for OnetimeCodeUI.Show. It is a struct
// (rather than positional arguments) so new fields can be added later without
// breaking implementations.
type InputShow struct {
	// OpenBrowser reports whether the browser will be opened automatically
	// afterwards. When false, the UI should ask the user to open the URL
	// themselves instead.
	OpenBrowser bool
	// AppName is the GitHub App name shown alongside the one-time code. It is
	// optional; when empty, the UI omits the App Name line from the message.
	AppName string
	// CopiedToClipboard reports whether the one-time code was successfully copied
	// to the system clipboard. When true, the UI shows a line telling the user the
	// code is already on their clipboard.
	CopiedToClipboard bool
}

// CopyTextToClipboard copies the given text (the device flow one-time code) to the
// system clipboard. The SDK does not provide an implementation: consumers inject one
// via Client.SetCopyOnetimeCodeToClipboard so the clipboard dependency stays out of
// the SDK module. It returns an error if the copy fails; the caller logs a warning
// and continues so authentication still succeeds.
type CopyTextToClipboard func(ctx context.Context, code string) error

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

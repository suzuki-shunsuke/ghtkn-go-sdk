package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/browser"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// availabilityChecker is an optional interface a Browser may implement to report
// whether it can actually open a browser on this host. When the configured Browser
// implements it and reports false, the verification URL is shown for the user to
// open manually instead of attempting (and failing) an automatic open.
type availabilityChecker interface {
	Available() bool
}

type Client struct {
	input *Input // Configuration and dependencies for the client
}

func New(input *Input) *Client {
	if input == nil {
		input = &Input{}
	}
	if input.Now == nil {
		input.Now = time.Now
	}
	if input.Stderr == nil {
		input.Stderr = os.Stderr
	}
	if input.Browser == nil {
		input.Browser = &browser.Browser{}
	}
	if input.Logger == nil {
		input.Logger = log.NewLogger()
	}
	if input.OnetimeCodeUI == nil {
		input.OnetimeCodeUI = newOnetimeCodeUI(os.Stdin, os.Stderr, &simpleWaiter{})
	}
	return &Client{input: input}
}

type Input struct {
	Now                        func() time.Time                  // Function to get current time (for testing)
	Stderr                     io.Writer                         // Writer for error output
	Browser                    pubdeviceflow.Browser             // Interface for opening URLs in browser
	Logger                     *publog.Logger                    // Logger for debugging and info messages
	OnetimeCodeUI              pubdeviceflow.OnetimeCodeUI       // UI for displaying the one-time code (user code)
	CopyOnetimeCodeToClipboard pubdeviceflow.CopyTextToClipboard // Function to copy one-time code to clipboard
}

// SetBrowser updates the browser implementation used by the client.
// This allows customization of how verification URLs are opened in the browser.
func (c *Client) SetBrowser(b pubdeviceflow.Browser) {
	c.input.Browser = b
}

// SetOnetimeCodeUI updates the low-level renderer that prints the one-time code and
// waits for the user. It lets an SDK consumer replace the default terminal UI.
func (c *Client) SetOnetimeCodeUI(o pubdeviceflow.OnetimeCodeUI) {
	c.input.OnetimeCodeUI = o
}

// SetCopyOnetimeCodeToClipboard sets the function used to copy the one-time code to
// the clipboard. Without it the clipboard step is skipped even when it is enabled.
func (c *Client) SetCopyOnetimeCodeToClipboard(f pubdeviceflow.CopyTextToClipboard) {
	c.input.CopyOnetimeCodeToClipboard = f
}

// InputCreate holds the parameters for Create.
type InputCreate struct {
	// ClientID is the GitHub App's client ID used to start the device flow. Required.
	ClientID string
	// AppName is the GitHub App name shown in the one-time code prompt. Optional;
	// when empty, the App Name line is omitted from the message.
	AppName string
	// SkipAccountPicker appends GitHub's unofficial skip_account_picker query
	// parameter to the verification URL.
	SkipAccountPicker bool
	// OpenBrowser controls whether the verification URL is opened in a browser
	// automatically. When false, the URL is only shown for the user to open
	// manually. Even when true, the browser is opened only if one is available.
	OpenBrowser bool
	// Clipboard controls whether the one-time code is copied to the system
	// clipboard. The copy also requires a clipboard implementation to have been
	// injected via Client.SetCopyOnetimeCodeToClipboard.
	Clipboard bool
}

func (c *Client) Show(ctx context.Context, logger *slog.Logger, input *InputCreate, deviceCode *pubdeviceflow.DeviceCodeResponse) error {
	if input.SkipAccountPicker {
		verificationURI, err := appendSkipAccountPickerParam(deviceCode.VerificationURI)
		if err != nil {
			return fmt.Errorf("add skip_account_picker to the verification URL: %w", err)
		}
		// Update the verification URL in place so both the displayed prompt (shown
		// by OnetimeCodeUI.Show) and the browser open below use the skip variant.
		// Otherwise a user opening the URL manually (browser disabled/unavailable,
		// e.g. the git credential helper or a headless host) would still hit the
		// account picker that skip_account_picker is meant to bypass.
		deviceCode.VerificationURI = verificationURI
	}

	// Decide up front whether the browser will actually be opened, so the UI can
	// show the right instruction and we only attempt an open that can succeed.
	willOpen := c.isOpenBrowser(input)

	// Copy the one-time code to the clipboard before showing it, so the UI can tell
	// the user it is already on their clipboard. A copy failure must not abort the
	// device flow: warn and continue so the user can still copy the code manually.
	copied := false
	if input.Clipboard && c.input.CopyOnetimeCodeToClipboard != nil {
		if err := c.input.CopyOnetimeCodeToClipboard(ctx, deviceCode.UserCode); err != nil {
			c.input.Logger.FailedToCopyOnetimeCodeToClipboard(logger, c.input.Stderr, err)
		} else {
			copied = true
		}
	}

	deviceCodeExpirationDate := c.input.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	if err := c.input.OnetimeCodeUI.Show(ctx, logger, deviceCode, deviceCodeExpirationDate, &pubdeviceflow.InputShow{
		OpenBrowser:       willOpen,
		AppName:           input.AppName,
		CopiedToClipboard: copied,
	}); err != nil {
		return fmt.Errorf("show device code: %w", err)
	}
	if willOpen {
		if err := c.input.Browser.Open(ctx, logger, deviceCode.VerificationURI); err != nil {
			if !errors.Is(err, browser.ErrNoCommandFound) {
				c.input.Logger.FailedToOpenBrowser(logger, err)
			}
		} else {
			c.input.Logger.OpenedBrowser(logger, deviceCode.VerificationURI)
		}
	}
	return nil
}

func (c *Client) isOpenBrowser(input *InputCreate) bool {
	if !input.OpenBrowser {
		return false
	}
	if ac, ok := c.input.Browser.(availabilityChecker); ok {
		return ac.Available()
	}
	return true
}

func appendSkipAccountPickerParam(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse verification URL: %w", err)
	}
	query := u.Query()
	query.Set("skip_account_picker", "true")
	u.RawQuery = query.Encode()
	return u.String(), nil
}

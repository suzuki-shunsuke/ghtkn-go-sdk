package deviceflow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/browser"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
)

// availabilityChecker is an optional interface a Browser may implement to report
// whether it can actually open a browser on this host. When the configured Browser
// implements it and reports false, the verification URL is shown for the user to
// open manually instead of attempting (and failing) an automatic open.
type availabilityChecker interface {
	Available() bool
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

// Create initiates the OAuth device flow and returns an access token.
// It displays the verification URL and user code, opens a browser when one is
// available, and polls for the access token until the user completes authentication.
// When no browser is available, the user is asked to open the URL themselves.
func (c *Client) Create(ctx context.Context, logger *slog.Logger, input *InputCreate) (*AccessToken, error) {
	if input.ClientID == "" {
		return nil, errors.New("client id is required")
	}
	deviceCode, err := c.input.Client.GetDeviceCode(ctx, input.ClientID)
	if err != nil {
		return nil, fmt.Errorf("get device code: %w", err)
	}
	if input.SkipAccountPicker {
		verificationURI, err := appendSkipAccountPickerParam(deviceCode.VerificationURI)
		if err != nil {
			return nil, fmt.Errorf("add skip_account_picker to the verification URL: %w", err)
		}
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
		return nil, fmt.Errorf("show device code: %w", err)
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

	token, err := c.input.Client.Poll(ctx, logger, input.ClientID, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	now := c.input.Now()

	return &AccessToken{
		AccessToken:    token.AccessToken,
		ExpirationDate: now.Add(time.Duration(token.ExpiresIn) * time.Second),
	}, nil
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

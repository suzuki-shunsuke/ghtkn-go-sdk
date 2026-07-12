package deviceflow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow/ui"
)

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

	if err := c.input.OnetimeCodeUI.Show(ctx, logger, &ui.InputCreate{
		ClientID:          input.ClientID,
		AppName:           input.AppName,
		SkipAccountPicker: input.SkipAccountPicker,
		OpenBrowser:       input.OpenBrowser,
		Clipboard:         input.Clipboard,
	}, deviceCode); err != nil {
		return nil, err
	}

	token, err := c.input.Client.Poll(ctx, logger, input.ClientID, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	now := time.Now()

	return &AccessToken{
		AccessToken:    token.AccessToken,
		ExpirationDate: now.Add(time.Duration(token.ExpiresIn) * time.Second),
	}, nil
}

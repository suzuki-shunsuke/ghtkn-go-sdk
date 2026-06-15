package deviceflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/browser"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
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
}

// Create initiates the OAuth device flow and returns an access token.
// It displays the verification URL and user code, opens a browser when one is
// available, and polls for the access token until the user completes authentication.
// When no browser is available, the user is asked to open the URL themselves.
func (c *Client) Create(ctx context.Context, logger *slog.Logger, input *InputCreate) (*AccessToken, error) {
	if input.ClientID == "" {
		return nil, errors.New("client id is required")
	}
	deviceCode, err := c.getDeviceCode(ctx, input.ClientID)
	if err != nil {
		return nil, fmt.Errorf("get device code: %w", err)
	}

	// Decide up front whether the browser will actually be opened, so the UI can
	// show the right instruction and we only attempt an open that can succeed.
	willOpen := true
	if ac, ok := c.input.Browser.(availabilityChecker); ok {
		willOpen = ac.Available()
	}

	deviceCodeExpirationDate := c.input.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	if err := c.input.OnetimeCodeUI.Show(ctx, logger, deviceCode, deviceCodeExpirationDate, &pubdeviceflow.InputShow{
		OpenBrowser: willOpen,
		AppName:     input.AppName,
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

	token, err := c.pollForAccessToken(ctx, logger, input.ClientID, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	now := c.input.Now()

	return &AccessToken{
		AccessToken:    token.AccessToken,
		ExpirationDate: now.Add(time.Duration(token.ExpiresIn) * time.Second),
	}, nil
}

// getDeviceCode requests a device code from GitHub's OAuth device endpoint.
// It returns the device code response containing the user code and verification URL.
func (c *Client) getDeviceCode(ctx context.Context, clientID string) (*pubdeviceflow.DeviceCodeResponse, error) {
	if clientID == "" {
		return nil, errors.New("client id is required")
	}
	jsonData, err := json.Marshal(map[string]string{
		"client_id": clientID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal a request body as JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/device/code", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create a request for device code: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.input.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send a request for device code: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, slogerr.With( //nolint:wrapcheck
			errors.New("error from GitHub"),
			"status_code", resp.StatusCode,
			"body", string(body))
	}

	deviceCode := &pubdeviceflow.DeviceCodeResponse{}
	if err := json.Unmarshal(body, deviceCode); err != nil {
		return nil, fmt.Errorf("unmarshal response body as JSON: %w", err)
	}

	return deviceCode, nil
}

// additionalInterval is the minimum polling interval to avoid rate limiting.
const additionalInterval = 5 * time.Second

// pollForAccessToken continuously polls GitHub for an access token.
// It respects the polling interval and handles authorization pending and slow down responses.
// The polling continues until the device code expires or the user completes authentication.
func (c *Client) pollForAccessToken(ctx context.Context, logger *slog.Logger, clientID string, deviceCode *pubdeviceflow.DeviceCodeResponse) (*accessTokenResponse, error) {
	interval := max(time.Duration(deviceCode.Interval)*time.Second, additionalInterval)
	ticker := c.input.NewTicker(interval)
	defer ticker.Stop()

	deadline := c.input.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context was cancelled: %w", ctx.Err())
		case <-ticker.C:
			if c.input.Now().After(deadline) {
				return nil, errors.New("device code expired")
			}

			token, err := c.checkAccessToken(ctx, clientID, deviceCode.DeviceCode)
			if err != nil {
				if err.Error() == "authorization_pending" {
					logger.Debug("device flow's authorization is still pending")
					continue
				}
				if err.Error() == "slow_down" {
					logger.Debug("device flow's polling was too frequent, slowing down")
					ticker.Reset(interval + 5*time.Second)
					continue
				}
				return nil, err
			}

			if token != nil {
				return token, nil
			}
		}
	}
}

// checkAccessToken checks if an access token is available for the given device code.
// It returns the access token if available, or an error indicating the current status.
func (c *Client) checkAccessToken(ctx context.Context, clientID, deviceCode string) (*accessTokenResponse, error) {
	reqBody := map[string]string{
		"client_id":   clientID,
		"device_code": deviceCode,
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body as JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create a request for access token: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.input.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send a request for access token: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	token := &accessTokenResponse{}
	if err := json.Unmarshal(body, token); err != nil {
		return nil, fmt.Errorf("unmarshal response body as JSON: %w", err)
	}

	if token.Error != "" {
		return nil, errors.New(token.Error)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("unexpected response: %s", body)
	}
	return token, nil
}

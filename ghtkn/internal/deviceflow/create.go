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

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/browser"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// SetDeviceCodeUI updates the device code UI implementation used by the client.
// This allows customization of how device flow information is presented to users.
func (c *Client) SetDeviceCodeUI(ui DeviceCodeUI) {
	c.input.DeviceCodeUI = ui
}

// SetBrowser updates the browser implementation used by the client.
// This allows customization of how verification URLs are opened in the browser.
func (c *Client) SetBrowser(b Browser) {
	c.input.Browser = b
}

// Create initiates the OAuth device flow and returns an access token.
// It displays the verification URL and user code, optionally opens a browser,
// and polls for the access token until the user completes authentication.
func (c *Client) Create(ctx context.Context, logger *slog.Logger, clientID string) (*AccessToken, error) {
	if clientID == "" {
		return nil, errors.New("client id is required")
	}
	deviceCode, err := c.getDeviceCode(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("get device code: %w", err)
	}

	deviceCodeExpirationDate := c.input.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	if err := c.input.DeviceCodeUI.Show(ctx, logger, deviceCode, deviceCodeExpirationDate); err != nil {
		return nil, fmt.Errorf("show device code: %w", err)
	}
	if err := c.input.Browser.Open(ctx, logger, deviceCode.VerificationURI); err != nil {
		if !errors.Is(err, browser.ErrNoCommandFound) {
			c.input.Logger.FailedToOpenBrowser(logger, err)
		}
	}

	token, err := c.pollForAccessToken(ctx, logger, clientID, deviceCode)
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
func (c *Client) getDeviceCode(ctx context.Context, clientID string) (*DeviceCodeResponse, error) {
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

	deviceCode := &DeviceCodeResponse{}
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
func (c *Client) pollForAccessToken(ctx context.Context, logger *slog.Logger, clientID string, deviceCode *DeviceCodeResponse) (*AccessTokenResponse, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval < additionalInterval {
		interval = additionalInterval
	}

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
func (c *Client) checkAccessToken(ctx context.Context, clientID, deviceCode string) (*AccessTokenResponse, error) {
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

	token := &AccessTokenResponse{}
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

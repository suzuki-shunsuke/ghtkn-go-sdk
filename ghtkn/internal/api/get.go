package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// SetLogger updates the logger instance used by the token manager.
// It propagates the logger to both the token manager and device flow components.
func (tm *TokenManager) SetLogger(logger *publog.Logger) {
	log.InitLogger(logger)
	tm.input.Logger = logger
	tm.input.DeviceFlow.SetLogger(logger)
}

// SetOnetimeCodeUI updates the one-time code UI implementation used during OAuth device flow.
// This allows customization of how the one-time code (user code) is presented to users.
func (tm *TokenManager) SetOnetimeCodeUI(ui pubdeviceflow.OnetimeCodeUI) {
	tm.input.DeviceFlow.SetOnetimeCodeUI(ui)
}

// SetBrowser updates the browser implementation used to open verification URLs.
// This allows customization of how the GitHub verification page is opened during device flow.
func (tm *TokenManager) SetBrowser(ui pubdeviceflow.Browser) {
	tm.input.DeviceFlow.SetBrowser(ui)
}

// Get executes the main logic for retrieving a GitHub App access token.
// It checks for cached tokens and creates new tokens if needed.
//
// If the GHTKN_GITHUB_TOKEN environment variable is set, its value is returned
// as is without reading the config or contacting GitHub. This is useful when a
// tool embedding the ghtkn SDK must be handed a Personal Access Token directly.
// In this case the returned app config is nil and the access token has no
// expiration date.
func (tm *TokenManager) Get(ctx context.Context, logger *slog.Logger, input *pubapi.InputGet) (*pubapi.AccessToken, *pubconfig.App, error) {
	if token := tm.input.Getenv("GHTKN_GITHUB_TOKEN"); token != "" {
		return &pubapi.AccessToken{AccessToken: token}, nil, nil
	}
	if input == nil {
		input = &pubapi.InputGet{}
	}
	cfg := &pubconfig.Config{}

	// Get a config file path
	configPath := input.ConfigFilePath
	if configPath == "" {
		p, err := config.GetPath(tm.input.Getenv, tm.input.GOOS)
		if err != nil {
			return nil, nil, fmt.Errorf("get config path: %w", err)
		}
		configPath = p
	}

	// Read the config file
	if err := tm.readConfig(cfg, configPath); err != nil {
		return nil, nil, err
	}

	// Get the app name
	appName := input.AppName
	if appName == "" {
		appName = tm.input.Getenv("GHTKN_APP")
	}

	logger.Debug("selecting app", "app_name", appName, "git_owner", input.AppOwner)

	// Get the app config
	app := config.SelectApp(cfg, appName, input.AppOwner)
	if app == nil {
		return nil, nil, errors.New("app is not found in the config")
	}
	logger = logger.With("app_name", app.Name)

	// Debug Log
	logger.Debug(
		"getting or creating a GitHub App User Access Token",
		"min_expiration", input.MinExpiration,
	)

	token, changed, err := tm.getOrCreateToken(ctx, logger, &inputGetOrCreateToken{
		MinExpiration:    input.MinExpiration,
		App:              app,
		EnableDeviceFlow: enableDeviceFlow(input.EnableDeviceFlow, tm.input.Getenv),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get or create token: %w", err)
	}

	if changed {
		// Store the token in keyring
		if err := tm.input.Backend.Set(ctx, app.ClientID, &pubapi.AccessToken{
			AccessToken:    token.AccessToken,
			ExpirationDate: token.ExpirationDate,
		}); err != nil {
			return token, app, errStoreToken
		}
	}

	return token, app, nil
}

// errStoreToken is returned when the token cannot be stored in the keyring.
// This is a non-fatal error as the token is still valid for immediate use.
var errStoreToken = errors.New("could not store the token in keyring")

// inputGetOrCreateToken contains the parameters needed for token retrieval or creation.
// It encapsulates the app configuration and expiration requirements
// used internally by the getOrCreateToken function.
type inputGetOrCreateToken struct {
	App              *pubconfig.App // App configuration containing client ID and other settings
	MinExpiration    time.Duration  // Minimum time before expiration to consider token valid
	EnableDeviceFlow bool           // Whether the device flow may run to create a new token
}

// enableDeviceFlow resolves whether the device flow may run. An explicit override
// (e.g. a CLI flag) takes precedence; otherwise the GHTKN_ENABLE_DEVICE_FLOW
// environment variable decides, defaulting to enabled unless it is set to "false".
func enableDeviceFlow(override *bool, getEnv func(string) string) bool {
	if override != nil {
		return *override
	}
	return getEnv("GHTKN_ENABLE_DEVICE_FLOW") != "false"
}

// getOrCreateToken retrieves an existing token from the keyring or creates a new one.
// It returns the token, a boolean indicating whether the token was newly created or modified,
// and any error that occurred. The changed flag is used to determine if the token should be
// saved back to the keyring.
func (tm *TokenManager) getOrCreateToken(ctx context.Context, logger *slog.Logger, input *inputGetOrCreateToken) (*pubapi.AccessToken, bool, error) {
	// Get an access token from keyring
	token, err := tm.getAccessTokenFromBackend(ctx, logger, input)
	if err != nil {
		return nil, false, err
	}
	if token != nil {
		return token, false, nil
	}
	// Create access token
	token, err = tm.createToken(ctx, logger, input.App.ClientID, input.EnableDeviceFlow)
	if err != nil {
		return nil, false, fmt.Errorf("create a GitHub App User Access Token: %w", err)
	}
	return token, true, nil
}

// createToken generates a new GitHub App access token using the OAuth device flow.
// It returns a keyring.AccessToken with the token details and expiration date.
func (tm *TokenManager) createToken(ctx context.Context, logger *slog.Logger, clientID string, enableDeviceFlow bool) (*pubapi.AccessToken, error) {
	if !enableDeviceFlow {
		return nil, pubapi.ErrDisableDeviceFlow
	}
	tk, err := tm.input.DeviceFlow.Create(ctx, logger, clientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return &pubapi.AccessToken{
		AccessToken:    tk.AccessToken,
		ExpirationDate: tk.ExpirationDate,
	}, nil
}

// getAccessTokenFromBackend retrieves a cached access token from the system keyring.
// It returns nil if the token doesn't exist or has expired based on MinExpiration.
func (tm *TokenManager) getAccessTokenFromBackend(ctx context.Context, logger *slog.Logger, input *inputGetOrCreateToken) (*pubapi.AccessToken, error) {
	// Get an access token from keyring
	tk, err := tm.input.Backend.Get(ctx, input.App.ClientID)
	if err != nil {
		return nil, err
	}
	if tk == nil {
		tm.input.Logger.AccessTokenIsNotFoundInBackend(logger)
		return nil, nil
	}
	// Check if the access token expires
	if tm.checkExpired(tk.ExpirationDate, input.MinExpiration) {
		tm.input.Logger.Expire(logger, tk.ExpirationDate)
		return nil, nil
	}
	// Not expires
	return tk, nil
}

// checkExpired determines if an access token should be considered expired.
// It returns true if the token will expire within the MinExpiration duration from now.
// This ensures tokens are renewed before they actually expire.
func (tm *TokenManager) checkExpired(exDate time.Time, minExpiration time.Duration) bool {
	// Expiration Date - Now < Min Expiration
	// Now + Min Expiration > Expiration Date
	return tm.input.Now().Add(minExpiration).After(exDate)
}

// readConfig loads and validates the configuration from the configured file path.
// It returns an error if the configuration cannot be read or is invalid.
func (tm *TokenManager) readConfig(cfg *pubconfig.Config, configFilePath string) error {
	if err := tm.input.ConfigReader.Read(cfg, configFilePath); err != nil {
		return fmt.Errorf("read config: %w", slogerr.With(err, "config", configFilePath))
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}

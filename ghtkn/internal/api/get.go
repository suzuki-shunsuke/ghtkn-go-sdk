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
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
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

// SetCopyOnetimeCodeToClipboard updates the clipboard implementation used to copy the one-time code.
// This allows customization of how the one-time code is copied to the user's clipboard.
func (tm *TokenManager) SetCopyOnetimeCodeToClipboard(f pubdeviceflow.CopyTextToClipboard) {
	tm.input.DeviceFlow.SetCopyOnetimeCodeToClipboard(f)
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

	// Get a config file path and read the config file
	configPath, err := tm.resolveConfigPath(input.ConfigFilePath)
	if err != nil {
		return nil, nil, err
	}
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

	attrs := slogerr.NewAttrs(1)
	logger = attrs.Add(logger, "app_name", app.Name)

	minExpiration, err := resolveMinExpiration(input.MinExpiration, cfg.MinExpiration, tm.input.Getenv)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve the min expiration: %w", attrs.With(err))
	}

	b, err := tm.resolveBackend(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve the backend: %w", attrs.With(err))
	}

	// Debug Log
	logger.Debug(
		"getting or creating a GitHub App User Access Token",
		"min_expiration", minExpiration,
	)

	token, changed, err := tm.getOrCreateToken(ctx, logger, &inputGetOrCreateToken{
		MinExpiration:     minExpiration,
		App:               app,
		Backend:           b,
		EnableDeviceFlow:  enableDeviceFlow(input.EnableDeviceFlow, cfg.DeviceFlow, tm.input.Getenv),
		SkipAccountPicker: skipAccountPicker(cfg.SkipAccountPicker),
		OpenBrowser:       openBrowser(cfg.OpenBrowser, tm.input.Getenv),
		Clipboard:         clipboard(input.Clipboard, cfg.Clipboard, tm.input.Getenv),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get or create token: %w", attrs.With(err))
	}

	if changed {
		// Store the token in the backend
		if err := b.Set(ctx, app.ClientID, &pubapi.AccessToken{
			AccessToken:    token.AccessToken,
			ExpirationDate: token.ExpirationDate,
		}); err != nil {
			return token, app, attrs.With(errStoreToken)
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
	App               *pubconfig.App // App configuration containing client ID and other settings
	Backend           Backend        // Resolved storage backend for reading and writing the token
	MinExpiration     time.Duration  // Minimum time before expiration to consider token valid
	EnableDeviceFlow  bool           // Whether the device flow may run to create a new token
	SkipAccountPicker bool           // Whether the GitHub account picker should be skipped
	OpenBrowser       bool           // Whether the device flow may open a browser automatically
	Clipboard         bool           // Whether the device flow copies the one-time code to the clipboard
}

// enableDeviceFlow resolves whether the device flow may run. An explicit override
// (the -device-flow flag) takes precedence; otherwise the GHTKN_ENABLE_DEVICE_FLOW
// environment variable decides (only "false" disables it), then the config's
// device_flow.enable, defaulting to enabled.
func enableDeviceFlow(override *bool, cfg *pubconfig.DeviceFlow, getEnv func(string) string) bool {
	if override != nil {
		return *override
	}
	if v := getEnv("GHTKN_ENABLE_DEVICE_FLOW"); v != "" {
		return v != "false"
	}
	if cfg != nil && cfg.Enable != nil {
		return *cfg.Enable
	}
	return true
}

// resolveBackendType resolves the storage backend type. The GHTKN_BACKEND
// environment variable takes precedence, then the config's backend.type. An empty
// result selects the default (the OS keyring); backend.New maps it accordingly.
func resolveBackendType(cfg *pubconfig.Backend, getEnv func(string) string) string {
	if v := getEnv("GHTKN_BACKEND"); v != "" {
		return v
	}
	if cfg != nil && cfg.Type != "" {
		return cfg.Type
	}
	return ""
}

// resolveMinExpiration resolves the minimum time before token expiration that
// triggers renewal. An explicit override (the -min-expiration flag) takes
// precedence, including an explicit zero; otherwise the GHTKN_MIN_EXPIRATION
// environment variable decides, then the config's min_expiration. It defaults to
// zero (renew only once the token has actually expired). The environment variable
// and config values are Go duration strings such as "1h" or "30m".
func resolveMinExpiration(override *time.Duration, cfg string, getEnv func(string) string) (time.Duration, error) {
	if override != nil {
		return *override, nil
	}
	if v := getEnv("GHTKN_MIN_EXPIRATION"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("parse GHTKN_MIN_EXPIRATION as a duration: %w", slogerr.With(err, "min_expiration", v))
		}
		return d, nil
	}
	if cfg != "" {
		d, err := time.ParseDuration(cfg)
		if err != nil {
			return 0, fmt.Errorf("parse min_expiration in the config as a duration: %w", slogerr.With(err, "min_expiration", cfg))
		}
		return d, nil
	}
	return 0, nil
}

// openBrowser resolves whether the device flow may open a browser automatically.
// The GHTKN_OPEN_BROWSER environment variable, when set, takes precedence and
// only "false" disables the open. Otherwise the config's open_browser.enable
// decides, defaulting to enabled. This lets users in WSL, containers, and
// headless environments suppress the unreliable browser launch (and its noisy
// errors) and open the URL manually instead.
func openBrowser(cfg *pubconfig.OpenBrowser, getEnv func(string) string) bool {
	if v := getEnv("GHTKN_OPEN_BROWSER"); v != "" {
		return v != "false"
	}
	if cfg != nil && cfg.Enable != nil {
		return *cfg.Enable
	}
	return true
}

// clipboard resolves whether the device flow copies the one-time code to the
// system clipboard. An explicit override (the -clipboard flag) takes precedence;
// otherwise the GHTKN_CLIPBOARD environment variable decides (only "true" enables
// it), then the config's clipboard.enable, defaulting to disabled. Copying also
// requires the consumer to inject an implementation via SetCopyOnetimeCodeToClipboard.
func clipboard(override *bool, cfg *pubconfig.Clipboard, getEnv func(string) string) bool {
	if override != nil {
		return *override
	}
	if v := getEnv("GHTKN_CLIPBOARD"); v != "" {
		return v == "true"
	}
	if cfg != nil && cfg.Enable != nil {
		return *cfg.Enable
	}
	return false
}

// skipAccountPicker resolves whether the GitHub Device Flow account picker is
// skipped from the config value. nil means "not specified" and defaults to true
// (the picker is skipped); set it to false to show the account picker.
func skipAccountPicker(cfg *bool) bool {
	if cfg != nil {
		return *cfg
	}
	return true
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
	token, err = tm.createToken(ctx, logger, &deviceflow.InputCreate{
		ClientID:          input.App.ClientID,
		AppName:           input.App.Name,
		SkipAccountPicker: input.SkipAccountPicker,
		OpenBrowser:       input.OpenBrowser,
		Clipboard:         input.Clipboard,
	}, input.EnableDeviceFlow)
	if err != nil {
		return nil, false, fmt.Errorf("create a GitHub App User Access Token: %w", err)
	}
	return token, true, nil
}

// createToken generates a new GitHub App access token using the OAuth device flow.
// It returns a keyring.AccessToken with the token details and expiration date.
func (tm *TokenManager) createToken(ctx context.Context, logger *slog.Logger, input *deviceflow.InputCreate, enableDeviceFlow bool) (*pubapi.AccessToken, error) {
	if !enableDeviceFlow {
		return nil, pubapi.ErrDisableDeviceFlow
	}
	tk, err := tm.input.DeviceFlow.Create(ctx, logger, input)
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
	// Get an access token from the backend
	tk, err := input.Backend.Get(ctx, input.App.ClientID)
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

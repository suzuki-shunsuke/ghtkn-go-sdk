package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
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
	if token := tm.input.Getenv(env.GitHubToken); token != "" {
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
	// The effective config: the file plus the environment overrides, so the resolvers
	// below read values the environment has already been folded into.
	if err := tm.loadConfig(cfg, configPath); err != nil {
		return nil, nil, err
	}

	// Get the app name
	appName := input.AppName
	if appName == "" {
		appName = tm.input.Getenv(env.App)
	}

	logger.Debug("selecting app", "app_name", appName, "git_owner", input.AppOwner)

	// Get the app config
	app := pubconfig.ResolveApp(cfg, appName, input.AppOwner)
	if app == nil {
		return nil, nil, errors.New("app is not found in the config")
	}

	attrs := slogerr.NewAttrs(1)
	logger = attrs.Add(logger, "app_name", app.Name)

	minExpiration, err := resolveMinExpiration(input.MinExpiration, cfg.MinExpiration)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve the min expiration: %w", attrs.With(err))
	}

	b, err := tm.resolveBackend(logger, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve the backend: %w", attrs.With(err))
	}

	// Debug Log
	logger.Debug(
		"getting or creating a GitHub App User Access Token",
		"min_expiration", minExpiration,
	)

	enableDF, err := enableDeviceFlow(input.EnableDeviceFlow, tm.input.Getenv)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve whether the device flow is enabled: %w", attrs.With(err))
	}

	token, changed, err := tm.getOrCreateToken(ctx, logger, &inputGetOrCreateToken{
		MinExpiration:     minExpiration,
		App:               app,
		Backend:           b,
		EnableDeviceFlow:  enableDF,
		SkipAccountPicker: skipAccountPicker(cfg.SkipAccountPicker),
		OpenBrowser:       openBrowser(cfg.OpenBrowser),
		Clipboard:         clipboard(input.Clipboard, cfg.Clipboard),
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
// environment variable decides (a boolean parsed by strconv.ParseBool; an
// unparsable value is a hard error). The device flow is disabled by default so it
// is never started automatically; it must be enabled explicitly (e.g. by `ghtkn auth`).
func enableDeviceFlow(override *bool, getEnv func(string) string) (bool, error) {
	if override != nil {
		return *override, nil
	}
	if v := getEnv(env.EnableDeviceFlow); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("parse %s as a boolean: %w", env.EnableDeviceFlow, err)
		}
		return b, nil
	}
	return false, nil
}

// resolveBackendType resolves the storage backend type from the (already
// env-overridden) config's backend.type. An empty result selects the default (the OS
// keyring); backend.New maps it accordingly. The GHTKN_BACKEND override is applied
// upstream by config.ApplyEnvOverrides.
func resolveBackendType(cfg *pubconfig.Backend) string {
	if cfg != nil && cfg.Type != "" {
		return cfg.Type
	}
	return ""
}

// resolveMinExpiration resolves the minimum time before token expiration that
// triggers renewal. An explicit override (the -min-expiration flag) takes precedence,
// including an explicit zero; otherwise the config's min_expiration is used (with the
// GHTKN_MIN_EXPIRATION override already folded in by config.ApplyEnvOverrides). It
// defaults to zero (renew only once the token has actually expired). The config value
// is a Go duration string such as "1h" or "30m".
func resolveMinExpiration(override *time.Duration, cfg string) (time.Duration, error) {
	if override != nil {
		return *override, nil
	}
	if cfg != "" {
		d, err := time.ParseDuration(cfg)
		if err != nil {
			return 0, fmt.Errorf("parse min_expiration as a duration: %w", slogerr.With(err, "min_expiration", cfg))
		}
		return d, nil
	}
	return 0, nil
}

// openBrowser resolves whether the device flow may open a browser automatically from
// the (already env-overridden) config's open_browser.enable, defaulting to enabled. The
// GHTKN_OPEN_BROWSER override is applied upstream by config.ApplyEnvOverrides, which
// parses it with strconv.ParseBool and rejects a value it cannot parse. This lets users
// in WSL, containers, and headless environments suppress the unreliable browser launch
// and open the URL manually instead.
func openBrowser(cfg *pubconfig.OpenBrowser) bool {
	if cfg != nil && cfg.Enable != nil {
		return *cfg.Enable
	}
	return true
}

// clipboard resolves whether the device flow copies the one-time code to the system
// clipboard. An explicit override (the -clipboard flag) takes precedence; otherwise the
// (already env-overridden) config's clipboard.enable decides, defaulting to disabled.
// The GHTKN_CLIPBOARD override is applied upstream by config.ApplyEnvOverrides, which
// parses it with strconv.ParseBool and rejects a value it cannot parse. Copying also
// requires the consumer to inject an implementation via SetCopyOnetimeCodeToClipboard.
func clipboard(override *bool, cfg *pubconfig.Clipboard) bool {
	if override != nil {
		return *override
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
	create := func() (*pubapi.AccessToken, bool, error) {
		token, changed, err := tm.createToken(ctx, logger, input.Backend, input.MinExpiration, &deviceflow.InputCreate{
			ClientID:          input.App.ClientID,
			AppName:           input.App.Name,
			SkipAccountPicker: input.SkipAccountPicker,
			OpenBrowser:       input.OpenBrowser,
			Clipboard:         input.Clipboard,
		}, input.EnableDeviceFlow)
		if err != nil {
			return nil, false, fmt.Errorf("create a GitHub App User Access Token: %w", err)
		}
		return token, changed, nil
	}

	// A backend that owns the lifecycle (the agent) checks its own cache before it starts
	// a flow, so when the flow may run, one request does both: it returns a still-valid
	// token if there is one. Reading it separately first would make the agent run its
	// cache path twice, which means a second attempt to refresh an expiring token and,
	// when that refresh fails while the refresh token is still valid, a second copy of
	// the incident warning that failure raises.
	if input.Backend.SupportsDeviceFlow() && input.EnableDeviceFlow {
		return create()
	}

	// Get an access token from keyring
	token, err := tm.getAccessTokenFromBackend(ctx, logger, input)
	if err != nil {
		return nil, false, err
	}
	if token != nil {
		return token, false, nil
	}
	// Create access token
	return create()
}

// createToken generates a new GitHub App access token using the OAuth device flow.
// It returns the token and whether the caller must persist it (changed).
//
// When the backend runs the device flow itself (the agent), the flow runs on the
// server: this asks it to begin and, unless the agent already has a valid token,
// displays the one-time code with the shared UI and polls the backend until the server
// has minted and stored the token. The server already stored it, so changed is false.
// Otherwise the client-side device flow runs and the caller must store the token, so
// changed is true.
//
// Beginning is also how a cached token is read on that backend when the flow may run:
// the agent returns a still-valid token instead of starting a flow, so getOrCreateToken
// comes straight here rather than reading the backend first (see its comment).
func (tm *TokenManager) createToken(ctx context.Context, logger *slog.Logger, backend Backend, minExpiration time.Duration, input *deviceflow.InputCreate, enableDeviceFlow bool) (*pubapi.AccessToken, bool, error) {
	if !enableDeviceFlow {
		return nil, false, pubapi.ErrDisableDeviceFlow
	}
	if backend.SupportsDeviceFlow() {
		token, deviceCode, err := backend.BeginDeviceFlow(ctx, input.ClientID, minExpiration)
		if err != nil {
			return nil, false, fmt.Errorf("begin the device flow on the agent: %w", err)
		}
		if token != nil {
			// The agent had a still-valid token (cached, refreshed, or minted
			// concurrently), so no flow was started and there is nothing to display.
			return token, false, nil
		}
		// No usable token: report the miss the way the backend read does, since this is
		// the path that replaces it when the flow may run.
		tm.input.Logger.AccessTokenIsNotFoundInBackend(logger)
		if err := tm.input.DeviceFlow.Show(ctx, logger, input, deviceCode); err != nil {
			return nil, false, fmt.Errorf("show the one-time code: %w", err)
		}
		token, err = backend.PollDeviceFlow(ctx, input.ClientID, minExpiration)
		if err != nil {
			return nil, false, fmt.Errorf("wait for the agent to mint the token: %w", err)
		}
		return token, false, nil
	}
	tk, err := tm.input.DeviceFlow.Create(ctx, logger, input)
	if err != nil {
		return nil, false, err //nolint:wrapcheck
	}
	return &pubapi.AccessToken{
		AccessToken:    tk.AccessToken,
		ExpirationDate: tk.ExpirationDate,
	}, true, nil
}

// getAccessTokenFromBackend retrieves a still-valid cached access token from the
// backend, or nil when there is none. For a backend that owns the token lifecycle
// (the agent) the expiration check runs server-side; otherwise it is checked here
// against MinExpiration.
func (tm *TokenManager) getAccessTokenFromBackend(ctx context.Context, logger *slog.Logger, input *inputGetOrCreateToken) (*pubapi.AccessToken, error) {
	if input.Backend.SupportsDeviceFlow() {
		tk, err := input.Backend.GetActive(ctx, input.App.ClientID, input.MinExpiration)
		if err != nil {
			return nil, err
		}
		if tk == nil {
			tm.input.Logger.AccessTokenIsNotFoundInBackend(logger)
		}
		return tk, nil
	}
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
	return time.Now().Add(minExpiration).After(exDate)
}

// loadConfig reads the configuration file and folds the environment overrides into
// it, so every caller works with the effective (file plus environment) config. The
// two steps are always paired: the resolvers downstream (resolveBackendType,
// resolveMinExpiration, openBrowser, clipboard) read the config alone and would
// silently ignore the environment if a caller read the file without this.
func (tm *TokenManager) loadConfig(cfg *pubconfig.Config, configFilePath string) error {
	if err := tm.readConfig(cfg, configFilePath); err != nil {
		return err
	}
	if err := config.ApplyEnvOverrides(cfg, tm.input.Getenv); err != nil {
		return fmt.Errorf("apply environment overrides: %w", err)
	}
	return nil
}

// readConfig loads and validates the configuration from the configured file path.
// It returns an error if the configuration cannot be read or is invalid. It is the
// plain file read; use loadConfig to get the effective config.
func (tm *TokenManager) readConfig(cfg *pubconfig.Config, configFilePath string) error {
	if err := tm.input.ConfigReader.Read(cfg, configFilePath); err != nil {
		return fmt.Errorf("read config: %w", slogerr.With(err, "config", configFilePath))
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}

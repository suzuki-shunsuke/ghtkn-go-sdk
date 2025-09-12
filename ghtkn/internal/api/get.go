package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type InputGet struct {
	KeyringService string
	AppName        string
	ConfigFilePath string
	ClientID       string
	MinExpiration  time.Duration
}

func (tm *TokenManager) SetLogger(logger *log.Logger) {
	tm.input.Logger = logger
	tm.input.DeviceFlow.SetLogger(logger)
}

func (tm *TokenManager) SetDeviceCodeUI(ui deviceflow.DeviceCodeUI) {
	tm.input.DeviceFlow.SetDeviceCodeUI(ui)
}

func (tm *TokenManager) SetBrowser(ui deviceflow.Browser) {
	tm.input.DeviceFlow.SetBrowser(ui)
}

// Get executes the main logic for retrieving a GitHub App access token.
// It checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary.
func (tm *TokenManager) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*keyring.AccessToken, *config.App, error) {
	cfg := &config.Config{}

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

	// Get the app config
	app := cfg.SelectApp(appName)
	if app == nil {
		return nil, nil, errors.New("app is not found in the config")
	}

	// Get the keyring service name
	keyringService := input.KeyringService
	if keyringService == "" {
		keyringService = keyring.DefaultServiceKey
	}

	// Debug Log
	logFields := []any{"app_name", app.Name}
	logger.Debug(
		"getting or creating a GitHub App User Access Token",
		"min_expiration", input.MinExpiration,
	)

	logger = logger.With(logFields...)

	token, changed, err := tm.getOrCreateToken(ctx, logger, &inputGetOrCreateToken{
		KeyringService: keyringService,
		MinExpiration:  input.MinExpiration,
		App:            app,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get or create token: %w", err)
	}

	if token.Login == "" {
		// Get the authenticated user info for Git Credential Helper.
		// Git Credential Helper requires both username and password for authentication.
		// The username is the GitHub user's login name retrieved via the GitHub API.
		gh := tm.input.NewGitHub(ctx, token.AccessToken)
		user, err := gh.GetUser(ctx)
		if err != nil {
			return nil, app, fmt.Errorf("get authenticated user: %w", err)
		}
		token.Login = user.Login
	}

	if changed {
		// Store the token in keyring
		if err := tm.input.Keyring.Set(keyringService, app.ClientID, &keyring.AccessToken{
			AccessToken:    token.AccessToken,
			ExpirationDate: token.ExpirationDate,
			Login:          token.Login,
		}); err != nil {
			return token, app, ErrStoreToken
		}
	}

	return token, app, nil
}

var ErrStoreToken = errors.New("could not store the token in keyring")

type inputGetOrCreateToken struct {
	KeyringService string
	App            *config.App
	MinExpiration  time.Duration
}

// getOrCreateToken retrieves an existing token from the keyring or creates a new one.
// It returns the token, a boolean indicating whether the token was newly created or modified,
// and any error that occurred. The changed flag is used to determine if the token should be
// saved back to the keyring.
func (tm *TokenManager) getOrCreateToken(ctx context.Context, logger *slog.Logger, input *inputGetOrCreateToken) (*keyring.AccessToken, bool, error) {
	// Get an access token from keyring
	if token := tm.getAccessTokenFromKeyring(logger, input.KeyringService, input.App.ClientID, input.MinExpiration); token != nil {
		return token, false, nil
	}
	// Create access token
	token, err := tm.createToken(ctx, logger, input.App.ClientID)
	if err != nil {
		return nil, false, fmt.Errorf("create a GitHub App User Access Token: %w", err)
	}
	return token, true, nil
}

// createToken generates a new GitHub App access token using the OAuth device flow.
// It returns a keyring.AccessToken with the token details and expiration date.
func (tm *TokenManager) createToken(ctx context.Context, logger *slog.Logger, clientID string) (*keyring.AccessToken, error) {
	tk, err := tm.input.DeviceFlow.Create(ctx, logger, clientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return &keyring.AccessToken{
		AccessToken:    tk.AccessToken,
		ExpirationDate: tk.ExpirationDate,
	}, nil
}

// getAccessTokenFromKeyring retrieves a cached access token from the system keyring.
// It returns nil if the token doesn't exist or has expired based on MinExpiration.
func (tm *TokenManager) getAccessTokenFromKeyring(logger *slog.Logger, keyringService, key string, minExpiration time.Duration) *keyring.AccessToken {
	// Get an access token from keyring
	tk, err := tm.input.Keyring.Get(keyringService, key)
	if err != nil {
		tm.input.Logger.FailedToGetAccessTokenFromKeyring(logger, err)
		return nil
	}
	if tk == nil {
		tm.input.Logger.AccessTokenIsNotFoundInKeyring(logger)
		return nil
	}
	// Check if the access token expires
	if tm.checkExpired(tk.ExpirationDate, minExpiration) {
		tm.input.Logger.Expire(logger, tk.ExpirationDate)
		return nil
	}
	// Not expires
	return tk
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
func (tm *TokenManager) readConfig(cfg *config.Config, configFilePath string) error {
	if err := tm.input.ConfigReader.Read(cfg, configFilePath); err != nil {
		return fmt.Errorf("read config: %w", slogerr.With(err, "config", configFilePath))
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}

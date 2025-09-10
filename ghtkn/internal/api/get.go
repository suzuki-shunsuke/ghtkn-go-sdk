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
	User           string
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

	// Get the user login
	login := input.User
	if login == "" {
		login = tm.input.Getenv("GHTKN_USER")
	}

	// Get the user config
	configUser := cfg.SelectUser(login)
	if configUser == nil {
		return nil, nil, errors.New("user is not found in the config")
	}

	// Get the app name
	appName := input.AppName
	if appName == "" {
		appName = tm.input.Getenv("GHTKN_APP")
	}

	// Get the app config
	app := configUser.SelectApp(appName)
	if app == nil {
		return nil, nil, errors.New("app is not found in the config")
	}

	// Get the keyring service name
	keyringService := input.KeyringService
	if keyringService == "" {
		keyringService = keyring.DefaultServiceKey
	}

	// Debug Log
	logFields := []any{"app_name", app.Name, "user", configUser.Login}
	logger.Debug(
		"getting or creating a GitHub App User Access Token",
		"min_expiration", input.MinExpiration,
	)

	logger = logger.With(logFields...)

	atKey := &keyring.AccessTokenKey{
		Login: configUser.Login,
		AppID: app.AppID,
	}

	token, changed, err := tm.getOrCreateToken(ctx, logger, &inputGetOrCreateToken{
		KeyringService: keyringService,
		MinExpiration:  input.MinExpiration,
		User:           configUser.Login,
		AppID:          app.AppID,
		Key:            atKey,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get or create token: %w", err)
	}

	if changed {
		// Get the authenticated user info for Git Credential Helper.
		// Git Credential Helper requires both username and password for authentication.
		// The username is the GitHub user's login name retrieved via the GitHub API.
		gh := tm.input.NewGitHub(ctx, token.AccessToken)
		user, err := gh.GetUser(ctx)
		if err != nil {
			return nil, app, fmt.Errorf("get authenticated user: %w", err)
		}
		token.Login = user.Login
	} else if token.Login == "" {
		token.Login = configUser.Login
	}

	// Update the key with the final login value
	atKey.Login = token.Login

	if changed {
		// Store the token in keyring
		if err := tm.input.Keyring.Set(keyringService, atKey, &keyring.AccessToken{
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
	User           string
	AppID          int
	MinExpiration  time.Duration
	Key            *keyring.AccessTokenKey
}

// getOrCreateToken retrieves an existing token from the keyring or creates a new one.
// It returns the token, a boolean indicating whether the token was newly created or modified,
// and any error that occurred. The changed flag is used to determine if the token should be
// saved back to the keyring.
func (tm *TokenManager) getOrCreateToken(ctx context.Context, logger *slog.Logger, input *inputGetOrCreateToken) (*keyring.AccessToken, bool, error) {
	// Get an access token from keyring
	if token := tm.getAccessTokenFromKeyring(logger, input.KeyringService, input.Key, input.MinExpiration); token != nil {
		return token, false, nil
	}
	// Get the client id from keyring
	app := tm.getAppFromKeyring(logger, input.KeyringService, input.AppID)
	if app == nil {
		// Read client id from stdin
		cID, err := tm.input.ClientIDReader.Read()
		if err != nil {
			return nil, false, fmt.Errorf("read client id: %w", err)
		}
		app = &keyring.App{
			ClientID: string(cID),
		}
		// Store the client id in keyring
		if err := tm.input.AppStore.Set(input.KeyringService, input.AppID, app); err != nil {
			return nil, false, fmt.Errorf("store client id in keyring: %w", err)
		}
	}
	// Create access token
	token, err := tm.createToken(ctx, logger, app.ClientID)
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
func (tm *TokenManager) getAccessTokenFromKeyring(logger *slog.Logger, keyringService string, key *keyring.AccessTokenKey, minExpiration time.Duration) *keyring.AccessToken {
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

func (tm *TokenManager) getAppFromKeyring(logger *slog.Logger, keyringService string, appID int) *keyring.App {
	app, err := tm.input.AppStore.Get(keyringService, appID)
	if err != nil {
		tm.input.Logger.FailedToGetAppFromKeyring(logger, err)
		return nil
	}
	if app == nil {
		tm.input.Logger.AppIsNotFoundInKeyring(logger)
		return nil
	}
	return app
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

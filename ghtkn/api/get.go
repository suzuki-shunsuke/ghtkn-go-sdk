package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type InputGet struct {
	ClientID       string
	UseKeyring     bool
	KeyringService string
	UseConfig      bool
	AppName        string
	ConfigFilePath string
}

func (tm *TokenManager) SetLogger(logger *log.Logger) {
	tm.input.Logger = logger
	tm.input.AppTokenClient.SetLogger(logger)
}

// Get executes the main logic for retrieving a GitHub App access token.
// It checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary.
func (tm *TokenManager) Get(ctx context.Context, logger *slog.Logger, input *InputGet) (*keyring.AccessToken, *config.App, error) {
	var app *config.App
	if input.UseConfig {
		cfg := &config.Config{}
		if err := tm.readConfig(cfg, input.ConfigFilePath); err != nil {
			return nil, nil, err
		}
		// Select the app config
		app = cfg.SelectApp(input.AppName)
	} else {
		if input.ClientID == "" {
			return nil, nil, errors.New("ClientID is required when not using config")
		}
		app = &config.App{
			Name:     input.AppName,
			ClientID: input.ClientID,
		}
	}

	logFields := []any{"app", app.Name}
	logger = logger.With(logFields...)

	token, changed, err := tm.getOrCreateToken(ctx, logger, input)
	if err != nil {
		return nil, nil, fmt.Errorf("get or create token: %w", err)
	}

	if token.Login == "" {
		// Get the authenticated user info for Git Credential Helper.
		// Git Credential Helper requires both username and password for authentication.
		// The username is the GitHub user's login name retrieved via the GitHub API.
		gh := tm.input.NewGitHub(ctx, token.AccessToken)
		user, err := gh.Get(ctx)
		if err != nil {
			return nil, app, fmt.Errorf("get authenticated user: %w", err)
		}
		token.Login = user.Login
		changed = true
	}

	if input.UseKeyring && changed {
		// Store the token in keyring
		if err := tm.input.Keyring.Set(input.KeyringService, input.ClientID, &keyring.AccessToken{
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

// getOrCreateToken retrieves an existing token from the keyring or creates a new one.
// It returns the token, a boolean indicating whether the token was newly created or modified,
// and any error that occurred. The changed flag is used to determine if the token should be
// saved back to the keyring.
func (tm *TokenManager) getOrCreateToken(ctx context.Context, logger *slog.Logger, input *InputGet) (*keyring.AccessToken, bool, error) {
	// Get an access token from keyring
	if input.UseKeyring {
		token, err := tm.getAccessTokenFromKeyring(logger, input.KeyringService, input.ClientID)
		if err != nil {
			slogerr.WithError(logger, err).Info("failed to get a GitHub App User Access Token from keyring")
		}
		if token != nil {
			return token, false, nil
		}
	}
	// Create access token
	token, err := tm.createToken(ctx, logger, input.ClientID)
	if err != nil {
		return nil, false, fmt.Errorf("create a GitHub App User Access Token: %w", err)
	}
	return token, true, nil
}

// createToken generates a new GitHub App access token using the OAuth device flow.
// It returns a keyring.AccessToken with the token details and expiration date.
func (tm *TokenManager) createToken(ctx context.Context, logger *slog.Logger, clientID string) (*keyring.AccessToken, error) {
	tk, err := tm.input.AppTokenClient.Create(ctx, logger, clientID)
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
func (tm *TokenManager) getAccessTokenFromKeyring(logger *slog.Logger, keyringService string, clientID string) (*keyring.AccessToken, error) {
	// Get an access token from keyring
	tk, err := tm.input.Keyring.Get(keyringService, clientID)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	if tk == nil {
		return nil, nil //nolint:nilnil
	}
	// Check if the access token expires
	expired, err := tm.checkExpired(tk.ExpirationDate)
	if err != nil {
		return nil, fmt.Errorf("check if the access token is expired: %w", err)
	}
	if expired {
		tm.input.Logger.Expire(logger, tk.ExpirationDate)
		return nil, nil //nolint:nilnil
	}
	// Not expires
	return tk, nil
}

// checkExpired determines if an access token should be considered expired.
// It returns true if the token will expire within the MinExpiration duration from now.
// This ensures tokens are renewed before they actually expire.
func (tm *TokenManager) checkExpired(exDate string) (bool, error) {
	t, err := keyring.ParseDate(exDate)
	if err != nil {
		return false, err //nolint:wrapcheck
	}
	// Expiration Date - Now < Min Expiration
	// Now + Min Expiration > Expiration Date
	return tm.input.Now().Add(tm.input.MinExpiration).After(t), nil
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

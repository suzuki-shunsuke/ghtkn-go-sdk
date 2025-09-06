package ghtkn

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Get executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary.
func (c *Client) Get(ctx context.Context, logger *slog.Logger) (*keyring.AccessToken, *config.App, error) {
	cfg := &config.Config{}
	if err := c.readConfig(cfg); err != nil {
		return nil, nil, err
	}

	// Select the app config
	app := cfg.SelectApp(c.input.Env.App)
	logFields := []any{"app", app.Name}
	logger = logger.With(logFields...)

	token, err := c.input.TokenManager.Get(ctx, logger, &api.InputGet{
		ClientID:   app.ClientID,
		UseKeyring: cfg.Persist,
	})
	if err != nil {
		return nil, app, fmt.Errorf("get access token: %w", slogerr.With(err, logFields...))
	}

	return token, app, nil
}

// readConfig loads and validates the configuration from the configured file path.
// It returns an error if the configuration cannot be read or is invalid.
func (c *Client) readConfig(cfg *config.Config) error {
	if err := c.input.ConfigReader.Read(cfg, c.input.ConfigFilePath); err != nil {
		return fmt.Errorf("read config: %w", slogerr.With(err, "config", c.input.ConfigFilePath))
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
}

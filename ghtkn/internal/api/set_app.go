package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
)

type InputSetApp struct {
	AppName        string
	ConfigFilePath string
	User           string
	KeyringService string
}

func (tm *TokenManager) SetApp(ctx context.Context, logger *slog.Logger, input *InputSetApp) error {
	// Read the config file
	cfg := &config.Config{}
	if err := tm.readConfig(cfg, input.ConfigFilePath); err != nil {
		return err
	}

	// Get the user login
	user, err := tm.getUserConfig(input.User, cfg)
	if err != nil {
		return err
	}

	// Get the app name
	appConfig, err := tm.getAppConfig(input.User, user)
	if err != nil {
		return err
	}

	// Get the keyring service name
	keyringService := input.KeyringService
	if keyringService == "" {
		keyringService = keyring.DefaultServiceKey
	}

	// Debug Log
	logFields := []any{"app_name", appConfig.Name, "user", user.Login}
	logger = logger.With(logFields...)
	logger.Debug(
		"Setting the app to store",
	)

	cID, err := tm.input.ClientIDReader.Read(ctx, logger, appConfig)
	if err != nil {
		return fmt.Errorf("read client id: %w", err)
	}
	if cID == "" {
		return errors.New("cancelled")
	}
	app := &keyring.App{
		ClientID: strings.TrimSpace(string(cID)),
	}
	// Store the client id in keyring
	if err := tm.input.AppStore.Set(keyringService, appConfig.AppID, app); err != nil {
		return fmt.Errorf("store client id in keyring: %w", err)
	}
	return nil
}

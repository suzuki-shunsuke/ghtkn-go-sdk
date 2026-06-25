package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

// resolveConfigPath returns the config file path to read. When p is empty the
// default path is auto-detected from the environment.
func (tm *TokenManager) resolveConfigPath(p string) (string, error) {
	if p != "" {
		return p, nil
	}
	path, err := config.GetPath(tm.input.Getenv, tm.input.GOOS)
	if err != nil {
		return "", fmt.Errorf("get config path: %w", err)
	}
	return path, nil
}

// Revoke revokes GitHub credentials and removes the revoked tokens from the backend.
//
// The tokens to revoke are the tokens stored in the backend for each app in
// input.AppNames. When AppNames is empty, it falls back to the app selected by
// GHTKN_APP (or the default app).
//
// Reading the backend never triggers the device flow and ignores expiration: a
// stored token is revoked regardless of whether it has expired. Apps with no
// stored token are skipped. Tokens read from the backend are deleted from it
// after they are revoked. When there is nothing to revoke, Revoke is a no-op.
func (tm *TokenManager) Revoke(ctx context.Context, logger *slog.Logger, input *pubapi.InputRevoke) error {
	if input == nil {
		input = &pubapi.InputRevoke{}
	}

	cfg := &pubconfig.Config{}
	configPath, err := tm.resolveConfigPath(input.ConfigFilePath)
	if err != nil {
		return err
	}
	if err := tm.readConfig(cfg, configPath); err != nil {
		return err
	}

	appNames := input.AppNames
	if len(appNames) == 0 {
		// No app names given: fall back to GHTKN_APP, then the default app.
		app := config.SelectApp(cfg, tm.input.Getenv("GHTKN_APP"), "")
		if app == nil {
			return errors.New("app is not found in the config")
		}
		appNames = []string{app.Name}
	}

	tokens := make([]string, 0, len(appNames))
	// clientIDs of tokens read from the backend, to delete after revocation.
	var clientIDs []string
	for _, name := range appNames {
		app := config.SelectApp(cfg, name, "")
		if app == nil {
			return fmt.Errorf("app is not found in the config: %s", name)
		}
		tk, err := tm.input.Backend.Get(ctx, app.ClientID)
		if err != nil {
			return fmt.Errorf("get a stored token from the backend: %w", err)
		}
		if tk == nil {
			// Nothing stored for this app: do nothing for it.
			logger.Debug("no stored token to revoke", "app_name", app.Name)
			continue
		}
		clientIDs = append(clientIDs, app.ClientID)
		tokens = append(tokens, tk.AccessToken)
	}

	if len(tokens) == 0 {
		return nil
	}

	if err := tm.input.Revoker.Revoke(ctx, tokens); err != nil {
		return fmt.Errorf("revoke credentials: %w", err)
	}

	// Remove the revoked tokens from the backend.
	for _, clientID := range clientIDs {
		if err := tm.input.Backend.Delete(ctx, clientID); err != nil {
			return fmt.Errorf("delete a revoked token from the backend: %w", err)
		}
	}
	return nil
}

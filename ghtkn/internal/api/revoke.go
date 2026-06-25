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
// The tokens to revoke are the union of:
//   - the tokens stored in the backend for each app in input.AppNames, and
//   - the raw tokens in input.Tokens.
//
// As a special case, when neither AppNames nor Tokens is given, it falls back to
// the app selected by GHTKN_APP (or the default app). When AppNames or Tokens is
// given, GHTKN_APP and the default app are NOT used, so passing only a raw token
// never revokes an unrelated app's stored token.
//
// Reading the backend never triggers the device flow and ignores expiration: a
// stored token is revoked regardless of whether it has expired. Apps with no
// stored token are skipped. Only tokens read from the backend are deleted from it;
// raw tokens in input.Tokens are revoked but not deleted (their backend, if any,
// is unknown). When there is nothing to revoke, Revoke is a no-op.
func (tm *TokenManager) Revoke(ctx context.Context, logger *slog.Logger, input *pubapi.InputRevoke) error {
	if input == nil {
		input = &pubapi.InputRevoke{}
	}

	tokens := make([]string, 0, len(input.Tokens)+len(input.AppNames))
	seen := map[string]struct{}{}
	add := func(t string) {
		if t == "" {
			return
		}
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		tokens = append(tokens, t)
	}

	// clientIDs of tokens read from the backend, to delete after revocation.
	var clientIDs []string

	// Resolve apps and collect their stored tokens. The config is read only when
	// an app needs to be resolved (explicit AppNames, or the no-argument fallback).
	fallback := len(input.AppNames) == 0 && len(input.Tokens) == 0
	if len(input.AppNames) > 0 || fallback {
		cfg := &pubconfig.Config{}
		configPath, err := tm.resolveConfigPath(input.ConfigFilePath)
		if err != nil {
			return err
		}
		if err := tm.readConfig(cfg, configPath); err != nil {
			return err
		}

		appNames := input.AppNames
		if fallback {
			// No arguments at all: fall back to GHTKN_APP, then the default app.
			app := config.SelectApp(cfg, tm.input.Getenv("GHTKN_APP"), input.AppOwner)
			if app == nil {
				return errors.New("app is not found in the config")
			}
			appNames = []string{app.Name}
		}

		for _, name := range appNames {
			app := config.SelectApp(cfg, name, input.AppOwner)
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
			add(tk.AccessToken)
		}
	}

	for _, t := range input.Tokens {
		add(t)
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

package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
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

// revokeAppNames resolves which apps' stored tokens Revoke should target.
//
//   - input.All: every app in the config (AppNames and the GHTKN_APP / default-app
//     fallback are ignored).
//   - input.AppNames given: those apps.
//   - otherwise: the app selected by GHTKN_APP, then the default app.
func (tm *TokenManager) revokeAppNames(cfg *pubconfig.Config, input *pubapi.InputRevoke) ([]string, error) {
	if input.All {
		names := make([]string, 0, len(cfg.Apps))
		for _, app := range cfg.Apps {
			names = append(names, app.Name)
		}
		return names, nil
	}
	if len(input.AppNames) > 0 {
		return input.AppNames, nil
	}
	app := pubconfig.ResolveApp(cfg, tm.input.Getenv(env.App), "")
	if app == nil {
		return nil, errors.New("app is not found in the config")
	}
	return []string{app.Name}, nil
}

// Revoke revokes GitHub credentials and removes the revoked tokens from the backend.
//
// The tokens to revoke are the tokens stored in the backend for the apps selected
// by input (see revokeAppNames): every app when input.All is set, the apps in
// input.AppNames, or the GHTKN_APP / default app as a fallback.
//
// Reading the backend never triggers the device flow and ignores expiration: a
// stored token is revoked regardless of whether it has expired. Apps with no
// stored token are skipped. Tokens read from the backend are deleted from it
// after they are revoked. When there is nothing to revoke, Revoke is a no-op.
//
// Revoke is best-effort: a failure for one app does not stop the others, and all
// failures are aggregated with errors.Join. Each aggregated error is wrapped with
// pubapi.ErrRevoke (the credential may still be live) or pubapi.ErrBackendCleanup
// (the credential is revoked but a stale copy remains in the backend) so callers
// can tell the two apart with errors.Is.
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

	appNames, err := tm.revokeAppNames(cfg, input)
	if err != nil {
		return err
	}

	b, err := tm.resolveBackend(logger, cfg)
	if err != nil {
		return fmt.Errorf("resolve the backend: %w", err)
	}

	// The agent owns the token lifecycle, so it revokes and deletes each stored token
	// itself; the client only tells it which apps to revoke.
	if b.SupportsDeviceFlow() {
		return tm.revokeViaBackend(ctx, logger, b, cfg, appNames)
	}

	tokens := make([]string, 0, len(appNames))
	// clientIDs of tokens read from the backend, to delete after revocation.
	var clientIDs []string
	// errs aggregates per-app failures so one bad app doesn't block the rest.
	var errs []error
	for _, name := range appNames {
		app := pubconfig.ResolveApp(cfg, name, "")
		if app == nil {
			// The intended token was not revoked: treat as a live-credential failure.
			errs = append(errs, fmt.Errorf("app is not found in the config: %s: %w", name, pubapi.ErrRevoke))
			continue
		}
		tk, err := b.Get(ctx, app.ClientID)
		if err != nil {
			// The token couldn't be read, so it can't be revoked: it may still be live.
			errs = append(errs, fmt.Errorf("get a stored token from the backend: app_name=%s: %w: %w", app.Name, err, pubapi.ErrRevoke))
			continue
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
		return errors.Join(errs...)
	}

	if err := tm.input.Revoker.Revoke(ctx, tokens); err != nil {
		// The revocation API call failed: the credentials may still be live, so they
		// must NOT be deleted from the backend.
		errs = append(errs, fmt.Errorf("revoke credentials: %w: %w", err, pubapi.ErrRevoke))
		return errors.Join(errs...)
	}

	// Remove the revoked tokens from the backend (best-effort). These tokens are
	// already revoked, so a failure here is a cleanup/UX issue, not a security one.
	for _, clientID := range clientIDs {
		if err := b.Delete(ctx, clientID); err != nil {
			errs = append(errs, fmt.Errorf("delete a revoked token from the backend: client_id=%s: %w: %w", clientID, err, pubapi.ErrBackendCleanup))
		}
	}
	return errors.Join(errs...)
}

// revokeViaBackend revokes the apps' stored tokens through a backend that owns the
// token lifecycle (the agent). It resolves the apps to client IDs and hands them to
// the backend, which revokes and deletes them in one batch. The backend reports which
// client IDs it could not revoke (ErrRevoke, the credential may be live) and which it
// revoked but could not delete (ErrBackendCleanup), which are mapped back to app names.
func (tm *TokenManager) revokeViaBackend(ctx context.Context, logger *slog.Logger, b Backend, cfg *pubconfig.Config, appNames []string) error {
	var errs []error
	clientIDs := make([]string, 0, len(appNames))
	appByClientID := make(map[string]string, len(appNames))
	for _, name := range appNames {
		app := pubconfig.ResolveApp(cfg, name, "")
		if app == nil {
			errs = append(errs, fmt.Errorf("app is not found in the config: %s: %w", name, pubapi.ErrRevoke))
			continue
		}
		clientIDs = append(clientIDs, app.ClientID)
		appByClientID[app.ClientID] = app.Name
	}
	if len(clientIDs) == 0 {
		return errors.Join(errs...)
	}

	revokeFailed, cleanupFailed, err := b.RevokeTokens(ctx, clientIDs)
	if err != nil {
		// The request itself failed (e.g. the agent is not running or locked), so no
		// credential was revoked.
		errs = append(errs, fmt.Errorf("revoke tokens through the backend: %w: %w", err, pubapi.ErrRevoke))
		return errors.Join(errs...)
	}
	logger.Debug("revoked tokens through the agent", "count", len(clientIDs)-len(revokeFailed))
	for _, clientID := range revokeFailed {
		errs = append(errs, fmt.Errorf("revoke a token through the backend: app_name=%s: %w", appByClientID[clientID], pubapi.ErrRevoke))
	}
	for _, clientID := range cleanupFailed {
		errs = append(errs, fmt.Errorf("delete a revoked token from the backend: app_name=%s: %w", appByClientID[clientID], pubapi.ErrBackendCleanup))
	}
	return errors.Join(errs...)
}

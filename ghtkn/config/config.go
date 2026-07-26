// Package config provides the public configuration types for ghtkn.
// These types describe the configuration of GitHub Apps used for authentication.
package config

import (
	"errors"
	"fmt"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// Config represents the main configuration structure for ghtkn.
// It contains settings a list of GitHub Apps.
type Config struct {
	Apps []*App `json:"apps"`
	// SkipAccountPicker skips the GitHub Device Flow account picker by appending
	// GitHub's unofficial skip_account_picker query parameter to the verification
	// URL. nil means "not specified" and defaults to true (the picker is skipped);
	// set it to false to show the account picker.
	SkipAccountPicker *bool `json:"skip_account_picker,omitempty" yaml:"skip_account_picker"`
	// OpenBrowser controls whether the device flow opens a browser automatically.
	OpenBrowser *OpenBrowser `json:"open_browser,omitempty" yaml:"open_browser"`
	// MinExpiration is the minimum time before token expiration that triggers
	// renewal, as a Go duration string (e.g. "1h", "30m"). Empty means "not
	// specified" and defaults to zero (renew only once the token has actually
	// expired). The -min-expiration flag and the GHTKN_MIN_EXPIRATION environment
	// variable take precedence over this value.
	MinExpiration string `json:"min_expiration,omitempty" yaml:"min_expiration"`
	// Backend selects the storage backend for access tokens.
	Backend *Backend `json:"backend,omitempty" yaml:"backend"`
	// Clipboard configures whether the device flow copies the one-time code to the
	// system clipboard.
	Clipboard *Clipboard `json:"clipboard,omitempty" yaml:"clipboard"`
}

// OpenBrowser configures automatic browser opening for the device flow.
type OpenBrowser struct {
	// Enable toggles automatic browser opening. nil means "not specified" and
	// defaults to true. The GHTKN_OPEN_BROWSER environment variable, when set,
	// takes precedence over this value.
	Enable *bool `json:"enable,omitempty" yaml:"enable"`
}

// Clipboard configures whether the device flow copies the one-time code to the
// system clipboard.
type Clipboard struct {
	// Enable toggles copying the one-time code to the clipboard. nil means "not
	// specified" and defaults to false. The -clipboard flag and the GHTKN_CLIPBOARD
	// environment variable take precedence over this value.
	Enable *bool `json:"enable,omitempty" yaml:"enable"`
}

// Backend selects the storage backend for access tokens.
type Backend struct {
	// Type is the backend type: "keyring" (the default), "text", or "agent". Empty
	// means "not specified". The GHTKN_BACKEND environment variable takes precedence
	// over this value.
	Type string `json:"type,omitempty" yaml:"type"`
}

// Validate checks if the Config is valid.
// It ensures the config is not nil and contains at least one app.
// It also validates each app in the configuration, and that name, client_id, and
// git_owner are each unique across the apps.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}
	if len(c.Apps) == 0 {
		return errors.New("apps is required")
	}
	names := map[string]struct{}{}
	owners := map[string]struct{}{}
	// clientIDs maps a client ID to the app that declared it, so a duplicate can name
	// the other app instead of only the ID.
	clientIDs := map[string]string{}
	for _, app := range c.Apps {
		if err := app.Validate(); err != nil {
			return fmt.Errorf("app is invalid: %w", slogerr.With(err, "app", app.Name))
		}
		if _, ok := names[app.Name]; ok {
			return fmt.Errorf("app name must be unique: %s", app.Name)
		}
		names[app.Name] = struct{}{}
		// A client ID identifies the GitHub App everywhere below the config: the stored
		// token, the refresh, and the revocation are all keyed by it. Two apps sharing
		// one are therefore two names for a single token, where revoking or minting for
		// one silently does it for the other. Reject that instead of letting the two
		// entries look independent.
		if other, ok := clientIDs[app.ClientID]; ok {
			return fmt.Errorf(
				"app client_id must be unique: %q and %q share the client id %s. "+
					"They would share one access token, so revoking or minting for one would silently do it for the other. "+
					"Keep a single app for that client id; to use it for several repository owners, "+
					"select it with the GHTKN_APP or GHTKN_GIT_APP environment variable instead of a second git_owner entry",
				other, app.Name, app.ClientID)
		}
		clientIDs[app.ClientID] = app.Name
		if app.GitOwner != "" {
			if _, ok := owners[app.GitOwner]; ok {
				return fmt.Errorf("app git_owner must be unique: %s", app.GitOwner)
			}
			owners[app.GitOwner] = struct{}{}
		}
	}
	return nil
}

// App represents a GitHub App configuration.
// Each app must have a unique name and a client ID for authentication.
type App struct {
	Name     string `json:"name"`
	ClientID string `json:"client_id" yaml:"client_id"`
	GitOwner string `json:"git_owner,omitempty" yaml:"git_owner"`
}

// Validate checks if the App configuration is valid.
// It ensures both Name and ClientID fields are present.
func (app *App) Validate() error {
	if app.Name == "" {
		return errors.New("name is required")
	}
	if app.ClientID == "" {
		return errors.New("client_id is required")
	}
	return nil
}

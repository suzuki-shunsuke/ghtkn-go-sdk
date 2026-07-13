package ghtkn

import (
	"fmt"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
	intconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

// LoadConfig finds the ghtkn configuration file (honoring GHTKN_CONFIG, then the
// XDG/OS default path), reads it with the SDK's own YAML decoder, and applies every
// environment override that maps onto a config field, so the returned Config reflects
// the file plus environment. A missing config file yields an empty Config with the
// environment overrides still applied.
//
// Per-call flag overrides (e.g. -min-expiration, -clipboard) are NOT reflected here;
// those are applied at Get time on top of this value.
func LoadConfig() (*config.Config, error) {
	return loadConfig(os.Getenv, runtime.GOOS)
}

// loadConfig is the implementation of LoadConfig with getEnv and goos injected so the
// path and environment branches can be tested without touching the real environment.
func loadConfig(getEnv func(string) string, goos string) (*config.Config, error) {
	cfg := &config.Config{}
	path, err := intconfig.GetPath(getEnv, goos)
	if err != nil {
		return nil, err //nolint:wrapcheck // GetPath returns a descriptive error
	}
	// A missing config file is not an error: the returned Config is empty and the
	// environment overrides below still apply.
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat the config file: %w", err)
		}
	} else if err := intconfig.NewReader().Read(cfg, path); err != nil {
		return nil, fmt.Errorf("read the config file: %w", err)
	}
	applyEnvOverrides(cfg, getEnv)
	return cfg, nil
}

// applyEnvOverrides overwrites every Config field that has a corresponding environment
// variable, when that variable is set. Keep the variable names and value semantics in
// sync with the api layer's resolvers (resolveBackendType, resolveMinExpiration,
// openBrowser, and clipboard in internal/api/get.go). SkipAccountPicker has no
// environment variable, so it is left as read from the file.
func applyEnvOverrides(cfg *config.Config, getEnv func(string) string) {
	if v := getEnv(env.Backend); v != "" {
		if cfg.Backend == nil {
			cfg.Backend = &config.Backend{}
		}
		cfg.Backend.Type = v
	}
	// A Go duration string (e.g. "1h"); it is parsed at Get time, not here.
	if v := getEnv(env.MinExpiration); v != "" {
		cfg.MinExpiration = v
	}
	// Any value other than "false" enables the browser open.
	if v := getEnv(env.OpenBrowser); v != "" {
		b := v != "false"
		if cfg.OpenBrowser == nil {
			cfg.OpenBrowser = &config.OpenBrowser{}
		}
		cfg.OpenBrowser.Enable = &b
	}
	// Only "true" enables copying the one-time code to the clipboard.
	if v := getEnv(env.Clipboard); v != "" {
		b := v == "true"
		if cfg.Clipboard == nil {
			cfg.Clipboard = &config.Clipboard{}
		}
		cfg.Clipboard.Enable = &b
	}
}

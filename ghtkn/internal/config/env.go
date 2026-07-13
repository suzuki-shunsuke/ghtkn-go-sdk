package config

import (
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
)

// ApplyEnvOverrides overwrites the Config fields that have a corresponding environment
// variable, when that variable is set:
//
//   - GHTKN_BACKEND -> Backend.Type
//   - GHTKN_MIN_EXPIRATION -> MinExpiration (a Go duration string, parsed later)
//   - GHTKN_OPEN_BROWSER -> OpenBrowser.Enable (any value other than "false" enables it)
//   - GHTKN_CLIPBOARD -> Clipboard.Enable (only "true" enables it)
//
// SkipAccountPicker has no environment variable, so it is left as read from the file.
// This is the single place that maps environment variables onto config fields, shared by
// LoadConfig and the token-retrieval path so their env semantics cannot drift.
func ApplyEnvOverrides(cfg *pubconfig.Config, getEnv func(string) string) {
	if v := getEnv(env.Backend); v != "" {
		if cfg.Backend == nil {
			cfg.Backend = &pubconfig.Backend{}
		}
		cfg.Backend.Type = v
	}
	if v := getEnv(env.MinExpiration); v != "" {
		cfg.MinExpiration = v
	}
	if v := getEnv(env.OpenBrowser); v != "" {
		b := v != "false"
		if cfg.OpenBrowser == nil {
			cfg.OpenBrowser = &pubconfig.OpenBrowser{}
		}
		cfg.OpenBrowser.Enable = &b
	}
	if v := getEnv(env.Clipboard); v != "" {
		b := v == "true"
		if cfg.Clipboard == nil {
			cfg.Clipboard = &pubconfig.Clipboard{}
		}
		cfg.Clipboard.Enable = &b
	}
}

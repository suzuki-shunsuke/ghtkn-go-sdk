package config

import (
	"fmt"
	"strconv"

	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
)

// ApplyEnvOverrides overwrites the Config fields that have a corresponding environment
// variable, when that variable is set:
//
//   - GHTKN_BACKEND -> Backend.Type
//   - GHTKN_MIN_EXPIRATION -> MinExpiration (a Go duration string, parsed later)
//   - GHTKN_OPEN_BROWSER -> OpenBrowser.Enable (a boolean parsed by strconv.ParseBool)
//   - GHTKN_CLIPBOARD -> Clipboard.Enable (a boolean parsed by strconv.ParseBool)
//
// A GHTKN_OPEN_BROWSER or GHTKN_CLIPBOARD value that strconv.ParseBool cannot parse is a
// hard error, so a typo fails fast instead of being silently misinterpreted.
// SkipAccountPicker has no environment variable, so it is left as read from the file.
// This is the single place that maps environment variables onto config fields, shared by
// LoadConfig and the token-retrieval path so their env semantics cannot drift.
func ApplyEnvOverrides(cfg *pubconfig.Config, getEnv func(string) string) error {
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
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("parse %s as a boolean: %w", env.OpenBrowser, err)
		}
		if cfg.OpenBrowser == nil {
			cfg.OpenBrowser = &pubconfig.OpenBrowser{}
		}
		cfg.OpenBrowser.Enable = &b
	}
	if v := getEnv(env.Clipboard); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("parse %s as a boolean: %w", env.Clipboard, err)
		}
		if cfg.Clipboard == nil {
			cfg.Clipboard = &pubconfig.Clipboard{}
		}
		cfg.Clipboard.Enable = &b
	}
	return nil
}

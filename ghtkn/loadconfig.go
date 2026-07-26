package ghtkn

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	intconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

// InputLoadConfig configures LoadConfig.
type InputLoadConfig struct {
	// ConfigFilePath is the path to the configuration file (auto-detected if empty).
	// A non-empty value wins over GHTKN_CONFIG and the default path, which lets a
	// caller honor its own -c/--config flag.
	ConfigFilePath string
}

// LoadConfig finds the ghtkn configuration file (input.ConfigFilePath, then
// GHTKN_CONFIG, then the XDG/OS default path), reads it with the SDK's own YAML
// decoder, and applies every environment override that maps onto a config field, so
// the returned Config reflects the file plus environment. A missing config file yields
// an empty Config with the environment overrides still applied. input may be nil,
// which is the same as an empty ConfigFilePath.
//
// Per-call flag overrides (e.g. -min-expiration, -clipboard) are NOT reflected here;
// those are applied at Get time on top of this value.
func LoadConfig(input *InputLoadConfig) (*config.Config, error) {
	path := ""
	if input != nil {
		path = input.ConfigFilePath
	}
	return loadConfig(os.Getenv, runtime.GOOS, path)
}

// loadConfig is the implementation of LoadConfig with getEnv and goos injected so the
// path and environment branches can be tested without touching the real environment.
// path is the explicitly requested config file path; it is auto-detected when empty.
func loadConfig(getEnv func(string) string, goos, path string) (*config.Config, error) {
	cfg := &config.Config{}
	if path == "" {
		p, err := intconfig.GetPath(getEnv, goos)
		if err != nil {
			return nil, err //nolint:wrapcheck // GetPath returns a descriptive error
		}
		path = p
	}
	// A missing config file is not an error: the returned Config is empty and the
	// environment overrides below still apply. Reader.Read wraps os.Open, so a missing
	// file surfaces as fs.ErrNotExist; reading directly avoids a Stat/Read race.
	if err := intconfig.NewReader().Read(cfg, path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read the config file: %w", err)
	}
	if err := intconfig.ApplyEnvOverrides(cfg, getEnv); err != nil {
		return nil, fmt.Errorf("apply environment overrides: %w", err)
	}
	return cfg, nil
}

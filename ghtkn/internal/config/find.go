package config

import (
	"errors"
	"path/filepath"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
)

// GetPath returns the configuration file path for ghtkn.
// If GHTKN_CONFIG is set, its value is returned as-is, overriding everything else.
// Otherwise it combines the XDG_CONFIG_HOME directory with the ghtkn configuration
// filename; the typical path is $XDG_CONFIG_HOME/ghtkn/ghtkn.yaml.
func GetPath(getEnv func(string) string, goos string) (string, error) {
	if f := getEnv(env.Config); f != "" {
		return f, nil
	}
	if goos == "windows" {
		appData := getEnv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "ghtkn", "ghtkn.yaml"), nil
		}
		return "", errors.New("APPDATA is required on Windows")
	}
	xdgConfigHome := getEnv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "ghtkn", "ghtkn.yaml"), nil
	}
	home := getEnv("HOME")
	if home != "" {
		return filepath.Join(home, ".config", "ghtkn", "ghtkn.yaml"), nil
	}
	return "", errors.New("XDG_CONFIG_HOME or HOME is required on Linux and macOS")
}

package config

import (
	"errors"
	"path/filepath"
)

// GetPath returns the default configuration file path for ghtkn.
// It combines the XDG_CONFIG_HOME directory with the ghtkn configuration filename.
// The typical path is $XDG_CONFIG_HOME/ghtkn/ghtkn.yaml.
func GetPath(getEnv func(string) string, goos string) (string, error) {
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

// Package config provides configuration management for ghtkn.
// It handles reading and validating configuration files for GitHub App authentication.
package config

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure for ghtkn.
// It contains settings for persistence and a list of GitHub Apps.
type Config struct {
	Users []*User `json:"users"`
}

type User struct {
	Login   string `json:"login"`
	Apps    []*App `json:"apps"`
	Default bool   `json:"default,omitempty"`
}

func (u *User) Validate() error {
	if u.Login == "" {
		return errors.New("login is required")
	}
	if len(u.Apps) == 0 {
		return errors.New("apps is required")
	}
	for _, app := range u.Apps {
		if err := app.Validate(); err != nil {
			return fmt.Errorf("app is invalid: %w", slogerr.With(err, "app", app.Name))
		}
	}
	return nil
}

// Validate checks if the Config is valid.
// It ensures the config is not nil and contains at least one app.
// It also validates each app in the configuration.
func (cfg *Config) Validate() error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if len(cfg.Users) == 0 {
		return errors.New("users is required")
	}
	for _, user := range cfg.Users {
		if err := user.Validate(); err != nil {
			return fmt.Errorf("user is invalid: %w", slogerr.With(err, "user", user.Login))
		}
	}
	return nil
}

// App represents a GitHub App configuration.
// Each app must have a unique name and a client ID for authentication.
type App struct {
	Name    string `json:"name"`
	AppID   int    `json:"app_id" yaml:"app_id"`
	Default bool   `json:"default,omitempty"`
}

// Validate checks if the App configuration is valid.
// It ensures both Name and AppID fields are present.
func (app *App) Validate() error {
	if app.Name == "" {
		return errors.New("name is required")
	}
	if app.AppID == 0 {
		return errors.New("app_id is required")
	}
	return nil
}

// Reader handles reading configuration files from the filesystem.
type Reader struct {
	fs afero.Fs
}

// NewReader creates a new configuration Reader with the given filesystem.
func NewReader(fs afero.Fs) *Reader {
	return &Reader{fs: fs}
}

// Read reads and parses a configuration file from the given path.
// It decodes the YAML content into the provided Config struct.
// If configFilePath is empty, it returns nil without reading anything.
func (r *Reader) Read(cfg *Config, configFilePath string) error {
	if configFilePath == "" {
		return nil
	}
	f, err := r.fs.Open(configFilePath)
	if err != nil {
		return fmt.Errorf("open a configuration file: %w", err)
	}
	defer f.Close() //nolint:errcheck
	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("decode a configuration file as YAML: %w", err)
	}
	return nil
}

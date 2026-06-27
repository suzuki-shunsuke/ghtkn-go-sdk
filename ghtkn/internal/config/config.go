// Package config provides configuration management for ghtkn.
// It handles reading and validating configuration files for GitHub App authentication.
// The configuration types themselves live in the public ghtkn/config package.
package config

import (
	"fmt"
	"os"

	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"gopkg.in/yaml.v3"
)

// Reader handles reading configuration files from the filesystem.
type Reader struct{}

// NewReader creates a new configuration Reader.
func NewReader() *Reader {
	return &Reader{}
}

// Read reads and parses a configuration file from the given path.
// It decodes the YAML content into the provided Config struct.
// If configFilePath is empty, it returns nil without reading anything.
func (r *Reader) Read(cfg *pubconfig.Config, configFilePath string) error {
	if configFilePath == "" {
		return nil
	}
	f, err := os.Open(configFilePath)
	if err != nil {
		return fmt.Errorf("open a configuration file: %w", err)
	}
	defer f.Close() //nolint:errcheck
	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("decode a configuration file as YAML: %w", err)
	}
	return nil
}

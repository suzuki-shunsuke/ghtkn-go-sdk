package config_test

import (
	"os"
	"path/filepath"
	"testing"

	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestNewReader(t *testing.T) {
	t.Parallel()

	reader := config.NewReader()

	if reader == nil {
		t.Error("NewReader() returned nil")
	}
}

func TestReader_Read(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name           string
		configPath     string
		configContent  string
		fileExists     bool
		expectedConfig *pubconfig.Config
		wantErr        bool
	}{
		{
			name:           "empty config path",
			configPath:     "",
			configContent:  "",
			fileExists:     false,
			expectedConfig: &pubconfig.Config{},
			wantErr:        false,
		},
		{
			name:       "valid config file",
			configPath: "config.yaml",
			configContent: `apps:
  - name: test-app
    client_id: xxx`,
			fileExists: true,
			expectedConfig: &pubconfig.Config{
				Apps: []*pubconfig.App{
					{
						Name:     "test-app",
						ClientID: "xxx",
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "multiple apps config",
			configPath: "config.yaml",
			configContent: `apps:
  - name: app1
    client_id: xxx
  - name: app2
    client_id: yyy`,
			fileExists: true,
			expectedConfig: &pubconfig.Config{
				Apps: []*pubconfig.App{
					{
						Name:     "app1",
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "file does not exist",
			configPath: "nonexistent.yaml",
			fileExists: false,
			wantErr:    true,
		},
		{
			name:       "invalid YAML",
			configPath: "config.yaml",
			configContent: `invalid yaml:
  - name: test-app
    client_id: xxx
    invalid: [`,
			fileExists: true,
			wantErr:    true,
		},
		{
			name:          "empty config file",
			configPath:    "config.yaml",
			configContent: `apps: []`,
			fileExists:    true,
			expectedConfig: &pubconfig.Config{
				Apps: []*pubconfig.App{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configPath := tt.configPath
			if tt.configPath != "" {
				configPath = filepath.Join(t.TempDir(), tt.configPath)
			}
			if tt.fileExists {
				if err := os.WriteFile(configPath, []byte(tt.configContent), 0o600); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			reader := config.NewReader()
			cfg := &pubconfig.Config{}

			err := reader.Read(cfg, configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reader.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.expectedConfig != nil {
				if len(cfg.Apps) != len(tt.expectedConfig.Apps) {
					t.Errorf("Reader.Read() apps count = %v, want %v", len(cfg.Apps), len(tt.expectedConfig.Apps))
					return
				}

				for i, app := range cfg.Apps {
					expectedApp := tt.expectedConfig.Apps[i]
					if app.Name != expectedApp.Name {
						t.Errorf("Reader.Read() app[%d].Name = %v, want %v", i, app.Name, expectedApp.Name)
					}
					if app.ClientID != expectedApp.ClientID {
						t.Errorf("Reader.Read() app[%d].ClientID = %v, want %v", i, app.ClientID, expectedApp.ClientID)
					}
				}
			}
		})
	}
}

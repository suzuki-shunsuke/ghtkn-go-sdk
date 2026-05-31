package config_test

import (
	"testing"

	"github.com/spf13/afero"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestNewReader(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	reader := config.NewReader(fs)

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
			configPath: "/test/config.yaml",
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
			configPath: "/test/config.yaml",
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
			configPath: "/test/nonexistent.yaml",
			fileExists: false,
			wantErr:    true,
		},
		{
			name:       "invalid YAML",
			configPath: "/test/config.yaml",
			configContent: `invalid yaml:
  - name: test-app
    client_id: xxx
    invalid: [`,
			fileExists: true,
			wantErr:    true,
		},
		{
			name:          "empty config file",
			configPath:    "/test/config.yaml",
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

			fs := afero.NewMemMapFs()
			if tt.fileExists {
				err := afero.WriteFile(fs, tt.configPath, []byte(tt.configContent), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			reader := config.NewReader(fs)
			cfg := &pubconfig.Config{}

			err := reader.Read(cfg, tt.configPath)
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

func TestConfig_SelectApp(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name     string
		config   *pubconfig.Config
		key      string
		owner    string
		expected *pubconfig.App
	}{
		{
			name:     "nil config",
			config:   nil,
			key:      "any-key",
			owner:    "",
			expected: nil,
		},
		{
			name: "empty apps",
			config: &pubconfig.Config{
				Apps: []*pubconfig.App{},
			},
			key:      "any-key",
			owner:    "",
			expected: nil,
		},
		{
			name: "select by owner match",
			config: &pubconfig.Config{
				Apps: []*pubconfig.App{
					{Name: "app1", ClientID: "xxx", GitOwner: "owner1"},
					{Name: "app2", ClientID: "yyy", GitOwner: "owner2"},
					{Name: "app3", ClientID: "zzz", GitOwner: "owner3"},
				},
			},
			key:      "",
			owner:    "owner2",
			expected: &pubconfig.App{Name: "app2", ClientID: "yyy", GitOwner: "owner2"},
		},
		{
			name: "select by key match",
			config: &pubconfig.Config{
				Apps: []*pubconfig.App{
					{Name: "app1", ClientID: "xxx"},
					{Name: "app2", ClientID: "yyy"},
					{Name: "app3", ClientID: "zzz"},
				},
			},
			key:      "app3",
			owner:    "",
			expected: &pubconfig.App{Name: "app3", ClientID: "zzz"},
		},
		{
			name: "owner takes priority over key",
			config: &pubconfig.Config{
				Apps: []*pubconfig.App{
					{Name: "app1", ClientID: "xxx", GitOwner: "owner1"},
					{Name: "app2", ClientID: "yyy", GitOwner: "owner2"},
				},
			},
			key:      "app2",
			owner:    "owner1",
			expected: &pubconfig.App{Name: "app1", ClientID: "xxx", GitOwner: "owner1"},
		},
		{
			name: "return nil when key does not match",
			config: &pubconfig.Config{
				Apps: []*pubconfig.App{
					{Name: "app1", ClientID: "xxx"},
					{Name: "app2", ClientID: "yyy"},
				},
			},
			key:      "nonexistent",
			owner:    "",
			expected: nil,
		},
		{
			name: "select first when both key and owner are empty",
			config: &pubconfig.Config{
				Apps: []*pubconfig.App{
					{Name: "app1", ClientID: "xxx"},
					{Name: "app2", ClientID: "yyy"},
				},
			},
			key:      "",
			owner:    "",
			expected: &pubconfig.App{Name: "app1", ClientID: "xxx"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := config.SelectApp(tt.config, tt.key, tt.owner)

			if tt.expected == nil {
				if got != nil {
					t.Errorf("Config.SelectApp() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("Config.SelectApp() = nil, want %v", tt.expected)
				return
			}

			if got.Name != tt.expected.Name {
				t.Errorf("Config.SelectApp().Name = %v, want %v", got.Name, tt.expected.Name)
			}
			if got.ClientID != tt.expected.ClientID {
				t.Errorf("Config.SelectApp().ClientID = %v, want %v", got.ClientID, tt.expected.ClientID)
			}
			if got.GitOwner != tt.expected.GitOwner {
				t.Errorf("Config.SelectApp().GitOwner = %v, want %v", got.GitOwner, tt.expected.GitOwner)
			}
		})
	}
}

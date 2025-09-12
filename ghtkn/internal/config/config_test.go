package config_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestConfig_Validate(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "test-app",
						ClientID: "xxx",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple apps",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						Default:  true,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty apps",
			config: &config.Config{
				Apps: []*config.App{},
			},
			wantErr: true,
		},
		{
			name: "nil apps",
			config: &config.Config{
				Apps: nil,
			},
			wantErr: true,
		},
		{
			name: "invalid app in config",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "valid-app",
						ClientID: "xxx",
					},
					{
						Name:     "", // invalid - empty name
						ClientID: "yyy",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "multiple invalid apps",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "", // invalid - empty name
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "", // invalid - empty client_id
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewReader(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	reader := config.NewReader(fs)

	if reader == nil {
		t.Error("NewReader() returned nil")
	}
}

func TestApp_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		app     *config.App
		wantErr bool
	}{
		{
			name: "valid app",
			app: &config.App{
				Name:     "test-app",
				ClientID: "xxx",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			app: &config.App{
				Name:     "",
				ClientID: "xxx",
			},
			wantErr: true,
		},
		{
			name: "zero app_id",
			app: &config.App{
				Name:     "test-app",
				ClientID: "",
			},
			wantErr: true,
		},
		{
			name: "both empty",
			app: &config.App{
				Name:     "",
				ClientID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.app.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("App.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReader_Read(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name           string
		configPath     string
		configContent  string
		fileExists     bool
		expectedConfig *config.Config
		wantErr        bool
	}{
		{
			name:           "empty config path",
			configPath:     "",
			configContent:  "",
			fileExists:     false,
			expectedConfig: &config.Config{},
			wantErr:        false,
		},
		{
			name:       "valid config file",
			configPath: "/test/config.yaml",
			configContent: `apps:
  - name: test-app
    client_id: xxx
    default: true`,
			fileExists: true,
			expectedConfig: &config.Config{
				Apps: []*config.App{
					{
						Name:     "test-app",
						ClientID: "xxx",
						Default:  true,
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
    client_id: yyy
    default: true`,
			fileExists: true,
			expectedConfig: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						Default:  true,
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
			expectedConfig: &config.Config{
				Apps: []*config.App{},
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
			cfg := &config.Config{}

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
					if app.Default != expectedApp.Default {
						t.Errorf("Reader.Read() app[%d].Default = %v, want %v", i, app.Default, expectedApp.Default)
					}
				}
			}
		})
	}
}

func TestConfig_SelectApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *config.Config
		key      string
		expected *config.App
	}{
		{
			name:     "nil config",
			config:   nil,
			key:      "any-key",
			expected: nil,
		},
		{
			name: "empty apps",
			config: &config.Config{
				Apps: []*config.App{},
			},
			key:      "any-key",
			expected: nil,
		},
		{
			name: "select by key match",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						Default:  true,
					},
					{
						Name:     "app3",
						ClientID: "zzz",
					},
				},
			},
			key: "app3",
			expected: &config.App{
				Name:     "app3",
				ClientID: "zzz",
			},
		},
		{
			name: "select default when key doesn't match",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						Default:  true,
					},
					{
						Name:     "app3",
						ClientID: "zzz",
					},
				},
			},
			key: "nonexistent",
			expected: &config.App{
				Name:     "app2",
				ClientID: "yyy",
				Default:  true,
			},
		},
		{
			name: "select default when key is empty",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						Default:  true,
					},
				},
			},
			key: "",
			expected: &config.App{
				Name:     "app2",
				ClientID: "yyy",
				Default:  true,
			},
		},
		{
			name: "select first when no default and key doesn't match",
			config: &config.Config{
				Apps: []*config.App{
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
			key: "nonexistent",
			expected: &config.App{
				Name:     "app1",
				ClientID: "xxx",
			},
		},
		{
			name: "select first when no default and key is empty",
			config: &config.Config{
				Apps: []*config.App{
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
			key: "",
			expected: &config.App{
				Name:     "app1",
				ClientID: "xxx",
			},
		},
		{
			name: "key match takes priority over default",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						Default:  true,
					},
				},
			},
			key: "app1",
			expected: &config.App{
				Name:     "app1",
				ClientID: "xxx",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.config.SelectApp(tt.key)

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
			if got.Default != tt.expected.Default {
				t.Errorf("Config.SelectApp().Default = %v, want %v", got.Default, tt.expected.Default)
			}
		})
	}
}

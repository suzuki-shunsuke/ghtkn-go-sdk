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
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "empty users",
			config:  &config.Config{},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				Users: []*config.User{
					{
						Login: "testuser",
						Apps: []*config.App{
							{
								Name:  "test-app",
								AppID: 123,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid user",
			config: &config.Config{
				Users: []*config.User{
					{
						Login: "",
						Apps: []*config.App{
							{
								Name:  "test-app",
								AppID: 123,
							},
						},
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
				Name:  "test-app",
				AppID: 123,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			app: &config.App{
				Name:  "",
				AppID: 123,
			},
			wantErr: true,
		},
		{
			name: "zero app_id",
			app: &config.App{
				Name:  "test-app",
				AppID: 0,
			},
			wantErr: true,
		},
		{
			name: "both empty",
			app: &config.App{
				Name:  "",
				AppID: 0,
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

	validYAML := `users:
- login: testuser
  apps:
  - name: test-app
    app_id: 123
`

	invalidYAML := `invalid: yaml: content:
  - [unclosed
`

	tests := []struct {
		name           string
		configFilePath string
		fileContent    string
		fileExists     bool
		wantErr        bool
		expectedUsers  int
	}{
		{
			name:           "empty config path",
			configFilePath: "",
			wantErr:        false,
			expectedUsers:  0,
		},
		{
			name:           "file not found",
			configFilePath: "nonexistent.yaml",
			fileExists:     false,
			wantErr:        true,
		},
		{
			name:           "valid YAML",
			configFilePath: "config.yaml",
			fileContent:    validYAML,
			fileExists:     true,
			wantErr:        false,
			expectedUsers:  1,
		},
		{
			name:           "invalid YAML",
			configFilePath: "invalid.yaml",
			fileContent:    invalidYAML,
			fileExists:     true,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fs := afero.NewMemMapFs()
			if tt.fileExists {
				if err := afero.WriteFile(fs, tt.configFilePath, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			reader := config.NewReader(fs)
			cfg := &config.Config{}

			err := reader.Read(cfg, tt.configFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reader.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(cfg.Users) != tt.expectedUsers {
					t.Errorf("Reader.Read() users count = %d, expected %d", len(cfg.Users), tt.expectedUsers)
				}

				if tt.expectedUsers > 0 {
					if cfg.Users[0].Login != "testuser" {
						t.Errorf("Reader.Read() user login = %s, expected testuser", cfg.Users[0].Login)
					}
					if len(cfg.Users[0].Apps) != 1 {
						t.Errorf("Reader.Read() apps count = %d, expected 1", len(cfg.Users[0].Apps))
					}
					if cfg.Users[0].Apps[0].Name != "test-app" {
						t.Errorf("Reader.Read() app name = %s, expected test-app", cfg.Users[0].Apps[0].Name)
					}
					if cfg.Users[0].Apps[0].AppID != 123 {
						t.Errorf("Reader.Read() app ID = %d, expected 123", cfg.Users[0].Apps[0].AppID)
					}
				}
			}
		})
	}
}

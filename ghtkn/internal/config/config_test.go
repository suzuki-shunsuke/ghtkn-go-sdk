package config_test

import (
	"strings"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestConfig_Validate(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "config is required",
		},
		{
			name: "empty apps",
			cfg: &config.Config{
				Users: []*config.User{
					{
						Login: "octocat",
						Apps:  []*config.App{},
					},
				},
			},
			wantErr: true,
			errMsg:  "apps is required",
		},
		{
			name: "valid config",
			cfg: &config.Config{
				Users: []*config.User{
					{
						Login: "octocat",
						Apps: []*config.App{
							{
								Name:    "test-app",
								Default: true,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid app",
			cfg: &config.Config{
				Users: []*config.User{
					{
						Login: "octocat",
						Apps: []*config.App{
							{
								Name: "test-app",
								// Missing AppID
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "app_id is required",
		},
		{
			name: "multiple apps with one invalid",
			cfg: &config.Config{
				Users: []*config.User{
					{
						Login: "octocat",
						Apps: []*config.App{
							{
								Name:  "app1",
								AppID: 12345,
							},
							{
								Name: "app2",
								// Missing AppID
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "app_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.cfg.Validate(); err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message does not contain expected string\ngot: %v\nwant substring: %v", err.Error(), tt.errMsg)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}

func TestApp_Validate(t *testing.T) {
	t.Parallel()
}

func TestReader_Read(t *testing.T) { //nolint:funlen
	t.Parallel()
}

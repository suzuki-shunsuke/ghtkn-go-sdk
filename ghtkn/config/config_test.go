package config_test

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
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
			name: "duplicate app names",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "duplicate-app",
						ClientID: "xxx",
					},
					{
						Name:     "duplicate-app",
						ClientID: "yyy",
					},
				},
			},
			wantErr: true,
		},
		{
			// Two apps sharing a client id are two names for one stored token: revoking
			// or minting for one would silently do it for the other.
			name: "duplicate client ids",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "same-client-id",
						GitOwner: "owner1",
					},
					{
						Name:     "app2",
						ClientID: "same-client-id",
						GitOwner: "owner2",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate git owners",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
						GitOwner: "same-owner",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						GitOwner: "same-owner",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid config with unique git owners",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
						GitOwner: "owner1",
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						GitOwner: "owner2",
					},
					{
						Name:     "app3",
						ClientID: "zzz",
					},
				},
			},
			wantErr: false,
		},
		{
			// The owner is declared by git_owner in one app and by git_owners in the
			// other, so the uniqueness check must span both fields.
			name: "duplicate git owners across git_owner and git_owners",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "xxx",
						GitOwner: "same-owner",
					},
					{
						Name:      "app2",
						ClientID:  "yyy",
						GitOwners: []string{"owner2", "same-owner"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate git owners across git_owners",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:      "app1",
						ClientID:  "xxx",
						GitOwners: []string{"owner1", "same-owner"},
					},
					{
						Name:      "app2",
						ClientID:  "yyy",
						GitOwners: []string{"same-owner", "owner2"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid config with git_owners",
			config: &config.Config{
				Apps: []*config.App{
					{
						Name:      "enterprise",
						ClientID:  "xxx",
						GitOwners: []string{"owner1", "owner2"},
					},
					{
						Name:     "app2",
						ClientID: "yyy",
						GitOwner: "owner3",
					},
				},
			},
			wantErr: false,
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
		{
			name: "valid app with git_owners",
			app: &config.App{
				Name:      "test-app",
				ClientID:  "xxx",
				GitOwners: []string{"owner1", "owner2"},
			},
			wantErr: false,
		},
		{
			name: "git_owner and git_owners are mutually exclusive",
			app: &config.App{
				Name:      "test-app",
				ClientID:  "xxx",
				GitOwner:  "owner1",
				GitOwners: []string{"owner2"},
			},
			wantErr: true,
		},
		{
			name: "empty git owner in git_owners",
			app: &config.App{
				Name:      "test-app",
				ClientID:  "xxx",
				GitOwners: []string{"owner1", ""},
			},
			wantErr: true,
		},
		{
			name: "duplicate git owners in git_owners",
			app: &config.App{
				Name:      "test-app",
				ClientID:  "xxx",
				GitOwners: []string{"owner1", "owner1"},
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

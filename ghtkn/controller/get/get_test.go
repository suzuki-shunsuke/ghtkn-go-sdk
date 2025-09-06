//nolint:forcetypeassert,funlen,maintidx
package get_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/controller/get"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
)

func TestController_Run(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name            string
		setupInput      func() *get.Input
		wantErr         bool
		wantAccessToken *keyring.AccessToken
		checkKeyring    bool
	}{
		{
			name: "successful token creation without persistence",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "test-token-123",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "test-user",
					}, nil),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: false,
			wantAccessToken: &keyring.AccessToken{
				AccessToken:    "test-token-123",
				ExpirationDate: keyring.FormatDate(futureTime),
				Login:          "test-user",
			},
		},
		{
			name: "successful token retrieval from keyring",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: true,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "cached-token",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "cached-user",
					}, nil),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: false,
			wantAccessToken: &keyring.AccessToken{
				AccessToken:    "cached-token",
				ExpirationDate: keyring.FormatDate(futureTime),
				Login:          "cached-user",
			},
		},
		{
			name: "expired token in keyring triggers new token creation",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: true,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "test-user",
					}, nil),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: false,
			wantAccessToken: &keyring.AccessToken{
				AccessToken:    "new-token",
				ExpirationDate: keyring.FormatDate(futureTime),
				Login:          "test-user",
			},
		},
		{
			name: "config read error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						err: errors.New("config read error"),
					},
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "invalid config",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Apps: []*config.App{}, // No apps configured
						},
					},
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "token creation error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env:          &config.Env{App: "test-app"},
					TokenManager: api.New(api.NewInput()),
					// AppTokenClient: &mockAppTokenClient{
					// 	err: errors.New("token creation failed"),
					// },
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "GitHub API GetUser error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env:          &config.Env{App: "test-app"},
					TokenManager: api.New(api.NewInput()),
					// AppTokenClient: &mockAppTokenClient{
					// 	token: &apptoken.AccessToken{
					// 		AccessToken:    "test-token-123",
					// 		ExpirationDate: keyring.FormatDate(futureTime),
					// 	},
					// },
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "cached token without login and GitHub API error",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: true,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env:          &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(nil, errors.New("GitHub API rate limit error")),
					Stdout:       &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "JSON output format",
			setupInput: func() *get.Input {
				return &get.Input{
					ConfigFilePath: "test.yaml",
					OutputFormat:   "json",
					FS:             afero.NewMemMapFs(),
					ConfigReader: &mockConfigReader{
						cfg: &config.Config{
							Persist: false,
							Apps: []*config.App{
								{
									Name:     "test-app",
									ClientID: "test-client-id",
								},
							},
						},
					},
					Env: &config.Env{App: "test-app"},
					TokenManager: api.NewMockTokenManager(&keyring.AccessToken{
						AccessToken:    "test-token-json",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "test-user",
					}, nil),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: false,
			wantAccessToken: &keyring.AccessToken{
				AccessToken:    "test-token-json",
				ExpirationDate: keyring.FormatDate(futureTime),
				Login:          "test-user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			controller := get.New(input)
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			token, err := controller.Run(ctx, logger)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Error("Run() unexpected error:", err)
			}
			if tt.wantErr {
				t.Error("Run() expected error but got none")
			}
			if diff := cmp.Diff(tt.wantAccessToken, token); diff != "" {
				t.Error(diff)
			}
		})
	}
}

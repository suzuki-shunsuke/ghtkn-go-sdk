//nolint:funlen
package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/github"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

func newMockInput() *Input {
	return &Input{
		DeviceFlow: &mockDeviceFlow{
			token: &deviceflow.AccessToken{
				AccessToken:    "test-token",
				ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
		Backend:      &mockKeyring{},
		Logger:       log.NewLogger(),
		ConfigReader: &mockConfigReader{},
		Getenv:       func(key string) string { return "" },
		NewGitHub:    mockNewGitHub,
	}
}

type mockKeyring struct {
	token *pubapi.AccessToken
	err   error
}

func (m *mockKeyring) Get(_ context.Context, _ string) (*pubapi.AccessToken, error) {
	return m.token, m.err
}

func (m *mockKeyring) Set(_ context.Context, _ string, _ *pubapi.AccessToken) error {
	return m.err
}

type mockConfigReader struct {
	err error
}

func (m *mockConfigReader) Read(cfg *pubconfig.Config, configFilePath string) error {
	if m.err != nil {
		return m.err
	}
	cfg.Apps = []*pubconfig.App{
		{
			Name:     "test-app",
			ClientID: "xxx",
		},
	}
	return nil
}

type mockGitHub struct {
	user *github.User
	err  error
}

func (m *mockGitHub) GetUser(_ context.Context) (*github.User, error) {
	return m.user, m.err
}

func mockNewGitHub(_ context.Context, _ string) (gitHub, error) {
	return &mockGitHub{
		user: &github.User{
			Login: "test-user",
		},
	}, nil
}

func TestTokenManager_Get(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setupInput func() *Input
		wantErr    bool
		wantToken  *pubapi.AccessToken
		input      *pubapi.InputGet
	}{
		{
			name: "successful token retrieval from keyring",
			setupInput: func() *Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{
					token: &deviceflow.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: futureTime,
					},
				}
				input.Backend = &mockKeyring{
					token: &pubapi.AccessToken{
						AccessToken:    "cached-token",
						ExpirationDate: futureTime,
					},
				}
				input.Now = func() time.Time {
					return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
				}
				return input
			},
			input:   &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"},
			wantErr: false,
			wantToken: &pubapi.AccessToken{
				AccessToken:    "cached-token",
				ExpirationDate: futureTime,
				Login:          "test-user",
			},
		},
		{
			name: "expired token in keyring triggers new token creation",
			setupInput: func() *Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{
					token: &deviceflow.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
					},
				}
				input.Backend = &mockKeyring{
					token: &pubapi.AccessToken{
						AccessToken:    "expired-token",
						ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				}
				input.Now = func() time.Time {
					return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
				}
				return input
			},
			input:   &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"},
			wantErr: false,
			wantToken: &pubapi.AccessToken{
				AccessToken:    "new-token",
				ExpirationDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				Login:          "test-user",
			},
		},
		{
			name: "token creation error",
			setupInput: func() *Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{
					err: errors.New("token creation failed"),
				}
				input.Backend = &mockKeyring{}
				input.Now = func() time.Time {
					return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
				}
				return input
			},
			input:   &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			tm := New(input)
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			token, _, err := tm.Get(t.Context(), logger, tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Error(err)
				}
				return
			}
			if tt.wantErr {
				t.Error("expected error but got nil")
				return
			}
			if diff := cmp.Diff(tt.wantToken, token); diff != "" {
				t.Error(diff)
			}
		})
	}
}

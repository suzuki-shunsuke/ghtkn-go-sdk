//nolint:funlen
package api_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

func newMockInput() *api.Input {
	return &api.Input{
		DeviceFlow: &mockDeviceFlow{
			token: &deviceflow.AccessToken{
				AccessToken:    "test-token",
				ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
		Keyring: keyring.New(&keyring.Input{}),
		Logger:  log.NewLogger(),
	}
}

type mockKeyring struct {
	token *keyring.AccessToken
	err   error
}

func (m *mockKeyring) Get(_ string, _ string) (*keyring.AccessToken, error) {
	return m.token, m.err
}

func (m *mockKeyring) Set(_ string, _ string, _ *keyring.AccessToken) error {
	return m.err
}

func TestTokenManager_Get(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setupInput func() *api.Input
		wantErr    bool
		wantToken  *keyring.AccessToken
		input      *api.InputGet
	}{
		{
			name: "successful token creation without persistence",
			setupInput: func() *api.Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{
					token: &deviceflow.AccessToken{
						AccessToken:    "test-token-123",
						ExpirationDate: futureTime,
					},
				}
				input.Keyring = &mockKeyring{}
				input.Now = func() time.Time {
					return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
				}
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: false,
			},
			wantErr: false,
			wantToken: &keyring.AccessToken{
				AccessToken:    "test-token-123",
				ExpirationDate: futureTime,
			},
		},
		{
			name: "successful token retrieval from keyring",
			setupInput: func() *api.Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{
					token: &deviceflow.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: futureTime,
					},
				}
				input.Keyring = &mockKeyring{
					token: &keyring.AccessToken{
						AccessToken:    "cached-token",
						ExpirationDate: futureTime,
						Login:          "cached-user",
					},
				}
				input.Now = func() time.Time {
					return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
				}
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: false,
			wantToken: &keyring.AccessToken{
				AccessToken:    "cached-token",
				ExpirationDate: futureTime,
				Login:          "cached-user",
			},
		},
		{
			name: "expired token in keyring triggers new token creation",
			setupInput: func() *api.Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{
					token: &deviceflow.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
					},
				}
				input.Keyring = &mockKeyring{
					token: &keyring.AccessToken{
						AccessToken:    "expired-token",
						ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				}
				input.Now = func() time.Time {
					return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
				}
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: false,
			wantToken: &keyring.AccessToken{
				AccessToken:    "new-token",
				ExpirationDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "token creation error",
			setupInput: func() *api.Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{
					err: errors.New("token creation failed"),
				}
				input.Keyring = &mockKeyring{}
				input.Now = func() time.Time {
					return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
				}
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			tm := api.New(input)
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

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
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/apptoken"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
)

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

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
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
				input := api.NewMockInput()
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "test-token-123",
						ExpirationDate: futureTime,
					},
				}
				input.Keyring = &mockKeyring{}
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
				Login:          "test-user",
			},
		},
		{
			name: "successful token retrieval from keyring",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
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
				expiredTime := fixedTime.Add(30 * time.Minute)
				input := api.NewMockInput()
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: futureTime,
					},
				}
				input.Keyring = &mockKeyring{
					token: &keyring.AccessToken{
						AccessToken:    "expired-token",
						ExpirationDate: expiredTime,
					},
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
				ExpirationDate: futureTime,
				Login:          "test-user",
			},
		},
		{
			name: "token creation error",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.AppTokenClient = &mockAppTokenClient{
					err: errors.New("token creation failed"),
				}
				input.Keyring = &mockKeyring{}
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

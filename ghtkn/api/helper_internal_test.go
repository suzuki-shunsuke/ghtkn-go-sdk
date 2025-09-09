//nolint:funlen,revive
package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

type testAppTokenClient struct {
	token *deviceflow.AccessToken
	err   error
}

func (m *testAppTokenClient) Create(_ context.Context, logger *slog.Logger, clientID string) (*deviceflow.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func (m *testAppTokenClient) SetLogger(_ *log.Logger) {
}

type testKeyring struct {
	tokens map[string]*keyring.AccessToken
	getErr error
	setErr error
}

func (m *testKeyring) Get(service, key string) (*keyring.AccessToken, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.tokens[service+":"+key], nil
}

func (m *testKeyring) Set(service, key string, token *keyring.AccessToken) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.tokens == nil {
		m.tokens = make(map[string]*keyring.AccessToken)
	}
	m.tokens[service+":"+key] = token
	return nil
}

func TestTokenManager_checkExpired(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name          string
		exDate        time.Time
		minExpiration time.Duration
		now           time.Time
		want          bool
	}{
		{
			name:          "not expired - future date",
			exDate:        fixedTime.Add(2 * time.Hour),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          false,
		},
		{
			name:          "expired - within min expiration",
			exDate:        fixedTime.Add(30 * time.Minute),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          true,
		},
		{
			name:          "expired - past date",
			exDate:        fixedTime.Add(-time.Hour),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          true,
		},
		{
			name:          "exactly at threshold",
			exDate:        fixedTime.Add(time.Hour),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &Input{
				Now: func() time.Time { return tt.now },
			}
			tm := &TokenManager{input: input}

			got := tm.checkExpired(tt.exDate, tt.minExpiration)
			if got != tt.want {
				t.Errorf("checkExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_createToken(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		clientID string
		client   AppTokenClient
		want     *keyring.AccessToken
		wantErr  bool
	}{
		{
			name:     "successful token creation",
			clientID: "test-client-id",
			client: &testAppTokenClient{
				token: &deviceflow.AccessToken{
					AccessToken:    "new-token",
					ExpirationDate: futureTime,
				},
			},
			want: &keyring.AccessToken{
				AccessToken:    "new-token",
				ExpirationDate: futureTime,
			},
			wantErr: false,
		},
		{
			name:     "token creation error",
			clientID: "test-client-id",
			client: &testAppTokenClient{
				err: errors.New("creation failed"),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &Input{
				AppTokenClient: tt.client,
			}
			tm := &TokenManager{input: input}

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			got, err := tm.createToken(t.Context(), logger, tt.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("createToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.AccessToken != tt.want.AccessToken || got.ExpirationDate != tt.want.ExpirationDate {
					t.Errorf("createToken() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestTokenManager_getAccessTokenFromKeyring(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	expiredTime := fixedTime.Add(30 * time.Minute)

	tests := []struct {
		name          string
		clientID      string
		keyring       Keyring
		minExpiration time.Duration
		now           time.Time
		want          *keyring.AccessToken
	}{
		{
			name:     "valid token from keyring",
			clientID: "test-client-id",
			keyring: &testKeyring{
				tokens: map[string]*keyring.AccessToken{
					keyring.DefaultServiceKey + ":test-client-id": {
						AccessToken:    "cached-token",
						ExpirationDate: futureTime,
					},
				},
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want: &keyring.AccessToken{
				AccessToken:    "cached-token",
				ExpirationDate: futureTime,
			},
		},
		{
			name:     "expired token in keyring",
			clientID: "test-client-id",
			keyring: &testKeyring{
				tokens: map[string]*keyring.AccessToken{
					keyring.DefaultServiceKey + ":test-client-id": {
						AccessToken:    "expired-token",
						ExpirationDate: expiredTime,
					},
				},
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          nil,
		},
		{
			name:     "token not found in keyring",
			clientID: "test-client-id",
			keyring: &testKeyring{
				tokens: map[string]*keyring.AccessToken{},
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          nil,
		},
		{
			name:     "keyring error",
			clientID: "test-client-id",
			keyring: &testKeyring{
				getErr: errors.New("keyring error"),
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &Input{
				Keyring: tt.keyring,
				Now:     func() time.Time { return tt.now },
				Logger:  log.NewLogger(),
			}
			tm := &TokenManager{input: input}

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			got := tm.getAccessTokenFromKeyring(logger, keyring.DefaultServiceKey, tt.clientID, tt.minExpiration)
			if got != nil {
				if got.AccessToken != tt.want.AccessToken || got.ExpirationDate != tt.want.ExpirationDate {
					t.Errorf("getAccessTokenFromKeyring() = %v, want %v", got, tt.want)
				}
			}
			if got == nil && tt.want != nil {
				t.Error("getAccessTokenFromKeyring() returned nil, expected token")
			}
		})
	}
}

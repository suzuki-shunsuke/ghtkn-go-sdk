//nolint:funlen,revive
package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

type testDeviceFlow struct {
	token *deviceflow.AccessToken
	err   error
}

func (m *testDeviceFlow) Create(_ context.Context, logger *slog.Logger, clientID string) (*deviceflow.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func (m *testDeviceFlow) SetLogger(_ *log.Logger) {}

func (m *testDeviceFlow) SetDeviceCodeUI(_ deviceflow.DeviceCodeUI) {}

func (m *testDeviceFlow) SetBrowser(_ deviceflow.Browser) {}

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
		client   DeviceFlow
		want     *keyring.AccessToken
		wantErr  bool
	}{
		{
			name:     "successful token creation",
			clientID: "test-client-id",
			client: &testDeviceFlow{
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
			client: &testDeviceFlow{
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
				DeviceFlow: tt.client,
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

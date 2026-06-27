//nolint:funlen,revive
package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

type testDeviceFlow struct {
	token *deviceflow.AccessToken
	err   error
}

func (m *testDeviceFlow) Create(_ context.Context, logger *slog.Logger, input *deviceflow.InputCreate) (*deviceflow.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func (m *testDeviceFlow) SetLogger(_ *publog.Logger) {}

func (m *testDeviceFlow) SetOnetimeCodeUI(_ pubdeviceflow.OnetimeCodeUI) {}

func (m *testDeviceFlow) SetBrowser(_ pubdeviceflow.Browser) {}

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
		client   deviceFlow
		want     *pubapi.AccessToken
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
			want: &pubapi.AccessToken{
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
				Getenv:     func(string) string { return "" },
			}
			tm := &TokenManager{input: input}

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			got, err := tm.createToken(t.Context(), logger, &deviceflow.InputCreate{ClientID: tt.clientID}, true)
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

func TestController_createToken_disableDeviceFlow(t *testing.T) {
	t.Parallel()

	input := &Input{
		DeviceFlow: &testDeviceFlow{
			token: &deviceflow.AccessToken{
				AccessToken:    "should-not-be-used",
				ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
	}
	tm := &TokenManager{input: input}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	got, err := tm.createToken(t.Context(), logger, &deviceflow.InputCreate{ClientID: "test-client-id"}, false)
	if !errors.Is(err, pubapi.ErrDisableDeviceFlow) {
		t.Errorf("createToken() error = %v, want ErrDisableDeviceFlow", err)
	}
	if got != nil {
		t.Errorf("createToken() = %v, want nil", got)
	}
}

func TestEnableDeviceFlow(t *testing.T) {
	t.Parallel()
	ptr := func(b bool) *bool { return &b }
	data := []struct {
		name     string
		override *bool
		env      string
		cfg      *bool // device_flow.enable in the config file
		want     bool
	}{
		{name: "default enabled when all unset", override: nil, env: "", cfg: nil, want: true},
		{name: "env false disables", override: nil, env: "false", want: false},
		{name: "env true enables", override: nil, env: "true", want: true},
		{name: "override true beats env false", override: ptr(true), env: "false", want: true},
		{name: "override false beats env true", override: ptr(false), env: "true", want: false},
		{name: "config false disables when override and env unset", override: nil, env: "", cfg: ptr(false), want: false},
		{name: "config true enables when override and env unset", override: nil, env: "", cfg: ptr(true), want: true},
		{name: "env beats config", override: nil, env: "false", cfg: ptr(true), want: false},
		{name: "override beats config", override: ptr(false), env: "", cfg: ptr(true), want: false},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(k string) string {
				if k == "GHTKN_ENABLE_DEVICE_FLOW" {
					return d.env
				}
				return ""
			}
			var cfg *pubconfig.DeviceFlow
			if d.cfg != nil {
				cfg = &pubconfig.DeviceFlow{Enable: d.cfg}
			}
			if got := enableDeviceFlow(d.override, cfg, getEnv); got != d.want {
				t.Errorf("enableDeviceFlow = %v, want %v", got, d.want)
			}
		})
	}
}

func TestResolveMinExpiration(t *testing.T) {
	t.Parallel()
	ptr := func(d time.Duration) *time.Duration { return &d }
	data := []struct {
		name     string
		override *time.Duration
		env      string
		cfg      string // min_expiration in the config file
		want     time.Duration
		wantErr  bool
	}{
		{name: "default zero when all unset", want: 0},
		{name: "override wins", override: ptr(time.Hour), env: "30m", cfg: "10m", want: time.Hour},
		{name: "override zero beats config", override: ptr(0), env: "", cfg: "1h", want: 0},
		{name: "env when override unset", env: "30m", cfg: "10m", want: 30 * time.Minute},
		{name: "config when override and env unset", cfg: "10m", want: 10 * time.Minute},
		{name: "invalid env errors", env: "nope", wantErr: true},
		{name: "invalid config errors", cfg: "nope", wantErr: true},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(k string) string {
				if k == "GHTKN_MIN_EXPIRATION" {
					return d.env
				}
				return ""
			}
			got, err := resolveMinExpiration(d.override, d.cfg, getEnv)
			if d.wantErr {
				if err == nil {
					t.Fatal("resolveMinExpiration: expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveMinExpiration: unexpected error: %v", err)
			}
			if got != d.want {
				t.Errorf("resolveMinExpiration = %v, want %v", got, d.want)
			}
		})
	}
}

func TestSkipAccountPicker(t *testing.T) {
	t.Parallel()
	ptr := func(b bool) *bool { return &b }
	data := []struct {
		name string
		cfg  *bool
		want bool
	}{
		{name: "default skipped when unset", cfg: nil, want: true},
		{name: "explicit true skips", cfg: ptr(true), want: true},
		{name: "explicit false shows picker", cfg: ptr(false), want: false},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			if got := skipAccountPicker(d.cfg); got != d.want {
				t.Errorf("skipAccountPicker = %v, want %v", got, d.want)
			}
		})
	}
}

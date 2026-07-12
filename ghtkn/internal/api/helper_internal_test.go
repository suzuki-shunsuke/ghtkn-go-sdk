//nolint:funlen,revive
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
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

type testDeviceFlow struct {
	token      *deviceflow.AccessToken
	err        error
	showCalled bool
	showErr    error
}

func (m *testDeviceFlow) Create(_ context.Context, logger *slog.Logger, input *deviceflow.InputCreate) (*deviceflow.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func (m *testDeviceFlow) Show(_ context.Context, _ *slog.Logger, _ *deviceflow.InputCreate, _ *pubdeviceflow.DeviceCodeResponse) error {
	m.showCalled = true
	return m.showErr
}

func (m *testDeviceFlow) SetLogger(_ *publog.Logger) {}

func (m *testDeviceFlow) SetOnetimeCodeUI(_ pubdeviceflow.OnetimeCodeUI) {}

func (m *testDeviceFlow) SetBrowser(_ pubdeviceflow.Browser) {}

func (m *testDeviceFlow) SetCopyOnetimeCodeToClipboard(_ pubdeviceflow.CopyTextToClipboard) {}

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

			got, _, err := tm.createToken(t.Context(), logger, &mockKeyring{}, 0, &deviceflow.InputCreate{ClientID: tt.clientID}, true)
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

	got, _, err := tm.createToken(t.Context(), logger, &mockKeyring{}, 0, &deviceflow.InputCreate{ClientID: "test-client-id"}, false)
	if !errors.Is(err, pubapi.ErrDisableDeviceFlow) {
		t.Errorf("createToken() error = %v, want ErrDisableDeviceFlow", err)
	}
	if got != nil {
		t.Errorf("createToken() = %v, want nil", got)
	}
}

// agentBackend is a Backend that runs the device flow itself (like the agent):
// SupportsDeviceFlow returns true and token creation is driven through
// GetActive/BeginDeviceFlow/PollDeviceFlow. It records whether Set was called so
// tests can assert the token is not re-stored by the caller.
type agentBackend struct {
	active     *pubapi.AccessToken            // token returned by GetActive (nil => none active)
	begun      *pubapi.AccessToken            // token returned by BeginDeviceFlow (nil => a flow starts)
	deviceCode *pubdeviceflow.DeviceCodeResponse
	polled     *pubapi.AccessToken
	setCalled  bool
	// revokeFailed and cleanupFailed are returned by RevokeTokens, and revokeErr is
	// its transport error. revoked records the client IDs passed to RevokeTokens.
	revokeFailed  []string
	cleanupFailed []string
	revokeErr     error
	revoked       []string // client IDs passed to RevokeTokens, in order
}

func (b *agentBackend) Get(_ context.Context, _ string) (*pubapi.AccessToken, error) {
	return nil, nil
}

func (b *agentBackend) Set(_ context.Context, _ string, _ *pubapi.AccessToken) error {
	b.setCalled = true
	return nil
}

func (b *agentBackend) Delete(_ context.Context, _ string) error { return nil }

func (b *agentBackend) SupportsDeviceFlow() bool { return true }

func (b *agentBackend) GetActive(_ context.Context, _ string, _ time.Duration) (*pubapi.AccessToken, error) {
	return b.active, nil
}

func (b *agentBackend) BeginDeviceFlow(_ context.Context, _ string, _ time.Duration) (*pubapi.AccessToken, *pubdeviceflow.DeviceCodeResponse, error) {
	return b.begun, b.deviceCode, nil
}

func (b *agentBackend) PollDeviceFlow(_ context.Context, _ string, _ time.Duration) (*pubapi.AccessToken, error) {
	return b.polled, nil
}

func (b *agentBackend) RevokeTokens(_ context.Context, clientIDs []string) (revokeFailed, cleanupFailed []string, err error) {
	b.revoked = append(b.revoked, clientIDs...)
	return b.revokeFailed, b.cleanupFailed, b.revokeErr
}

// TestTokenManager_getOrCreateToken_agentDeviceFlow verifies that when the
// backend runs the device flow itself, getOrCreateToken begins the flow on the
// backend, shows the one-time code, polls the backend for the minted token, and
// reports changed=false so the caller does not re-store it.
func TestTokenManager_getOrCreateToken_agentDeviceFlow(t *testing.T) {
	t.Parallel()

	polled := &pubapi.AccessToken{
		AccessToken:    "agent-minted-token",
		ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	backend := &agentBackend{
		deviceCode: &pubdeviceflow.DeviceCodeResponse{UserCode: "ABCD-1234"},
		polled:     polled,
	}
	df := &mockDeviceFlow{}
	input := &Input{
		DeviceFlow: df,
		Backend:    backend,
		Logger:     log.NewLogger(),
		Getenv:     func(string) string { return "" },
	}
	tm := &TokenManager{input: input}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	token, changed, err := tm.getOrCreateToken(t.Context(), logger, &inputGetOrCreateToken{
		App:              &pubconfig.App{Name: "test-app", ClientID: "cid"},
		Backend:          backend,
		EnableDeviceFlow: true,
	})
	if err != nil {
		t.Fatalf("getOrCreateToken() error = %v", err)
	}
	if !df.showCalled {
		t.Error("deviceFlow.Show was not invoked")
	}
	if changed {
		t.Error("changed = true, want false (the agent already stored the token)")
	}
	if backend.setCalled {
		t.Error("backend.Set was called, want it not called")
	}
	if diff := cmp.Diff(polled, token); diff != "" {
		t.Errorf("token mismatch (-want +got):\n%s", diff)
	}
}

// TestTokenManager_getAccessTokenFromBackend_agentActive verifies that for a
// backend that owns the token lifecycle, getOrCreateToken returns the token from
// GetActive without re-fetching via Get, and does not run the device flow.
func TestTokenManager_getAccessTokenFromBackend_agentActive(t *testing.T) {
	t.Parallel()

	active := &pubapi.AccessToken{
		AccessToken:    "agent-active-token",
		ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	backend := &agentBackend{active: active}
	df := &mockDeviceFlow{}
	input := &Input{
		DeviceFlow: df,
		Backend:    backend,
		Logger:     log.NewLogger(),
		Getenv:     func(string) string { return "" },
	}
	tm := &TokenManager{input: input}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	token, changed, err := tm.getOrCreateToken(t.Context(), logger, &inputGetOrCreateToken{
		App:              &pubconfig.App{Name: "test-app", ClientID: "cid"},
		Backend:          backend,
		EnableDeviceFlow: true,
	})
	if err != nil {
		t.Fatalf("getOrCreateToken() error = %v", err)
	}
	if changed {
		t.Error("changed = true, want false (an active token already existed)")
	}
	if df.showCalled {
		t.Error("deviceFlow.Show was invoked, want it not called")
	}
	if diff := cmp.Diff(active, token); diff != "" {
		t.Errorf("token mismatch (-want +got):\n%s", diff)
	}
}

// TestTokenManager_getOrCreateToken_agentNoActive verifies that when GetActive
// returns nil the flow falls through to createToken (which begins the device
// flow on the backend).
func TestTokenManager_getOrCreateToken_agentNoActive(t *testing.T) {
	t.Parallel()

	polled := &pubapi.AccessToken{
		AccessToken:    "agent-minted-token",
		ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	backend := &agentBackend{
		active:     nil,
		deviceCode: &pubdeviceflow.DeviceCodeResponse{UserCode: "ABCD-1234"},
		polled:     polled,
	}
	df := &mockDeviceFlow{}
	input := &Input{
		DeviceFlow: df,
		Backend:    backend,
		Logger:     log.NewLogger(),
		Getenv:     func(string) string { return "" },
	}
	tm := &TokenManager{input: input}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	token, changed, err := tm.getOrCreateToken(t.Context(), logger, &inputGetOrCreateToken{
		App:              &pubconfig.App{Name: "test-app", ClientID: "cid"},
		Backend:          backend,
		EnableDeviceFlow: true,
	})
	if err != nil {
		t.Fatalf("getOrCreateToken() error = %v", err)
	}
	if !df.showCalled {
		t.Error("deviceFlow.Show was not invoked")
	}
	if changed {
		t.Error("changed = true, want false (the agent already stored the token)")
	}
	if diff := cmp.Diff(polled, token); diff != "" {
		t.Errorf("token mismatch (-want +got):\n%s", diff)
	}
}

// TestTokenManager_createToken_agentBeginReturnsToken verifies the concurrency
// case: when BeginDeviceFlow returns a token directly, createToken returns it
// with changed=false and does not show the one-time code or poll.
func TestTokenManager_createToken_agentBeginReturnsToken(t *testing.T) {
	t.Parallel()

	begun := &pubapi.AccessToken{
		AccessToken:    "concurrently-minted-token",
		ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	backend := &agentBackend{begun: begun}
	df := &mockDeviceFlow{}
	input := &Input{
		DeviceFlow: df,
		Getenv:     func(string) string { return "" },
	}
	tm := &TokenManager{input: input}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	token, changed, err := tm.createToken(t.Context(), logger, backend, 0, &deviceflow.InputCreate{ClientID: "cid"}, true)
	if err != nil {
		t.Fatalf("createToken() error = %v", err)
	}
	if changed {
		t.Error("changed = true, want false")
	}
	if df.showCalled {
		t.Error("deviceFlow.Show was invoked, want it not called")
	}
	if diff := cmp.Diff(begun, token); diff != "" {
		t.Errorf("token mismatch (-want +got):\n%s", diff)
	}
}

func TestEnableDeviceFlow(t *testing.T) {
	t.Parallel()
	data := []struct {
		name     string
		override *bool
		env      string
		want     bool
	}{
		{name: "default disabled when all unset", override: nil, env: "", want: false},
		{name: "env true enables", override: nil, env: "true", want: true},
		{name: "env false disables", override: nil, env: "false", want: false},
		{name: "env other value disables", override: nil, env: "1", want: false},
		{name: "override true beats env false", override: new(true), env: "false", want: true},
		{name: "override false beats env true", override: new(false), env: "true", want: false},
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
			if got := enableDeviceFlow(d.override, getEnv); got != d.want {
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

func TestResolveBackendType(t *testing.T) {
	t.Parallel()
	data := []struct {
		name string
		env  string
		cfg  string // backend.type in the config file
		want string
	}{
		{name: "default empty when all unset", want: ""},
		{name: "env wins", env: "agent", cfg: "text", want: "agent"},
		{name: "config when env unset", env: "", cfg: "text", want: "text"},
		{name: "env beats config", env: "keyring", cfg: "text", want: "keyring"},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(k string) string {
				if k == "GHTKN_BACKEND" {
					return d.env
				}
				return ""
			}
			var cfg *pubconfig.Backend
			if d.cfg != "" {
				cfg = &pubconfig.Backend{Type: d.cfg}
			}
			if got := resolveBackendType(cfg, getEnv); got != d.want {
				t.Errorf("resolveBackendType = %q, want %q", got, d.want)
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

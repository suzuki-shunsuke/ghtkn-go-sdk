//nolint:funlen
package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
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
	}
}

type mockKeyring struct {
	token   *pubapi.AccessToken
	err     error
	delErr  error
	deleted []string
}

func (m *mockKeyring) Get(_ context.Context, _ string) (*pubapi.AccessToken, error) {
	return m.token, m.err
}

func (m *mockKeyring) Set(_ context.Context, _ string, _ *pubapi.AccessToken) error {
	return m.err
}

func (m *mockKeyring) Delete(_ context.Context, clientID string) error {
	if m.delErr != nil {
		return m.delErr
	}
	m.deleted = append(m.deleted, clientID)
	return nil
}

// SupportsDeviceFlow reports false: mockKeyring is a keyring-like backend that
// does not run the device flow itself, so GetActive/BeginDeviceFlow/
// PollDeviceFlow/RevokeTokens are never called.
func (m *mockKeyring) SupportsDeviceFlow() bool { return false }

func (m *mockKeyring) GetActive(_ context.Context, _ string, _ time.Duration) (*pubapi.AccessToken, error) {
	return nil, errors.New("GetActive should not be called")
}

func (m *mockKeyring) BeginDeviceFlow(_ context.Context, _ string, _ time.Duration) (*pubapi.AccessToken, *pubdeviceflow.DeviceCodeResponse, error) {
	return nil, nil, errors.New("BeginDeviceFlow should not be called")
}

func (m *mockKeyring) PollDeviceFlow(_ context.Context, _ string, _ time.Duration) (*pubapi.AccessToken, error) {
	return nil, errors.New("PollDeviceFlow should not be called")
}

func (m *mockKeyring) RevokeTokens(_ context.Context, _ []string) (revokeFailed, cleanupFailed []string, err error) {
	return nil, nil, errors.New("RevokeTokens should not be called")
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

func TestTokenManager_Get(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setupInput func() *Input
		wantErr    bool
		wantErrIs  error
		wantToken  *pubapi.AccessToken
		input      *pubapi.InputGet
	}{
		{
			name: "GHTKN_GITHUB_TOKEN environment variable is returned as is",
			setupInput: func() *Input {
				input := newMockInput()
				input.Getenv = func(key string) string {
					if key == "GHTKN_GITHUB_TOKEN" {
						return "env-token"
					}
					return ""
				}
				return input
			},
			input:   &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"},
			wantErr: false,
			wantToken: &pubapi.AccessToken{
				AccessToken: "env-token",
			},
		},
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
				return input
			},
			input:   &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"},
			wantErr: false,
			wantToken: &pubapi.AccessToken{
				AccessToken:    "cached-token",
				ExpirationDate: futureTime,
			},
		},
		{
			name: "an expired token can't be replaced because Get never runs the device flow",
			setupInput: func() *Input {
				input := newMockInput()
				input.Backend = &mockKeyring{
					token: &pubapi.AccessToken{
						AccessToken:    "expired-token",
						ExpirationDate: time.Now().Add(-time.Hour),
					},
				}
				return input
			},
			input:     &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"},
			wantErr:   true,
			wantErrIs: pubapi.ErrDisableDeviceFlow,
		},
		{
			// GHTKN_ENABLE_DEVICE_FLOW was the temporary opt-in removed with
			// https://github.com/suzuki-shunsuke/ghtkn/issues/474. Nothing in the
			// environment may bring the device flow back to Get, so a stale export of it
			// must not change the outcome above.
			name: "GHTKN_ENABLE_DEVICE_FLOW is not read",
			setupInput: func() *Input {
				input := newMockInput()
				input.Getenv = func(key string) string {
					if key == "GHTKN_ENABLE_DEVICE_FLOW" {
						return "true"
					}
					return ""
				}
				input.DeviceFlow = &mockDeviceFlow{
					token: &deviceflow.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: futureTime,
					},
				}
				input.Backend = &mockKeyring{}
				return input
			},
			input:     &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"},
			wantErr:   true,
			wantErrIs: pubapi.ErrDisableDeviceFlow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Run under synctest so time.Now() is fixed at the bubble epoch
			// (2000-01-01 UTC). setupInput runs inside the bubble, so a token dated in
			// the future (e.g. 2025) reads as still valid and one built with
			// time.Now().Add(-d) reads as expired, deterministically and without a Now seam.
			synctest.Test(t, func(t *testing.T) {
				input := tt.setupInput()
				tm := New(input)
				logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

				token, _, err := tm.Get(t.Context(), logger, tt.input)
				if err != nil {
					if !tt.wantErr {
						t.Error(err)
					}
					if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
						t.Errorf("error = %v, want it to wrap %v", err, tt.wantErrIs)
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
		})
	}
}

func TestTokenManager_Auth(t *testing.T) {
	t.Parallel()

	newToken := &deviceflow.AccessToken{
		AccessToken:    "new-token",
		ExpirationDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name       string
		setupInput func() *Input
		wantErr    bool
		wantToken  *pubapi.AccessToken
		input      *pubapi.InputAuth
	}{
		{
			name: "no cached token runs the device flow",
			setupInput: func() *Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{token: newToken}
				input.Backend = &mockKeyring{}
				return input
			},
			input: &pubapi.InputAuth{ConfigFilePath: "/path/to/config.yaml"},
			wantToken: &pubapi.AccessToken{
				AccessToken:    newToken.AccessToken,
				ExpirationDate: newToken.ExpirationDate,
			},
		},
		{
			// Auth asks for more validity than GitHub's 8h maximum, so every real token
			// reads as expiring and is replaced. This is what makes 'ghtkn auth' refresh
			// proactively, and what lets it replace a revoked token. The cached token here
			// still has half its life left, which 'ghtkn get' would happily return.
			name: "a still-valid cached token is regenerated anyway",
			setupInput: func() *Input {
				input := newMockInput()
				input.DeviceFlow = &mockDeviceFlow{token: newToken}
				input.Backend = &mockKeyring{
					token: &pubapi.AccessToken{
						AccessToken:    "cached-token",
						ExpirationDate: time.Now().Add(4 * time.Hour),
					},
				}
				return input
			},
			input: &pubapi.InputAuth{ConfigFilePath: "/path/to/config.yaml"},
			wantToken: &pubapi.AccessToken{
				AccessToken:    newToken.AccessToken,
				ExpirationDate: newToken.ExpirationDate,
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
				return input
			},
			input:   &pubapi.InputAuth{ConfigFilePath: "/path/to/config.yaml"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// See TestTokenManager_Get for why this runs under synctest.
			synctest.Test(t, func(t *testing.T) {
				tm := New(tt.setupInput())
				logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

				token, _, err := tm.Auth(t.Context(), logger, tt.input)
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
		})
	}
}

func TestOpenBrowser(t *testing.T) {
	t.Parallel()
	// The GHTKN_OPEN_BROWSER override is folded into the config upstream by
	// config.ApplyEnvOverrides (tested there); this covers the config/default resolution.
	tests := []struct {
		name string
		cfg  *pubconfig.OpenBrowser
		want bool
	}{
		{name: "config unset defaults to open", cfg: nil, want: true},
		{name: "config disables", cfg: &pubconfig.OpenBrowser{Enable: new(false)}, want: false},
		{name: "config enables", cfg: &pubconfig.OpenBrowser{Enable: new(true)}, want: true},
		{name: "config present but unspecified defaults to open", cfg: &pubconfig.OpenBrowser{}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := openBrowser(tt.cfg); got != tt.want {
				t.Errorf("openBrowser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClipboard(t *testing.T) {
	t.Parallel()
	// The GHTKN_CLIPBOARD override is folded into the config upstream by
	// config.ApplyEnvOverrides (tested there); this covers the flag/config/default resolution.
	tests := []struct {
		name     string
		override *bool
		cfg      *pubconfig.Clipboard
		want     bool
	}{
		{name: "all unset defaults to disabled", want: false},
		{name: "config enables", cfg: &pubconfig.Clipboard{Enable: new(true)}, want: true},
		{name: "config disables", cfg: &pubconfig.Clipboard{Enable: new(false)}, want: false},
		{name: "config present but unspecified defaults to disabled", cfg: &pubconfig.Clipboard{}, want: false},
		{name: "override true beats config disable", override: new(true), cfg: &pubconfig.Clipboard{Enable: new(false)}, want: true},
		{name: "override false beats config enable", override: new(false), cfg: &pubconfig.Clipboard{Enable: new(true)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := clipboard(tt.override, tt.cfg); got != tt.want {
				t.Errorf("clipboard() = %v, want %v", got, tt.want)
			}
		})
	}
}

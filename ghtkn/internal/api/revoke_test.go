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
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// mockRevoker records the batches of tokens passed to Revoke.
type mockRevoker struct {
	revoked [][]string
	err     error
}

func (m *mockRevoker) Revoke(_ context.Context, tokens []string) error {
	m.revoked = append(m.revoked, append([]string(nil), tokens...))
	return m.err
}

func TestTokenManager_Revoke(t *testing.T) {
	t.Parallel()

	pastTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	storedToken := func() *pubapi.AccessToken {
		return &pubapi.AccessToken{AccessToken: "stored-tok", ExpirationDate: pastTime}
	}

	tests := []struct {
		name        string
		backend     *mockKeyring
		getenv      func(string) string
		input       *pubapi.InputRevoke
		wantErr     bool
		wantRevoked [][]string
		wantDeleted []string
	}{
		{
			name:        "app name revokes the stored token and deletes it from the backend",
			backend:     &mockKeyring{token: storedToken()},
			input:       &pubapi.InputRevoke{AppNames: []string{"test-app"}},
			wantRevoked: [][]string{{"stored-tok"}},
			wantDeleted: []string{"xxx"},
		},
		{
			name:        "app with no stored token is a no-op",
			backend:     &mockKeyring{token: nil},
			input:       &pubapi.InputRevoke{AppNames: []string{"test-app"}},
			wantRevoked: nil,
			wantDeleted: nil,
		},
		{
			name:        "no arguments falls back to the default app",
			backend:     &mockKeyring{token: storedToken()},
			input:       &pubapi.InputRevoke{},
			wantRevoked: [][]string{{"stored-tok"}},
			wantDeleted: []string{"xxx"},
		},
		{
			name:        "no arguments falls back to GHTKN_APP",
			backend:     &mockKeyring{token: storedToken()},
			getenv:      func(k string) string { return map[string]string{"GHTKN_APP": "test-app"}[k] },
			input:       &pubapi.InputRevoke{},
			wantRevoked: [][]string{{"stored-tok"}},
			wantDeleted: []string{"xxx"},
		},
		{
			name:    "unknown app name errors",
			backend: &mockKeyring{token: storedToken()},
			input:   &pubapi.InputRevoke{AppNames: []string{"nope"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			revoker := &mockRevoker{}
			getenv := tt.getenv
			if getenv == nil {
				getenv = func(string) string { return "" }
			}
			input := &Input{
				Backend:      tt.backend,
				Revoker:      revoker,
				Logger:       log.NewLogger(),
				ConfigReader: &mockConfigReader{},
				Getenv:       getenv,
			}
			tm := New(input)
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			// Set a config path so resolveConfigPath does not consult the real env;
			// mockConfigReader ignores the path anyway.
			if tt.input.ConfigFilePath == "" {
				tt.input.ConfigFilePath = "/path/to/config.yaml"
			}
			err := tm.Revoke(t.Context(), logger, tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Fatal(err)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("expected an error but got nil")
			}
			if diff := cmp.Diff(tt.wantRevoked, revoker.revoked); diff != "" {
				t.Errorf("revoked tokens mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantDeleted, tt.backend.deleted); diff != "" {
				t.Errorf("deleted client ids mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// multiConfigReader is a ConfigReader that returns a fixed set of apps.
type multiConfigReader struct {
	apps []*pubconfig.App
}

func (m *multiConfigReader) Read(cfg *pubconfig.Config, _ string) error {
	cfg.Apps = m.apps
	return nil
}

// mapKeyring is a Backend keyed by client ID, with optional per-client Get/Delete
// errors so partial-failure scenarios can be exercised.
type mapKeyring struct {
	tokens  map[string]*pubapi.AccessToken
	getErr  map[string]error
	delErr  map[string]error
	deleted []string
}

func (m *mapKeyring) Get(_ context.Context, clientID string) (*pubapi.AccessToken, error) {
	if err := m.getErr[clientID]; err != nil {
		return nil, err
	}
	return m.tokens[clientID], nil
}

func (m *mapKeyring) Set(_ context.Context, _ string, _ *pubapi.AccessToken) error {
	return nil
}

func (m *mapKeyring) Delete(_ context.Context, clientID string) error {
	if err := m.delErr[clientID]; err != nil {
		return err
	}
	m.deleted = append(m.deleted, clientID)
	return nil
}

func TestTokenManager_Revoke_all(t *testing.T) {
	t.Parallel()

	tok := func(s string) *pubapi.AccessToken {
		return &pubapi.AccessToken{AccessToken: s}
	}
	apps := []*pubconfig.App{
		{Name: "a", ClientID: "ca"},
		{Name: "b", ClientID: "cb"},
		{Name: "c", ClientID: "cc"},
	}

	tests := []struct {
		name        string
		tokens      map[string]*pubapi.AccessToken
		getErr      map[string]error
		delErr      map[string]error
		revokerErr  error
		wantRevoked [][]string
		wantDeleted []string
		// wantLive is whether the error should be classified as a live-credential
		// failure (ErrRevoke); wantStale for a backend cleanup failure.
		wantErr   bool
		wantLive  bool
		wantStale bool
	}{
		{
			name:        "all apps with a stored token are revoked and deleted",
			tokens:      map[string]*pubapi.AccessToken{"ca": tok("ta"), "cb": tok("tb"), "cc": tok("tc")},
			wantRevoked: [][]string{{"ta", "tb", "tc"}},
			wantDeleted: []string{"ca", "cb", "cc"},
		},
		{
			name:        "apps without a stored token are skipped",
			tokens:      map[string]*pubapi.AccessToken{"ca": tok("ta"), "cc": tok("tc")},
			wantRevoked: [][]string{{"ta", "tc"}},
			wantDeleted: []string{"ca", "cc"},
		},
		{
			name:        "a Get failure does not stop the others (live-credential failure)",
			tokens:      map[string]*pubapi.AccessToken{"ca": tok("ta"), "cc": tok("tc")},
			getErr:      map[string]error{"cb": errors.New("get boom")},
			wantRevoked: [][]string{{"ta", "tc"}},
			wantDeleted: []string{"ca", "cc"},
			wantErr:     true,
			wantLive:    true,
		},
		{
			name:        "a Delete failure does not stop the others (cleanup failure)",
			tokens:      map[string]*pubapi.AccessToken{"ca": tok("ta"), "cb": tok("tb"), "cc": tok("tc")},
			delErr:      map[string]error{"cb": errors.New("delete boom")},
			wantRevoked: [][]string{{"ta", "tb", "tc"}},
			wantDeleted: []string{"ca", "cc"},
			wantErr:     true,
			wantStale:   true,
		},
		{
			name:        "a revoker failure leaves the backend untouched (live-credential failure)",
			tokens:      map[string]*pubapi.AccessToken{"ca": tok("ta"), "cb": tok("tb"), "cc": tok("tc")},
			revokerErr:  errors.New("revoke boom"),
			wantRevoked: [][]string{{"ta", "tb", "tc"}},
			wantDeleted: nil,
			wantErr:     true,
			wantLive:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			revoker := &mockRevoker{err: tt.revokerErr}
			backend := &mapKeyring{tokens: tt.tokens, getErr: tt.getErr, delErr: tt.delErr}
			input := &Input{
				Backend:      backend,
				Revoker:      revoker,
				Logger:       log.NewLogger(),
				ConfigReader: &multiConfigReader{apps: apps},
				Getenv:       func(string) string { return "" },
			}
			tm := New(input)
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			err := tm.Revoke(t.Context(), logger, &pubapi.InputRevoke{All: true, ConfigFilePath: "/path/to/config.yaml"})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				if got := errors.Is(err, pubapi.ErrRevoke); got != tt.wantLive {
					t.Errorf("errors.Is(err, ErrRevoke) = %v, want %v (err: %v)", got, tt.wantLive, err)
				}
				if got := errors.Is(err, pubapi.ErrBackendCleanup); got != tt.wantStale {
					t.Errorf("errors.Is(err, ErrBackendCleanup) = %v, want %v (err: %v)", got, tt.wantStale, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.wantRevoked, revoker.revoked); diff != "" {
				t.Errorf("revoked tokens mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantDeleted, backend.deleted); diff != "" {
				t.Errorf("deleted client ids mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTokenManager_Revoke_revokerError(t *testing.T) {
	t.Parallel()

	storedToken := &pubapi.AccessToken{
		AccessToken:    "stored-tok",
		ExpirationDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	input := &Input{
		Backend:      &mockKeyring{token: storedToken},
		Revoker:      &mockRevoker{err: errors.New("boom")},
		Logger:       log.NewLogger(),
		ConfigReader: &mockConfigReader{},
		Getenv:       func(string) string { return "" },
	}
	tm := New(input)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	err := tm.Revoke(t.Context(), logger, &pubapi.InputRevoke{AppNames: []string{"test-app"}, ConfigFilePath: "/path/to/config.yaml"})
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
}

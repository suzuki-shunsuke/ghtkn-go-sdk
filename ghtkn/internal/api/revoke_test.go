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

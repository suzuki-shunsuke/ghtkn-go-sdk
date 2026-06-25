package backend

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
)

// mockInner is a stub implementation of the inner backend interface.
type mockInner struct {
	data []byte
	err  error
	set  func(clientID, token string) error
	del  func(clientID string) error
}

func (m *mockInner) Get(_ context.Context, _ string) ([]byte, error) {
	return m.data, m.err
}

func (m *mockInner) Set(_ context.Context, clientID, token string) error {
	if m.set != nil {
		return m.set(clientID, token)
	}
	return nil
}

func (m *mockInner) Delete(_ context.Context, clientID string) error {
	if m.del != nil {
		return m.del(clientID)
	}
	return nil
}

func TestBackend_Get(t *testing.T) {
	t.Parallel()

	exp := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	valid := `{"access_token":"tok","expiration_date":"2025-01-15T10:30:00Z"}`
	tests := []struct {
		name    string
		inner   *mockInner
		want    *api.AccessToken
		wantErr bool
	}{
		{
			name:  "valid token",
			inner: &mockInner{data: []byte(valid)},
			want:  &api.AccessToken{AccessToken: "tok", ExpirationDate: exp},
		},
		{
			name:  "not found returns nil",
			inner: &mockInner{data: nil},
			want:  nil,
		},
		{
			name:    "inner error is propagated",
			inner:   &mockInner{err: errors.New("boom")},
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			inner:   &mockInner{data: []byte("not json")},
			wantErr: true,
		},
		{
			name:    "invalid token fails validation",
			inner:   &mockInner{data: []byte(`{"access_token":"tok"}`)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := &Backend{backend: tt.inner}
			got, err := b.Get(t.Context(), "client-id")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.want == nil {
				if got != nil {
					t.Errorf("Get() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("Get() = nil, want %v", tt.want)
			}
			if got.AccessToken != tt.want.AccessToken ||
				!got.ExpirationDate.Equal(tt.want.ExpirationDate) {
				t.Errorf("Get() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBackend_Set(t *testing.T) {
	t.Parallel()

	t.Run("marshals the token to JSON", func(t *testing.T) {
		t.Parallel()

		var stored string
		b := &Backend{backend: &mockInner{set: func(_, token string) error {
			stored = token
			return nil
		}}}
		token := &api.AccessToken{
			AccessToken:    "tok",
			ExpirationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		}
		if err := b.Set(t.Context(), "client-id", token); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		got := &api.AccessToken{}
		if err := json.Unmarshal([]byte(stored), got); err != nil {
			t.Fatalf("stored value is not valid JSON: %v", err)
		}
		if got.AccessToken != token.AccessToken {
			t.Errorf("stored token = %+v, want %+v", got, token)
		}
	})

	t.Run("propagates inner errors", func(t *testing.T) {
		t.Parallel()

		b := &Backend{backend: &mockInner{set: func(_, _ string) error {
			return errors.New("boom")
		}}}
		if err := b.Set(t.Context(), "client-id", &api.AccessToken{}); err == nil {
			t.Error("Set() expected an error, got nil")
		}
	})
}

func TestBackend_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delegates to the inner backend", func(t *testing.T) {
		t.Parallel()

		var gotClientID string
		b := &Backend{backend: &mockInner{del: func(clientID string) error {
			gotClientID = clientID
			return nil
		}}}
		if err := b.Delete(t.Context(), "client-id"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if gotClientID != "client-id" {
			t.Errorf("Delete() client id = %q, want %q", gotClientID, "client-id")
		}
	})

	t.Run("propagates inner errors", func(t *testing.T) {
		t.Parallel()

		b := &Backend{backend: &mockInner{del: func(string) error {
			return errors.New("boom")
		}}}
		if err := b.Delete(t.Context(), "client-id"); err == nil {
			t.Error("Delete() expected an error, got nil")
		}
	})
}

func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("empty defaults to keyring", func(t *testing.T) {
		b, err := New("", os.Getenv)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if b == nil {
			t.Fatal("New() returned nil")
		}
	})

	t.Run("keyring", func(t *testing.T) {
		if _, err := New("keyring", os.Getenv); err != nil {
			t.Fatalf("New() error = %v", err)
		}
	})

	t.Run("text", func(t *testing.T) {
		if _, err := New("text", func(s string) string {
			if s == "XDG_CACHE_HOME" {
				return t.TempDir()
			}
			return os.Getenv(s)
		}); err != nil {
			t.Fatalf("New() error = %v", err)
		}
	})

	t.Run("unsupported backend errors", func(t *testing.T) {
		if _, err := New("bogus", os.Getenv); err == nil {
			t.Error("New() expected an error for an unsupported backend")
		}
	})
}

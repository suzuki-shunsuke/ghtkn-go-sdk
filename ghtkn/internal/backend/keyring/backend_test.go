package keyring

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zalando/go-keyring"
)

func TestBackend_Get(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	tests := []struct {
		name    string
		get     func(service, key string) (string, error)
		want    []byte
		wantErr bool
	}{
		{
			name: "found",
			get:  func(_, _ string) (string, error) { return "secret", nil },
			want: []byte("secret"),
		},
		{
			name: "not found returns nil",
			get:  func(_, _ string) (string, error) { return "", keyring.ErrNotFound },
			want: nil,
		},
		{
			name:    "other error is propagated",
			get:     func(_, _ string) (string, error) { return "", errBoom },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := &Backend{get: tt.get, service: DefaultServiceKey}
			got, err := b.Get(context.Background(), "client-id")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Get() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBackend_Set(t *testing.T) {
	t.Parallel()

	t.Run("stores the token under the service and client id", func(t *testing.T) {
		t.Parallel()

		var gotService, gotKey, gotToken string
		b := &Backend{
			set: func(service, key, token string) error {
				gotService, gotKey, gotToken = service, key, token
				return nil
			},
			service: DefaultServiceKey,
		}
		if err := b.Set(context.Background(), "client-id", "token"); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if gotService != DefaultServiceKey || gotKey != "client-id" || gotToken != "token" {
			t.Errorf("Set() stored (%q, %q, %q), want (%q, %q, %q)",
				gotService, gotKey, gotToken, DefaultServiceKey, "client-id", "token")
		}
	})

	t.Run("propagates errors", func(t *testing.T) {
		t.Parallel()

		b := &Backend{
			set:     func(_, _, _ string) error { return errors.New("boom") },
			service: DefaultServiceKey,
		}
		if err := b.Set(context.Background(), "client-id", "token"); err == nil {
			t.Error("Set() expected an error, got nil")
		}
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	b := New(&Input{ServiceKey: DefaultServiceKey})
	if b == nil {
		t.Fatal("New() returned nil")
	}
	if b.service != DefaultServiceKey {
		t.Errorf("service = %q, want %q", b.service, DefaultServiceKey)
	}
}

package text

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBackend_GetSet(t *testing.T) {
	t.Parallel()

	b := &Backend{dir: filepath.Join(t.TempDir(), "ghtkn", "tokens")}
	ctx := context.Background()

	// Get before Set returns (nil, nil).
	got, err := b.Get(ctx, "client-id")
	if err != nil {
		t.Fatalf("Get() before Set error = %v", err)
	}
	if got != nil {
		t.Fatalf("Get() before Set = %q, want nil", got)
	}

	// Set then Get round-trips.
	if err := b.Set(ctx, "client-id", "token"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err = b.Get(ctx, "client-id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if diff := cmp.Diff([]byte("token"), got); diff != "" {
		t.Errorf("Get() mismatch (-want +got):\n%s", diff)
	}

	// The file is created with permission 0600.
	info, err := os.Stat(filepath.Join(b.dir, "client-id"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 600", perm)
	}

	// Set overwrites an existing token.
	if err := b.Set(ctx, "client-id", "token2"); err != nil {
		t.Fatalf("Set() overwrite error = %v", err)
	}
	got, err = b.Get(ctx, "client-id")
	if err != nil {
		t.Fatalf("Get() after overwrite error = %v", err)
	}
	if diff := cmp.Diff([]byte("token2"), got); diff != "" {
		t.Errorf("Get() after overwrite mismatch (-want +got):\n%s", diff)
	}
}

func TestNew(t *testing.T) {
	t.Run("honors XDG_CACHE_HOME", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/tmp/xdg-cache")
		b, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if want := filepath.Join("/tmp/xdg-cache", "ghtkn", "tokens"); b.dir != want {
			t.Errorf("dir = %q, want %q", b.dir, want)
		}
	})

	t.Run("falls back to HOME/.cache", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("HOME", "/home/tester")
		b, err := New()
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if want := filepath.Join("/home/tester", ".cache", "ghtkn", "tokens"); b.dir != want {
			t.Errorf("dir = %q, want %q", b.dir, want)
		}
	})

	t.Run("errors when neither is set", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("HOME", "")
		if _, err := New(); err == nil {
			t.Error("New() expected an error when XDG_CACHE_HOME and HOME are unset")
		}
	})
}

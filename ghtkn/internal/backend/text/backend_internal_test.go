package text

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBackend_GetSet(t *testing.T) {
	t.Parallel()

	b := &Backend{dir: filepath.Join(t.TempDir(), "ghtkn", "tokens")}
	ctx := t.Context()

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

func Test_cacheDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "honors XDG_CACHE_HOME",
			env:  map[string]string{"XDG_CACHE_HOME": "/tmp/xdg-cache"},
			want: "/tmp/xdg-cache",
		},
		{
			name: "prefers XDG_CACHE_HOME over HOME",
			env:  map[string]string{"XDG_CACHE_HOME": "/tmp/xdg-cache", "HOME": "/home/tester"},
			want: "/tmp/xdg-cache",
		},
		{
			name: "falls back to HOME/.cache",
			env:  map[string]string{"HOME": "/home/tester"},
			want: filepath.Join("/home/tester", ".cache"),
		},
		{
			name:    "errors when neither is set",
			env:     map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := cacheDir(func(k string) string { return tt.env[k] })
			if (err != nil) != tt.wantErr {
				t.Fatalf("cacheDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("cacheDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_tokenDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "honors GHTKN_TEXT_BACKEND_DIR",
			env:  map[string]string{"GHTKN_TEXT_BACKEND_DIR": "/tmp/tokens"},
			want: "/tmp/tokens",
		},
		{
			name: "prefers GHTKN_TEXT_BACKEND_DIR over XDG_CACHE_HOME",
			env:  map[string]string{"GHTKN_TEXT_BACKEND_DIR": "/tmp/tokens", "XDG_CACHE_HOME": "/tmp/xdg-cache"},
			want: "/tmp/tokens",
		},
		{
			name: "falls back to XDG_CACHE_HOME/ghtkn/tokens",
			env:  map[string]string{"XDG_CACHE_HOME": "/tmp/xdg-cache"},
			want: filepath.Join("/tmp/xdg-cache", "ghtkn", "tokens"),
		},
		{
			name: "falls back to HOME/.cache/ghtkn/tokens",
			env:  map[string]string{"HOME": "/home/tester"},
			want: filepath.Join("/home/tester", ".cache", "ghtkn", "tokens"),
		},
		{
			name:    "errors when nothing is set",
			env:     map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tokenDir(func(k string) string { return tt.env[k] })
			if (err != nil) != tt.wantErr {
				t.Fatalf("tokenDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("tokenDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

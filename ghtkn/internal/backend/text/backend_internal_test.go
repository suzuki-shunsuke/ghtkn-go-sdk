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

	// Set then Get round-trips. Set appends a trailing newline that Get trims.
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

	// Set overwrites an existing token; a token that already ends with a newline
	// is not given a second one, so Get still round-trips it.
	if err := b.Set(ctx, "client-id", "token2\n"); err != nil {
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

func TestBackend_Delete(t *testing.T) {
	t.Parallel()

	b := &Backend{dir: filepath.Join(t.TempDir(), "ghtkn", "tokens")}
	ctx := t.Context()

	// Delete before Set is a no-op (no file exists yet).
	if err := b.Delete(ctx, "client-id"); err != nil {
		t.Fatalf("Delete() before Set error = %v", err)
	}

	// After Set, Delete removes the file so a subsequent Get returns (nil, nil).
	if err := b.Set(ctx, "client-id", "token"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := b.Delete(ctx, "client-id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	got, err := b.Get(ctx, "client-id")
	if err != nil {
		t.Fatalf("Get() after Delete error = %v", err)
	}
	if got != nil {
		t.Errorf("Get() after Delete = %q, want nil", got)
	}
}

func Test_cacheDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		goos    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "honors XDG_CACHE_HOME",
			goos: "linux",
			env:  map[string]string{"XDG_CACHE_HOME": "/tmp/xdg-cache"},
			want: "/tmp/xdg-cache",
		},
		{
			name: "prefers XDG_CACHE_HOME over HOME",
			goos: "linux",
			env:  map[string]string{"XDG_CACHE_HOME": "/tmp/xdg-cache", "HOME": "/home/tester"},
			want: "/tmp/xdg-cache",
		},
		{
			name: "falls back to HOME/.cache",
			goos: "linux",
			env:  map[string]string{"HOME": "/home/tester"},
			want: filepath.Join("/home/tester", ".cache"),
		},
		{
			name:    "errors when neither is set",
			goos:    "linux",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "on Windows uses LocalAppData/cache",
			goos: "windows",
			env:  map[string]string{"LocalAppData": "/local-app-data"},
			want: filepath.Join("/local-app-data", "cache"),
		},
		{
			name: "on Windows ignores XDG_CACHE_HOME",
			goos: "windows",
			env:  map[string]string{"LocalAppData": "/local-app-data", "XDG_CACHE_HOME": "/tmp/xdg-cache"},
			want: filepath.Join("/local-app-data", "cache"),
		},
		{
			name:    "errors on Windows when LocalAppData is not set",
			goos:    "windows",
			env:     map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := cacheDir(func(k string) string { return tt.env[k] }, tt.goos)
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
		goos    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "honors GHTKN_TEXT_BACKEND_DIR",
			goos: "linux",
			env:  map[string]string{"GHTKN_TEXT_BACKEND_DIR": "/tmp/tokens"},
			want: "/tmp/tokens",
		},
		{
			name: "prefers GHTKN_TEXT_BACKEND_DIR over XDG_CACHE_HOME",
			goos: "linux",
			env:  map[string]string{"GHTKN_TEXT_BACKEND_DIR": "/tmp/tokens", "XDG_CACHE_HOME": "/tmp/xdg-cache"},
			want: "/tmp/tokens",
		},
		{
			name: "falls back to XDG_CACHE_HOME/ghtkn/tokens",
			goos: "linux",
			env:  map[string]string{"XDG_CACHE_HOME": "/tmp/xdg-cache"},
			want: filepath.Join("/tmp/xdg-cache", "ghtkn", "tokens"),
		},
		{
			name: "falls back to HOME/.cache/ghtkn/tokens",
			goos: "linux",
			env:  map[string]string{"HOME": "/home/tester"},
			want: filepath.Join("/home/tester", ".cache", "ghtkn", "tokens"),
		},
		{
			name: "on Windows uses LocalAppData/cache/ghtkn/tokens",
			goos: "windows",
			env:  map[string]string{"LocalAppData": "/local-app-data"},
			want: filepath.Join("/local-app-data", "cache", "ghtkn", "tokens"),
		},
		{
			name:    "errors when nothing is set",
			goos:    "linux",
			env:     map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tokenDir(func(k string) string { return tt.env[k] }, tt.goos)
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

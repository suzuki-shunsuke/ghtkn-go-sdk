//go:build !windows

package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowser_Available(t *testing.T) {
	b := &Browser{}

	// With no browser command on PATH, the browser is not available.
	t.Setenv("PATH", "")
	if b.Available() {
		t.Error("Available() = true with empty PATH, want false")
	}

	// With a platform browser command on PATH, the browser is available.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, cmds()[0]), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if !b.Available() {
		t.Error("Available() = false with a browser command on PATH, want true")
	}
}

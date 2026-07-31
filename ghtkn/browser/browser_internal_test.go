//go:build !windows

package browser

import (
	"errors"
	"os"
	"testing"
)

// TestBrowserCmd checks that opening a browser can't write to the standard output of
// the process opening it, which carries the access token in 'ghtkn get' and git's
// credential protocol in 'ghtkn git-credential'.
func TestBrowserCmd(t *testing.T) {
	t.Parallel()

	cmd := browserCmd(t.Context(), "xdg-open", "https://github.com/login/device")
	if cmd.Stdout != os.Stderr {
		t.Errorf("Stdout = %v, want os.Stderr", cmd.Stdout)
	}
	if cmd.Stderr != os.Stderr {
		t.Errorf("Stderr = %v, want os.Stderr", cmd.Stderr)
	}
}

func TestBrowser_Available(t *testing.T) {
	t.Parallel()

	// With no browser command on PATH, the browser is not available.
	notFound := &Browser{lookPath: func(string) (string, error) {
		return "", errors.New("not found")
	}}
	if notFound.Available() {
		t.Error("Available() = true when no command is on PATH, want false")
	}

	// With a platform browser command on PATH, the browser is available.
	found := &Browser{lookPath: func(cmd string) (string, error) {
		return "/usr/bin/" + cmd, nil
	}}
	if !found.Available() {
		t.Error("Available() = false when a browser command is on PATH, want true")
	}
}

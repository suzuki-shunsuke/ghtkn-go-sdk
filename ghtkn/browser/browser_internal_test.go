//go:build !windows

package browser

import (
	"errors"
	"testing"
)

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

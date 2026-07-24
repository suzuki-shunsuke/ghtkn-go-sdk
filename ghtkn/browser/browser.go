// Package browser provides functionality to open URLs in the system's default web browser.
// It supports multiple platforms and handles various browser commands.
package browser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// ErrNoCommandFound is returned when no suitable command is found to open a browser.
var ErrNoCommandFound = errors.New("no command found to open the browser")

// Browser provides methods to open URLs in the system's default web browser.
// It implements platform-specific logic to handle different operating systems.
type Browser struct {
	// lookPath resolves a command to its path on PATH. It is a field so tests can inject
	// a stub instead of driving the real PATH via t.Setenv (which forbids t.Parallel). A
	// nil value falls back to exec.LookPath.
	lookPath func(file string) (string, error)
}

// findCmd resolves cmd on PATH, using the injected lookPath or exec.LookPath by default.
func (b *Browser) findCmd(cmd string) (string, error) {
	if b.lookPath != nil {
		return b.lookPath(cmd)
	}
	return exec.LookPath(cmd) //nolint:wrapcheck
}

// Open opens the specified URL in the system's default browser.
// It is platform-specific and delegates to the appropriate implementation.
func (b *Browser) Open(ctx context.Context, _ *slog.Logger, url string) error {
	return b.openB(ctx, url)
}

// Available reports whether a command to open the browser is available on this
// host. Callers can use it to fall back to asking the user to open the URL
// themselves instead of attempting an open that would fail.
func (b *Browser) Available() bool {
	return b.availableB()
}

// hasCmd reports whether any of the platform's browser commands is on PATH.
// It is used by the command-based platforms (Linux, macOS).
func (b *Browser) hasCmd() bool {
	for _, cmd := range cmds() {
		if _, err := b.findCmd(cmd); err == nil {
			return true
		}
	}
	return false
}

// runCmd attempts to open a URL using available browser commands.
// It tries each command in order until one succeeds or all fail.
// Returns errNoCommandFound if no suitable command is available.
func (b *Browser) runCmd(ctx context.Context, url string) error {
	for _, cmd := range cmds() {
		if _, err := b.findCmd(cmd); err != nil {
			continue
		}
		if err := command(ctx, cmd, url).Run(); err != nil {
			return fmt.Errorf("open the browser: %w", slogerr.With(err, "command_to_open_browser", cmd))
		}
		return nil
	}
	return ErrNoCommandFound
}

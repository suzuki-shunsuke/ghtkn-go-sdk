// Package browser provides functionality to open URLs in the system's default web browser.
// It supports multiple platforms and handles various browser commands.
package browser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/suzuki-shunsuke/go-exec/goexec"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

// ErrNoCommandFound is returned when no suitable command is found to open a browser.
var ErrNoCommandFound = errors.New("no command found to open the browser")

// Browser provides methods to open URLs in the system's default web browser.
// It implements platform-specific logic to handle different operating systems.
type Browser struct{}

// Open opens the specified URL in the system's default browser.
// It is platform-specific and delegates to the appropriate implementation.
func (b *Browser) Open(ctx context.Context, _ *slog.Logger, url string) error {
	return openB(ctx, url)
}

// Available reports whether a command to open the browser is available on this
// host. Callers can use it to fall back to asking the user to open the URL
// themselves instead of attempting an open that would fail.
func (b *Browser) Available() bool {
	return availableB()
}

// hasCmd reports whether any of the platform's browser commands is on PATH.
// It is used by the command-based platforms (Linux, macOS).
func hasCmd() bool {
	for _, cmd := range cmds() {
		if _, err := exec.LookPath(cmd); err == nil {
			return true
		}
	}
	return false
}

// runCmd attempts to open a URL using available browser commands.
// It tries each command in order until one succeeds or all fail.
// Returns errNoCommandFound if no suitable command is available.
func runCmd(ctx context.Context, url string) error {
	for _, cmd := range cmds() {
		if _, err := exec.LookPath(cmd); err != nil {
			continue
		}
		if err := goexec.Command(ctx, cmd, url).Run(); err != nil {
			return fmt.Errorf("open the browser: %w", slogerr.With(err, "command_to_open_browser", cmd))
		}
		return nil
	}
	return ErrNoCommandFound
}

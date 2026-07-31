// Package browser provides functionality to open URLs in the system's default web browser.
// It supports multiple platforms and handles various browser commands.
package browser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
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
	// stdout receives the standard output of the command that opens the browser. A nil
	// value falls back to os.Stderr; see SetStdout for why it is never the standard
	// output of the process opening the browser.
	stdout io.Writer
}

// SetStdout sets the writer that receives the standard output of the command opening
// the browser. Passing nil restores the default, os.Stderr.
//
// The default is os.Stderr rather than os.Stdout because the standard output of the
// process opening the browser carries data its caller parses: a CLI built on this SDK
// writes the access token there, and a git credential helper speaks git's credential
// protocol there. A single line from the browser command corrupts either one, and the
// resulting failure looks like a bad token rather than like browser output. On Linux
// the browser command can even be a text browser, since the www-browser alternative
// resolves to w3m or lynx on some hosts, and those render the whole page to stdout.
//
// Set this when the process embedding the SDK needs that output somewhere else, such
// as a log or io.Discard, instead of on its standard error.
func (b *Browser) SetStdout(w io.Writer) {
	b.stdout = w
}

// stdoutWriter returns the writer for the browser command's standard output, using
// the injected stdout or os.Stderr by default.
func (b *Browser) stdoutWriter() io.Writer {
	if b.stdout != nil {
		return b.stdout
	}
	return os.Stderr
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
		// Run the resolved path rather than the bare name: it is what findCmd looked
		// up, and it lets tests drive runCmd through the injected lookPath without
		// touching PATH or launching a real browser.
		path, err := b.findCmd(cmd)
		if err != nil {
			continue
		}
		if err := b.browserCmd(ctx, path, url).Run(); err != nil {
			return fmt.Errorf("open the browser: %w", slogerr.With(err, "command_to_open_browser", cmd))
		}
		return nil
	}
	return ErrNoCommandFound
}

// browserCmd builds the command that opens url, sending its standard output to the
// writer from stdoutWriter instead of the standard output of this process. See
// SetStdout for why that output must not go to os.Stdout.
//
// Its standard input is nil, so the command reads from the null device rather than
// from the standard input of this process. That input is not spare either: a git
// credential helper built on this SDK is fed git's credential request on it, and
// deviceflow's UI reads the user's Enter key from it. A browser command that reads a
// byte of it takes that byte away from its owner, and the failure looks like a
// truncated credential request rather than like the browser. A command that opens a
// GUI browser never reads stdin, but on Linux the command can be a text browser,
// since the www-browser alternative resolves to w3m or lynx on some hosts, and those
// read stdin as keyboard input.
//
// A text browser therefore sees EOF and exits instead of taking over the terminal.
// That is the intended outcome: the verification URL is already on stderr, so the
// user can open it, whereas a text browser driven by a credential request is not a
// state anyone can act on.
//
// The rest of the defaults, including stderr and the signal handling, come from
// command, which is kept identical to its upstream copy.
func (b *Browser) browserCmd(ctx context.Context, name, url string) *exec.Cmd {
	cmd := command(ctx, name, url)
	cmd.Stdout = b.stdoutWriter()
	cmd.Stdin = nil
	return cmd
}

//go:build !windows

package browser

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const deviceURL = "https://github.com/login/device"

// TestBrowser_runCmd_stdin checks that the command opening the browser can't read the
// standard input of the process opening it, which carries git's credential request for
// a git credential helper and the user's Enter key for deviceflow's UI.
//
// The check has to be behavioral rather than an assertion on cmd.Stdin, because
// asserting the field would restate browserCmd's body. It therefore gives this process
// a stdin holding a sentinel and uses a browser command that copies whatever stdin it
// gets to stdout: reaching the sentinel is the bug, and it shows up in the output.
//
// It does not call t.Parallel, because os.Stdin is process-global. Go runs the body of
// a serial test while every parallel test in the package is paused, so nothing else
// observes the swap.
func TestBrowser_runCmd_stdin(t *testing.T) {
	// A script rather than cat itself: runCmd passes the URL as an argument, and cat
	// would read that as a file name instead of reading stdin.
	script := filepath.Join(t.TempDir(), "browser")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat\n"), 0o700); err != nil { //nolint:gosec // the script must be executable
		t.Fatalf("write the stub browser command: %v", err)
	}

	// The writer is closed right after the sentinel so that a command reading this
	// stdin sees EOF and exits, making the bug fail the test instead of hanging it.
	const sentinel = "CREDENTIAL_REQUEST_FROM_THE_CALLER\n"
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create a pipe: %v", err)
	}
	if _, err := io.WriteString(w, sentinel); err != nil {
		t.Fatalf("write the sentinel: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close the pipe writer: %v", err)
	}

	stdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = stdin
		if err := r.Close(); err != nil {
			t.Errorf("close the pipe reader: %v", err)
		}
	})

	stdout := &bytes.Buffer{}
	b := &Browser{lookPath: func(string) (string, error) { return script, nil }}
	b.SetStdout(stdout)

	if err := b.runCmd(t.Context(), deviceURL); err != nil {
		t.Fatalf("runCmd() = %v, want nil", err)
	}
	if got := stdout.String(); got != "" {
		t.Errorf("the browser command read %q from the standard input of this process, want it to read nothing", got)
	}
}

// TestBrowser_runCmd_stdout checks that the command opening the browser can't write to
// the standard output of the process opening it, which carries the access token for a
// CLI built on this SDK and git's credential protocol for a git credential helper.
//
// It drives runCmd rather than browserCmd so that building the command any other way
// fails the test: asserting on browserCmd alone would restate its body and stay green
// even if runCmd stopped calling it.
func TestBrowser_runCmd_stdout(t *testing.T) {
	t.Parallel()

	// echo stands in for the browser command: runCmd runs the path lookPath resolved,
	// so no real browser is launched and PATH is left alone.
	echo, err := exec.LookPath("echo")
	if err != nil {
		t.Skipf("echo is not on PATH: %v", err)
	}

	stdout := &bytes.Buffer{}
	b := &Browser{lookPath: func(string) (string, error) { return echo, nil }}
	b.SetStdout(stdout)

	if err := b.runCmd(t.Context(), deviceURL); err != nil {
		t.Fatalf("runCmd() = %v, want nil", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != deviceURL {
		t.Errorf("browser command stdout = %q, want %q; it must not reach os.Stdout", got, deviceURL)
	}
}

// TestBrowser_browserCmd_defaultStdout checks the default a caller gets without
// SetStdout, which the runCmd test can't observe because it injects a writer.
func TestBrowser_browserCmd_defaultStdout(t *testing.T) {
	t.Parallel()

	cmd := (&Browser{}).browserCmd(t.Context(), "xdg-open", deviceURL)
	if cmd.Stdout != os.Stderr {
		t.Errorf("Stdout = %v, want os.Stderr", cmd.Stdout)
	}
}

// TestBrowser_runCmd_noCommand checks that runCmd reports ErrNoCommandFound instead of
// running anything when no browser command resolves.
func TestBrowser_runCmd_noCommand(t *testing.T) {
	t.Parallel()

	b := &Browser{lookPath: func(string) (string, error) { return "", errors.New("not found") }}
	if err := b.runCmd(t.Context(), deviceURL); !errors.Is(err, ErrNoCommandFound) {
		t.Errorf("runCmd() = %v, want ErrNoCommandFound", err)
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

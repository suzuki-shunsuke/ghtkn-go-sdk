package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"syscall"

	"github.com/charmbracelet/x/term"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

type passwordReader struct {
	stderr io.Writer
}

func NewPasswordReader(stderr io.Writer) *passwordReader {
	return &passwordReader{
		stderr: stderr,
	}
}

type readResult struct {
	input string
	err   error
}

func (p *passwordReader) Read(ctx context.Context, _ *slog.Logger, app *config.App) (string, error) {
	fmt.Fprintf(p.stderr, "Enter GitHub App Client ID (id: %d, name: %s): ", app.AppID, app.Name) //nolint:errcheck
	inputCh := make(chan *readResult, 1)
	go func() {
		b, err := term.ReadPassword(uintptr(syscall.Stdin))
		fmt.Fprintln(p.stderr, "") //nolint:errcheck
		if err != nil {
			inputCh <- &readResult{err: fmt.Errorf("read password: %w", err)}
		} else {
			inputCh <- &readResult{input: strings.TrimSpace(string(b))}
		}
		close(inputCh)
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(p.stderr, "Cancelled") //nolint:errcheck
		return "", ctx.Err()
	case result := <-inputCh:
		return result.input, result.err
	}
}

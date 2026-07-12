package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"golang.org/x/term"
)

var _ pubdeviceflow.OnetimeCodeUI = &simpleOnetimeCodeUI{}

// simpleOnetimeCodeUI is a basic implementation of OnetimeCodeUI that displays the
// one-time code (user code) and verification URL to stderr and waits for user input from stdin.
// It handles the GitHub device flow authentication process by showing the user code
// and verification URL, then waiting for the user to press Enter.
type simpleOnetimeCodeUI struct {
	stdin  io.Reader // Input source for reading user interaction (typically os.Stdin)
	stderr io.Writer // Output destination for displaying messages (typically os.Stderr)
	waiter waiter    // waiter for handling wait operations, can be customized for testing
}

// newOnetimeCodeUI creates a new simpleOnetimeCodeUI instance.
// It takes stdin for user input and stderr for output messages.
func newOnetimeCodeUI(stdin io.Reader, stderr io.Writer, waiter waiter) *simpleOnetimeCodeUI {
	return &simpleOnetimeCodeUI{
		stdin:  stdin,
		stderr: stderr,
		waiter: waiter,
	}
}

// Show displays the one-time code (user code) to the user and waits for Enter key press.
// It shows the user code, expiration time, and verification URL.
// The function returns when Enter is pressed or the context is cancelled.
// Note that it exits immediately without waiting input if stdin is not a terminal (pipe/redirect).
// In case of Git Credential Helper stdin is not a terminal, so it exits immediately.
func (d *simpleOnetimeCodeUI) Show(ctx context.Context, _ *slog.Logger, deviceCode *pubdeviceflow.DeviceCodeResponse, expirationDate time.Time, input *pubdeviceflow.InputShow) error {
	msgHeader := `The application uses the device flow to generate your GitHub User Access Token.
Copy your one-time code: %s
`
	if input.CopiedToClipboard {
		msgHeader += `(The one-time code has been copied to your clipboard.)
`
	}
	if input.AppName != "" {
		msgHeader += `App Name: %s
`
	}
	msgHeader += `This code is valid until %s
`

	// Build the format arguments to match the verbs in msgHeader. The App Name
	// line (and its verb) is only present when AppName is set, so AppName must be
	// omitted from the arguments otherwise to keep verbs and arguments aligned.
	args := []any{deviceCode.UserCode}
	if input.AppName != "" {
		args = append(args, input.AppName)
	}
	args = append(args, expirationDate.Format(time.RFC3339), deviceCode.VerificationURI)
	if !input.OpenBrowser {
		// The browser won't be opened automatically (disabled, or no browser is
		// available), so ask the user to open the URL themselves. Polling proceeds
		// immediately; there is nothing to wait for.
		msgTemplate := msgHeader + `Open the following URL in your browser and enter the one-time code above:
%s
`
		fmt.Fprintf(d.stderr, msgTemplate, args...) //nolint:errcheck
		return nil
	}
	if term.IsTerminal(0) {
		msgTemplate := msgHeader + `Press Enter to open %s in your browser (it opens automatically after 10 seconds)...
`
		fmt.Fprintf(d.stderr, msgTemplate, args...) //nolint:errcheck
		inputCh := make(chan error, 1)
		go func() {
			// Wait until Enter is pressed
			scanner := bufio.NewScanner(d.stdin)
			if scanner.Scan() {
				inputCh <- scanner.Err()
			}
			close(inputCh)
		}()
		select {
		case <-ctx.Done():
			fmt.Fprintln(d.stderr, "Cancelled") //nolint:errcheck
			return ctx.Err()
		case err := <-inputCh:
			return err
		case <-time.After(10 * time.Second):
			return nil
		}
	}
	msgTemplate := msgHeader + `%s will open automatically after a few seconds...
`
	fmt.Fprintf(d.stderr, msgTemplate, args...) //nolint:errcheck
	// If stdin is not a terminal, we cannot wait for user input.
	// So, we just wait for a few seconds to show the message and return.
	// In case of Git Credential Helper stdin is not a terminal.
	if err := d.waiter.Wait(ctx, 5*time.Second); err != nil {
		return err //nolint:wrapcheck
	}
	return nil
}

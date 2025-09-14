package deviceflow

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"golang.org/x/term"
)

var _ DeviceCodeUI = &SimpleDeviceCodeUI{}

// DeviceCodeUI provides an interface for displaying device flow information to users.
// Implementations should show the device code, verification URL, and handle user interaction.
type DeviceCodeUI interface {
	Show(ctx context.Context, logger *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error
}

// SimpleDeviceCodeUI is a basic implementation of DeviceCodeUI that displays device flow
// information to stderr and waits for user input from stdin.
// It handles the GitHub device flow authentication process by showing the user code
// and verification URL, then waiting for the user to press Enter.
type SimpleDeviceCodeUI struct {
	stdin  io.Reader // Input source for reading user interaction (typically os.Stdin)
	stderr io.Writer // Output destination for displaying messages (typically os.Stderr)
	waiter Waiter    // Waiter for handling wait operations, can be customized for testing
}

// NewDeviceCodeUI creates a new SimpleDeviceCodeUI instance.
// It takes stdin for user input and stderr for output messages.
func NewDeviceCodeUI(stdin io.Reader, stderr io.Writer, waiter Waiter) *SimpleDeviceCodeUI {
	return &SimpleDeviceCodeUI{
		stdin:  stdin,
		stderr: stderr,
		waiter: waiter,
	}
}

// Show displays the device flow information to the user and waits for Enter key press.
// It shows the user code, expiration time, and verification URL.
// The function returns when Enter is pressed or the context is cancelled.
// Note that it exits immediately without waiting input if stdin is not a terminal (pipe/redirect).
// In case of Git Credential Helper stdin is not a terminal, so it exits immediately.
func (d *SimpleDeviceCodeUI) Show(ctx context.Context, _ *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error {
	if term.IsTerminal(0) {
		const msgTemplate = `The application uses the device flow to generate your GitHub User Access Token.
Copy your one-time code: %s
This code is valid until %s
Press Enter to open %s in your browser...
`
		fmt.Fprintf(d.stderr, msgTemplate, deviceCode.UserCode, expirationDate.Format(time.RFC3339), deviceCode.VerificationURI) //nolint:errcheck
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
		}
	}
	const msgTemplate = `The application uses the device flow to generate your GitHub User Access Token.
Copy your one-time code: %s
This code is valid until %s
%s will open automatically after a few seconds...
`
	fmt.Fprintf(d.stderr, msgTemplate, deviceCode.UserCode, expirationDate.Format(time.RFC3339), deviceCode.VerificationURI) //nolint:errcheck
	// If stdin is not a terminal, we cannot wait for user input.
	// So, we just wait for a few seconds to show the message and return.
	// In case of Git Credential Helper stdin is not a terminal.
	if err := d.waiter.Wait(ctx, 5*time.Second); err != nil {
		return err //nolint:wrapcheck
	}
	return nil
}

package deviceflow

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
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
}

// NewDeviceCodeUI creates a new SimpleDeviceCodeUI instance.
// It takes stdin for user input and stderr for output messages.
func NewDeviceCodeUI(stdin io.Reader, stderr io.Writer) *SimpleDeviceCodeUI {
	return &SimpleDeviceCodeUI{
		stdin:  stdin,
		stderr: stderr,
	}
}

const msgTemplate = `The application uses the device flow to generate your GitHub User Access Token.
Copy your one-time code: %s
This code is valid until %s
Press Enter to open %s in your browser...
`

// Show displays the device flow information to the user and waits for Enter key press.
// It shows the user code, expiration time, and verification URL.
// The function returns when Enter is pressed or the context is cancelled.
// Note that it exits immediately without waiting input if stdin is not a terminal (pipe/redirect).
// In case of Git Credential Helper stdin is not a terminal, so it exits immediately.
func (d *SimpleDeviceCodeUI) Show(ctx context.Context, _ *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error {
	fmt.Fprintf(d.stderr, msgTemplate, deviceCode.UserCode, expirationDate.Format(time.RFC3339), deviceCode.VerificationURI) //nolint:errcheck
	inputCh := make(chan error, 1)

	go func() {
		// Wait until Enter is pressed
		// Note that this exits immediately without waiting input if stdin is not a terminal (pipe/redirect).
		// In case of Git Credential Helper stdin is not a terminal, so this exits immediately.
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

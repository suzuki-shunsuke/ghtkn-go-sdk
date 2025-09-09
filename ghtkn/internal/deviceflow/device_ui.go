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

type DeviceCodeUI interface {
	Show(ctx context.Context, logger *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error
}

type SimpleDeviceCodeUI struct {
	stdin  io.Reader
	stderr io.Writer
}

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

func (d *SimpleDeviceCodeUI) Show(_ context.Context, _ *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error {
	fmt.Fprintf(d.stderr, msgTemplate, deviceCode.UserCode, expirationDate.Format(time.RFC3339), deviceCode.VerificationURI) //nolint:errcheck
	scanner := bufio.NewScanner(d.stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read the input from stdin: %w", err)
	}
	return nil
}

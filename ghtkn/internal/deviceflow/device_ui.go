package deviceflow

import (
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
	stderr io.Writer
}

func NewDeviceCodeUI(stderr io.Writer) *SimpleDeviceCodeUI {
	return &SimpleDeviceCodeUI{
		stderr: stderr,
	}
}

func (d *SimpleDeviceCodeUI) Show(_ context.Context, _ *slog.Logger, deviceCode *DeviceCodeResponse, expirationDate time.Time) error {
	fmt.Fprintf(d.stderr, "Please visit: %s\n", deviceCode.VerificationURI)             //nolint:errcheck
	fmt.Fprintf(d.stderr, "And enter code: %s\n", deviceCode.UserCode)                  //nolint:errcheck
	fmt.Fprintf(d.stderr, "Expiration date: %s\n", expirationDate.Format(time.RFC3339)) //nolint:errcheck
	return nil
}

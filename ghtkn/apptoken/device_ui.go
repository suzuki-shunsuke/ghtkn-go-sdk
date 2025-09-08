package apptoken

import (
	"fmt"
	"io"
	"time"
)

var (
	_ DeviceCodeUI = &MockDeviceCodeUI{}
	_ DeviceCodeUI = &SimpleDeviceCodeUI{}
)

type DeviceCodeUI interface {
	Show(deviceCode *DeviceCodeResponse, expirationDate time.Time)
}

type MockDeviceCodeUI struct{}

func (m *MockDeviceCodeUI) Show(_ *DeviceCodeResponse, _ time.Time) {}

type SimpleDeviceCodeUI struct {
	stderr io.Writer
}

func NewDeviceCodeUI(stderr io.Writer) *SimpleDeviceCodeUI {
	return &SimpleDeviceCodeUI{
		stderr: stderr,
	}
}

func (d *SimpleDeviceCodeUI) Show(deviceCode *DeviceCodeResponse, expirationDate time.Time) {
	fmt.Fprintf(d.stderr, "Please visit: %s\n", deviceCode.VerificationURI)             //nolint:errcheck
	fmt.Fprintf(d.stderr, "And enter code: %s\n", deviceCode.UserCode)                  //nolint:errcheck
	fmt.Fprintf(d.stderr, "Expiration date: %s\n", expirationDate.Format(time.RFC3339)) //nolint:errcheck
}

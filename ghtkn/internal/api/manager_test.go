//nolint:revive
package api

import (
	"context"
	"log/slog"
	"os"
	"testing"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

type mockDeviceFlow struct {
	token *deviceflow.AccessToken
	err   error
}

func (m *mockDeviceFlow) SetLogger(_ *publog.Logger) {}

func (m *mockDeviceFlow) SetOnetimeCodeUI(_ pubdeviceflow.OnetimeCodeUI) {}

func (m *mockDeviceFlow) SetBrowser(_ pubdeviceflow.Browser) {}

func (m *mockDeviceFlow) Create(_ context.Context, logger *slog.Logger, input *deviceflow.InputCreate) (*deviceflow.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	input := &Input{}
	tm := New(input)
	if tm == nil {
		t.Error("New() returned nil")
	}
}

func TestNewInput(t *testing.T) {
	t.Parallel()

	input, err := NewInput(os.Getenv)
	if err != nil {
		t.Fatalf("NewInput() returned an error: %v", err)
	}
	if input == nil {
		t.Error("NewInput() returned nil")
		return
	}

	if input.DeviceFlow == nil {
		t.Error("NewInput().AppTokenClient is nil")
	}

	// Backend is built lazily (its type can come from the config file), so NewInput
	// leaves it nil and resolveBackend builds it on demand in Get/Revoke.
	if input.Backend != nil {
		t.Error("NewInput().Backend should be nil; it is resolved lazily")
	}

	if input.Now == nil {
		t.Error("NewInput().Now is nil")
	}
}

func TestInput_Validate(t *testing.T) {
	t.Parallel()

	// Currently, Input.Validate() always returns nil
	// since there are no validation rules for the Input struct
	input := &Input{}

	err := input.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

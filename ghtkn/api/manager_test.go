//nolint:revive
package api_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

type mockDeviceFlow struct {
	token *deviceflow.AccessToken
	err   error
}

func (m *mockDeviceFlow) SetLogger(_ *log.Logger) {}

func (m *mockDeviceFlow) SetDeviceCodeUI(_ deviceflow.DeviceCodeUI) {}

func (m *mockDeviceFlow) SetBrowser(_ deviceflow.Browser) {}

func (m *mockDeviceFlow) Create(_ context.Context, logger *slog.Logger, clientID string) (*deviceflow.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	input := &api.Input{}
	tm := api.New(input)
	if tm == nil {
		t.Error("New() returned nil")
	}
}

func TestNewInput(t *testing.T) {
	t.Parallel()

	input := api.NewInput()
	if input == nil {
		t.Error("NewInput() returned nil")
		return
	}

	if input.DeviceFlow == nil {
		t.Error("NewInput().AppTokenClient is nil")
	}

	if input.Keyring == nil {
		t.Error("NewInput().Keyring is nil")
	}

	if input.Now == nil {
		t.Error("NewInput().Now is nil")
	}
}

func TestInput_Validate(t *testing.T) {
	t.Parallel()

	// Currently, Input.Validate() always returns nil
	// since there are no validation rules for the Input struct
	input := &api.Input{}

	err := input.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

package ghtkn_test

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
)

type mockConfigReader struct {
	cfg *config.Config
	err error
}

func (m *mockConfigReader) Read(cfg *config.Config, _ string) error {
	if m.err != nil {
		return m.err
	}
	if m.cfg != nil {
		*cfg = *m.cfg
	}
	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	input := &ghtkn.Input{}
	client := ghtkn.New(input)
	if client == nil {
		t.Error("New() returned nil")
	}
}

func TestNewInput(t *testing.T) {
	t.Parallel()

	input := ghtkn.NewInput("/path/to/config")
	if input == nil {
		t.Error("NewInput() returned nil")
		return
	}

	if input.ConfigFilePath != "/path/to/config" {
		t.Errorf("NewInput().ConfigFilePath = %v, want /path/to/config", input.ConfigFilePath)
	}

	if input.FS == nil {
		t.Error("NewInput().FS is nil")
	}

	if input.ConfigReader == nil {
		t.Error("NewInput().ConfigReader is nil")
	}

	if input.Env == nil {
		t.Error("NewInput().Env is nil")
	}

	if input.Stdout == nil {
		t.Error("NewInput().Stdout is nil")
	}
}

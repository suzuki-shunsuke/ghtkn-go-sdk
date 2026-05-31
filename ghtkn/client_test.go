package ghtkn_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

func TestNew(t *testing.T) {
	t.Parallel()

	if c := ghtkn.New(); c == nil {
		t.Fatal("New() returned nil")
	}
}

// stubBrowser is a user-defined Browser implementation, verifying that a value
// satisfying ghtkn.Browser is accepted by Client.SetBrowser.
type stubBrowser struct{}

func (stubBrowser) Open(_ context.Context, _ *slog.Logger, _ string) error { return nil }

// stubDeviceCodeUI is a user-defined DeviceCodeUI implementation.
type stubDeviceCodeUI struct{}

func (stubDeviceCodeUI) Show(_ context.Context, _ *slog.Logger, _ *ghtkn.DeviceCodeResponse, _ time.Time) error {
	return nil
}

func TestClient_Setters(t *testing.T) {
	t.Parallel()

	c := ghtkn.New()
	// Must compile and not panic: public implementations are accepted by the wrapper.
	c.SetBrowser(stubBrowser{})
	c.SetBrowser(&ghtkn.DefaultBrowser{})
	c.SetDeviceCodeUI(stubDeviceCodeUI{})
	c.SetLogger(&ghtkn.Logger{})
}

func TestDefaultBrowser_ImplementsBrowser(t *testing.T) {
	t.Parallel()

	var _ ghtkn.Browser = &ghtkn.DefaultBrowser{}
}

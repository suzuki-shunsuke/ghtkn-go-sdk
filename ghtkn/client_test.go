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

	c, err := ghtkn.New()
	if err != nil {
		t.Fatalf("New() returned an error: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

// stubBrowser is a user-defined Browser implementation, verifying that a value
// satisfying ghtkn.Browser is accepted by Client.SetBrowser.
type stubBrowser struct{}

func (stubBrowser) Open(_ context.Context, _ *slog.Logger, _ string) error { return nil }

// stubOnetimeCodeUI is a user-defined OnetimeCodeUI implementation.
type stubOnetimeCodeUI struct{}

func (stubOnetimeCodeUI) Show(_ context.Context, _ *slog.Logger, _ *ghtkn.DeviceCodeResponse, _ time.Time, _ *ghtkn.InputShow) error {
	return nil
}

func TestClient_Setters(t *testing.T) {
	t.Parallel()

	c, err := ghtkn.New()
	if err != nil {
		t.Fatalf("New() returned an error: %v", err)
	}
	// Must compile and not panic: public implementations are accepted by the wrapper.
	c.SetBrowser(stubBrowser{})
	c.SetBrowser(&ghtkn.DefaultBrowser{})
	c.SetOnetimeCodeUI(stubOnetimeCodeUI{})
	c.SetLogger(&ghtkn.Logger{})
}

func TestDefaultBrowser_ImplementsBrowser(t *testing.T) {
	t.Parallel()

	var _ ghtkn.Browser = &ghtkn.DefaultBrowser{}
}

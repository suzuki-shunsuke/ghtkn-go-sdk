package deviceflow_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	intdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow/ui"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
)

// mockDeviceFlow is a fake DeviceFlow that records the order of its calls and
// returns canned responses, so Create's coordination can be tested without HTTP.
type mockDeviceFlow struct {
	calls      *[]string
	deviceCode *pubdeviceflow.DeviceCodeResponse
	token      *deviceflow.AccessToken
	getErr     error
	pollErr    error
}

func (m *mockDeviceFlow) GetDeviceCode(_ context.Context, _ string) (*pubdeviceflow.DeviceCodeResponse, error) {
	*m.calls = append(*m.calls, "GetDeviceCode")
	return m.deviceCode, m.getErr
}

func (m *mockDeviceFlow) Poll(_ context.Context, _ *slog.Logger, _ string, _ *pubdeviceflow.DeviceCodeResponse) (*deviceflow.AccessToken, error) {
	*m.calls = append(*m.calls, "Poll")
	return m.token, m.pollErr
}

// mockOnetimeCodeUI is a fake OnetimeCodeUI (the package-local interface) that
// records what Show received and the order of the call.
type mockOnetimeCodeUI struct {
	calls         *[]string
	gotInput      *ui.InputCreate
	gotDeviceCode *pubdeviceflow.DeviceCodeResponse
	err           error
	clipboardFn   pubdeviceflow.CopyTextToClipboard
}

func (m *mockOnetimeCodeUI) Show(_ context.Context, _ *slog.Logger, input *ui.InputCreate, deviceCode *pubdeviceflow.DeviceCodeResponse) error {
	if m.calls != nil {
		*m.calls = append(*m.calls, "Show")
	}
	m.gotInput = input
	m.gotDeviceCode = deviceCode
	return m.err
}

func (m *mockOnetimeCodeUI) SetBrowser(_ pubdeviceflow.Browser)             {}
func (m *mockOnetimeCodeUI) SetOnetimeCodeUI(_ pubdeviceflow.OnetimeCodeUI) {}
func (m *mockOnetimeCodeUI) SetCopyOnetimeCodeToClipboard(f pubdeviceflow.CopyTextToClipboard) {
	m.clipboardFn = f
}

// TestClient_SetCopyOnetimeCodeToClipboard guards that the clipboard function reaches
// the UI layer that actually copies the code. It used to be stored on the client and
// never forwarded, so the clipboard step was silently skipped.
func TestClient_SetCopyOnetimeCodeToClipboard(t *testing.T) {
	t.Parallel()
	onetime := &mockOnetimeCodeUI{}
	c := intdeviceflow.NewClient(&intdeviceflow.Input{OnetimeCodeUI: onetime})

	called := false
	c.SetCopyOnetimeCodeToClipboard(func(context.Context, string) error {
		called = true
		return nil
	})
	if onetime.clipboardFn == nil {
		t.Fatal("SetCopyOnetimeCodeToClipboard did not forward the function to the UI")
	}
	if err := onetime.clipboardFn(t.Context(), "code"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("the UI received a different function than the one set")
	}
}

// TestClient_Create_coordination verifies Create calls GetDeviceCode, then Show,
// then Poll, forwards the converted InputCreate and device code to Show, and returns
// the token with the expiration date computed from Now.
func TestClient_Create_coordination(t *testing.T) {
	t.Parallel()

	// synctest runs Create under a fake clock that starts at 2000-01-01 00:00:00 UTC
	// and only advances when goroutines block on a timer. The mocked path has none, so
	// create.go's time.Now() is fixed at the epoch and the expiration date is
	// deterministic without re-introducing a Now seam.
	synctest.Test(t, func(t *testing.T) {
		var calls []string
		deviceCode := &pubdeviceflow.DeviceCodeResponse{
			UserCode:        "USER-CODE",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       10,
		}
		df := &mockDeviceFlow{
			calls:      &calls,
			deviceCode: deviceCode,
			token:      &deviceflow.AccessToken{AccessToken: "gho_testtoken123", ExpiresIn: 28800},
		}
		onetime := &mockOnetimeCodeUI{calls: &calls}
		fixedTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

		input := &intdeviceflow.Input{
			Stderr:        io.Discard,
			Logger:        log.NewLogger(),
			OnetimeCodeUI: onetime,
			Client:        df,
		}

		tk, err := intdeviceflow.NewClient(input).Create(t.Context(), slog.New(slog.DiscardHandler), &intdeviceflow.InputCreate{
			ClientID:          "test-client-id",
			AppName:           "my-app",
			SkipAccountPicker: true,
			OpenBrowser:       true,
			Clipboard:         true,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if diff := cmp.Diff([]string{"GetDeviceCode", "Show", "Poll"}, calls); diff != "" {
			t.Errorf("call order mismatch (-want +got):\n%s", diff)
		}
		if tk.AccessToken != "gho_testtoken123" {
			t.Errorf("AccessToken = %q, want gho_testtoken123", tk.AccessToken)
		}
		wantExpiration := fixedTime.Add(28800 * time.Second)
		if !tk.ExpirationDate.Equal(wantExpiration) {
			t.Errorf("ExpirationDate = %v, want %v", tk.ExpirationDate, wantExpiration)
		}

		wantInput := &ui.InputCreate{
			ClientID:          "test-client-id",
			AppName:           "my-app",
			SkipAccountPicker: true,
			OpenBrowser:       true,
			Clipboard:         true,
		}
		if diff := cmp.Diff(wantInput, onetime.gotInput); diff != "" {
			t.Errorf("Show input mismatch (-want +got):\n%s", diff)
		}
		if onetime.gotDeviceCode != deviceCode {
			t.Errorf("Show received device code %v, want %v", onetime.gotDeviceCode, deviceCode)
		}
	})
}

// TestClient_Create_emptyClientID verifies Create fails before contacting the device
// flow when no client ID is given.
func TestClient_Create_emptyClientID(t *testing.T) {
	t.Parallel()

	var calls []string
	df := &mockDeviceFlow{calls: &calls}
	input := &intdeviceflow.Input{
		Stderr:        io.Discard,
		Logger:        log.NewLogger(),
		OnetimeCodeUI: &mockOnetimeCodeUI{calls: &calls},
		Client:        df,
	}

	_, err := intdeviceflow.NewClient(input).Create(t.Context(), slog.New(slog.DiscardHandler), &intdeviceflow.InputCreate{ClientID: ""})
	if err == nil {
		t.Fatal("Create() expected an error, got nil")
	}
	if len(calls) != 0 {
		t.Errorf("device flow was contacted before the client ID check: %v", calls)
	}
}

// TestClient_Show verifies Show forwards to the injected OnetimeCodeUI with a
// converted ui.InputCreate and the given device code.
func TestClient_Show(t *testing.T) {
	t.Parallel()

	onetime := &mockOnetimeCodeUI{}
	input := &intdeviceflow.Input{
		Stderr:        io.Discard,
		Logger:        log.NewLogger(),
		OnetimeCodeUI: onetime,
	}
	deviceCode := &pubdeviceflow.DeviceCodeResponse{
		UserCode:        "USER-CODE",
		VerificationURI: "https://github.com/login/device",
	}

	if err := intdeviceflow.NewClient(input).Show(t.Context(), slog.New(slog.DiscardHandler), &intdeviceflow.InputCreate{
		ClientID:          "test-client-id",
		AppName:           "my-app",
		SkipAccountPicker: true,
		OpenBrowser:       true,
		Clipboard:         true,
	}, deviceCode); err != nil {
		t.Fatalf("Show() error = %v", err)
	}

	wantInput := &ui.InputCreate{
		ClientID:          "test-client-id",
		AppName:           "my-app",
		SkipAccountPicker: true,
		OpenBrowser:       true,
		Clipboard:         true,
	}
	if diff := cmp.Diff(wantInput, onetime.gotInput); diff != "" {
		t.Errorf("Show input mismatch (-want +got):\n%s", diff)
	}
	if onetime.gotDeviceCode != deviceCode {
		t.Errorf("Show received device code %v, want %v", onetime.gotDeviceCode, deviceCode)
	}
}

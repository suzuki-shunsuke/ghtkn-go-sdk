package deviceflow

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// recordingBrowser records whether Open was called. It does not implement the
// availabilityChecker interface, so the device flow treats it as "available".
type recordingBrowser struct {
	opened bool
}

func (b *recordingBrowser) Open(_ context.Context, _ *slog.Logger, _ string) error {
	b.opened = true
	return nil
}

// unavailableBrowser is a recordingBrowser that reports it can't open a browser.
type unavailableBrowser struct {
	recordingBrowser
}

func (b *unavailableBrowser) Available() bool { return false }

// successHandler serves a device code then an access token, so Create completes.
func successHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/device/code":
			json.NewEncoder(w).Encode(pubdeviceflow.DeviceCodeResponse{ //nolint:errcheck
				DeviceCode:      "device123",
				UserCode:        "USER-CODE",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       10,
				Interval:        1,
			})
		case "/login/oauth/access_token":
			json.NewEncoder(w).Encode(accessTokenResponse{ //nolint:errcheck
				AccessToken: "gho_testtoken123",
				ExpiresIn:   28800,
			})
		}
	}
}

func TestClient_Create_browser(t *testing.T) {
	t.Parallel()

	const manualMsg = "Open the following URL in your browser"

	tests := []struct {
		name       string
		setup      func() (pubdeviceflow.Browser, *bool) // browser and a pointer to its "opened" flag
		wantOpened bool
		wantManual bool // stderr shows the manual-open instruction
	}{
		{
			name:       "available browser is opened",
			setup:      func() (pubdeviceflow.Browser, *bool) { b := &recordingBrowser{}; return b, &b.opened },
			wantOpened: true,
			wantManual: false,
		},
		{
			name:       "unavailable browser asks the user to open the URL",
			setup:      func() (pubdeviceflow.Browser, *bool) { b := &unavailableBrowser{}; return b, &b.opened },
			wantOpened: false,
			wantManual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(successHandler())
			defer server.Close()

			br, opened := tt.setup()
			var stderr strings.Builder
			input := &Input{
				HTTPClient:    &http.Client{Transport: &testTransport{server: server, base: http.DefaultTransport}},
				Now:           time.Now,
				Stderr:        &stderr,
				Browser:       br,
				NewTicker:     func(_ time.Duration) *time.Ticker { return time.NewTicker(time.Millisecond) },
				Logger:        log.NewLogger(),
				OnetimeCodeUI: newOnetimeCodeUI(strings.NewReader("\n"), &stderr, &mockWaiter{}),
			}

			tk, err := NewClient(input).Create(t.Context(), slog.New(slog.DiscardHandler), &InputCreate{ClientID: "test-client-id"})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if tk.AccessToken != "gho_testtoken123" {
				t.Fatalf("AccessToken = %q, want gho_testtoken123", tk.AccessToken)
			}
			if *opened != tt.wantOpened {
				t.Errorf("browser opened = %v, want %v", *opened, tt.wantOpened)
			}
			if got := strings.Contains(stderr.String(), manualMsg); got != tt.wantManual {
				t.Errorf("manual-open instruction shown = %v, want %v\nstderr:\n%s", got, tt.wantManual, stderr.String())
			}
		})
	}
}

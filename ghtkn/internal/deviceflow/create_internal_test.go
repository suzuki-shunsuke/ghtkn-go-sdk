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
	url    string
}

func (b *recordingBrowser) Open(_ context.Context, _ *slog.Logger, rawURL string) error {
	b.opened = true
	b.url = rawURL
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
		name              string
		setup             func() (pubdeviceflow.Browser, *bool, *string) // browser, pointer to opened flag, pointer to recorded URL
		openBrowser       bool
		skipAccountPicker bool
		wantOpened        bool
		wantManual        bool   // stderr shows the manual-open instruction
		wantBrowserURL    string // if non-empty, browser must have opened this exact URL
		wantStderrURL     string // if non-empty, stderr must contain this URL
	}{
		{
			name:           "available browser is opened",
			setup:          func() (pubdeviceflow.Browser, *bool, *string) { b := &recordingBrowser{}; return b, &b.opened, &b.url },
			openBrowser:    true,
			wantOpened:     true,
			wantManual:     false,
			wantBrowserURL: "https://github.com/login/device",
		},
		{
			name: "unavailable browser asks the user to open the URL",
			setup: func() (pubdeviceflow.Browser, *bool, *string) {
				b := &unavailableBrowser{}
				return b, &b.opened, &b.url
			},
			openBrowser: true,
			wantOpened:  false,
			wantManual:  true,
		},
		{
			name:              "skip_account_picker appended to verification URL",
			setup:             func() (pubdeviceflow.Browser, *bool, *string) { b := &recordingBrowser{}; return b, &b.opened, &b.url },
			openBrowser:       true,
			skipAccountPicker: true,
			wantOpened:        true,
			wantBrowserURL:    "https://github.com/login/device?skip_account_picker=true",
			wantStderrURL:     "https://github.com/login/device?skip_account_picker=true",
		},
		{
			name:        "open browser disabled asks the user to open the URL",
			setup:       func() (pubdeviceflow.Browser, *bool, *string) { b := &recordingBrowser{}; return b, &b.opened, &b.url },
			openBrowser: false,
			wantOpened:  false,
			wantManual:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(successHandler())
			defer server.Close()

			br, opened, browserURL := tt.setup()
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

			tk, err := NewClient(input).Create(t.Context(), slog.New(slog.DiscardHandler), &InputCreate{
				ClientID:          "test-client-id",
				SkipAccountPicker: tt.skipAccountPicker,
				OpenBrowser:       tt.openBrowser,
			})
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
			if tt.wantBrowserURL != "" && *browserURL != tt.wantBrowserURL {
				t.Errorf("browser URL = %q, want %q", *browserURL, tt.wantBrowserURL)
			}
			if tt.wantStderrURL != "" && !strings.Contains(stderr.String(), tt.wantStderrURL) {
				t.Errorf("prompt does not contain %q:\n%s", tt.wantStderrURL, stderr.String())
			}
		})
	}
}

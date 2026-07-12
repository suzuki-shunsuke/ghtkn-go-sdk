package ui

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
)

// recordingBrowser records whether Open was called. It does not implement the
// availabilityChecker interface, so Show treats it as "available".
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

func newTestDeviceCode() *pubdeviceflow.DeviceCodeResponse {
	return &pubdeviceflow.DeviceCodeResponse{
		DeviceCode:      "device123",
		UserCode:        "USER-CODE",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       10,
		Interval:        1,
	}
}

func TestClient_Show_clipboard(t *testing.T) {
	t.Parallel()

	const copiedMsg = "copied to your clipboard"

	tests := []struct {
		name       string
		enabled    bool // InputCreate.Clipboard
		inject     bool // whether a copy function is injected
		copyErr    bool // the injected copy function returns an error
		wantCalled bool // the copy function is invoked
		wantCopied bool // stderr shows the "copied" line
	}{
		{
			name:    "disabled with a function injected does not copy",
			enabled: false,
			inject:  true,
		},
		{
			name:    "enabled without a function injected does not show the copied line",
			enabled: true,
			inject:  false,
		},
		{
			name:       "enabled with a successful copy shows the copied line",
			enabled:    true,
			inject:     true,
			wantCalled: true,
			wantCopied: true,
		},
		{
			name:       "enabled but copy fails does not abort and does not show the copied line",
			enabled:    true,
			inject:     true,
			copyErr:    true,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotCode string
			var copyFn pubdeviceflow.CopyTextToClipboard
			if tt.inject {
				copyFn = func(_ context.Context, code string) error {
					gotCode = code
					if tt.copyErr {
						return errors.New("clipboard unavailable")
					}
					return nil
				}
			}

			var stderr strings.Builder
			client := New(&Input{
				Now:                        time.Now,
				Stderr:                     &stderr,
				Browser:                    &recordingBrowser{},
				OnetimeCodeUI:              newOnetimeCodeUI(strings.NewReader("\n"), &stderr, &mockWaiter{}),
				CopyOnetimeCodeToClipboard: copyFn,
			})

			if err := client.Show(t.Context(), slog.New(slog.DiscardHandler), &InputCreate{
				ClientID:    "test-client-id",
				OpenBrowser: true,
				Clipboard:   tt.enabled,
			}, newTestDeviceCode()); err != nil {
				t.Fatalf("Show() error = %v", err)
			}
			called := gotCode != ""
			if called != tt.wantCalled {
				t.Errorf("copy function called = %v, want %v", called, tt.wantCalled)
			}
			if tt.wantCalled && gotCode != "USER-CODE" {
				t.Errorf("copied code = %q, want USER-CODE", gotCode)
			}
			if got := strings.Contains(stderr.String(), copiedMsg); got != tt.wantCopied {
				t.Errorf("copied line shown = %v, want %v\nstderr:\n%s", got, tt.wantCopied, stderr.String())
			}
		})
	}
}

func TestClient_Show_browser(t *testing.T) {
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
			name:              "skip_account_picker shown in the manual-open URL",
			setup:             func() (pubdeviceflow.Browser, *bool, *string) { b := &recordingBrowser{}; return b, &b.opened, &b.url },
			openBrowser:       false,
			skipAccountPicker: true,
			wantOpened:        false,
			wantManual:        true,
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

			br, opened, browserURL := tt.setup()
			var stderr strings.Builder
			client := New(&Input{
				Now:           time.Now,
				Stderr:        &stderr,
				Browser:       br,
				OnetimeCodeUI: newOnetimeCodeUI(strings.NewReader("\n"), &stderr, &mockWaiter{}),
			})

			if err := client.Show(t.Context(), slog.New(slog.DiscardHandler), &InputCreate{
				ClientID:          "test-client-id",
				SkipAccountPicker: tt.skipAccountPicker,
				OpenBrowser:       tt.openBrowser,
			}, newTestDeviceCode()); err != nil {
				t.Fatalf("Show() error = %v", err)
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

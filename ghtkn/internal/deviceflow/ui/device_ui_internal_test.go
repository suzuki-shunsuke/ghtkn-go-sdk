package ui

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
)

type mockWaiter struct {
	err error
}

func (w *mockWaiter) Wait(ctx context.Context, duration time.Duration) error {
	return w.err
}

func TestSimpleOnetimeCodeUI_Show(t *testing.T) {
	t.Parallel()

	deviceCode := &pubdeviceflow.DeviceCodeResponse{
		UserCode:        "ABCD-1234",
		VerificationURI: "https://github.com/login/device",
	}
	expirationDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	expiration := expirationDate.Format(time.RFC3339)

	tests := []struct {
		name        string
		appName     string
		wantContain []string
		wantExclude []string
	}{
		{
			name:    "without app name",
			appName: "",
			wantContain: []string{
				"Copy your one-time code: ABCD-1234",
				"This code is valid until " + expiration,
				deviceCode.VerificationURI,
			},
			wantExclude: []string{
				"App Name:",
				"%!", // no leftover/EXTRA format args
			},
		},
		{
			name:    "with app name",
			appName: "my-app",
			wantContain: []string{
				"Copy your one-time code: ABCD-1234",
				"App Name: my-app",
				"This code is valid until " + expiration,
				deviceCode.VerificationURI,
			},
			wantExclude: []string{
				"%!",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stderr := &bytes.Buffer{}
			ui := newOnetimeCodeUI(strings.NewReader(""), stderr, &mockWaiter{})
			// OpenBrowser:false takes the deterministic branch that does not depend
			// on whether stdin is a terminal.
			if err := ui.Show(t.Context(), slog.New(slog.DiscardHandler), deviceCode, expirationDate, &pubdeviceflow.InputShow{
				OpenBrowser: false,
				AppName:     tt.appName,
			}); err != nil {
				t.Fatalf("Show() error = %v", err)
			}
			got := stderr.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\noutput:\n%s", want, got)
				}
			}
			for _, exclude := range tt.wantExclude {
				if strings.Contains(got, exclude) {
					t.Errorf("output unexpectedly contains %q\noutput:\n%s", exclude, got)
				}
			}
		})
	}
}

package config_test

import (
	"testing"

	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Parallel()
	bp := func(b bool) *bool { return &b }
	tests := []struct {
		name        string
		env         map[string]string
		cfg         *pubconfig.Config
		wantBackend string
		wantMinExp  string
		wantOpen    *bool
		wantClip    *bool
	}{
		{
			name: "no env leaves the config untouched",
			env:  map[string]string{},
			cfg:  &pubconfig.Config{Backend: &pubconfig.Backend{Type: "text"}, MinExpiration: "10m"},
			// existing file values are preserved
			wantBackend: "text", wantMinExp: "10m",
		},
		{
			name:        "GHTKN_BACKEND overrides the file value",
			env:         map[string]string{"GHTKN_BACKEND": "agent"},
			cfg:         &pubconfig.Config{Backend: &pubconfig.Backend{Type: "text"}},
			wantBackend: "agent",
		},
		{
			name:        "GHTKN_BACKEND allocates Backend when absent",
			env:         map[string]string{"GHTKN_BACKEND": "agent"},
			cfg:         &pubconfig.Config{},
			wantBackend: "agent",
		},
		{
			name:       "GHTKN_MIN_EXPIRATION overrides the file value",
			env:        map[string]string{"GHTKN_MIN_EXPIRATION": "1h"},
			cfg:        &pubconfig.Config{MinExpiration: "10m"},
			wantMinExp: "1h",
		},
		{
			name:     "GHTKN_OPEN_BROWSER=false disables",
			env:      map[string]string{"GHTKN_OPEN_BROWSER": "false"},
			cfg:      &pubconfig.Config{},
			wantOpen: bp(false),
		},
		{
			name:     "GHTKN_OPEN_BROWSER any non-false enables",
			env:      map[string]string{"GHTKN_OPEN_BROWSER": "0"},
			cfg:      &pubconfig.Config{},
			wantOpen: bp(true),
		},
		{
			name:     "GHTKN_OPEN_BROWSER is case-sensitive (FALSE enables)",
			env:      map[string]string{"GHTKN_OPEN_BROWSER": "FALSE"},
			cfg:      &pubconfig.Config{},
			wantOpen: bp(true),
		},
		{
			name:     "GHTKN_CLIPBOARD=true enables",
			env:      map[string]string{"GHTKN_CLIPBOARD": "true"},
			cfg:      &pubconfig.Config{},
			wantClip: bp(true),
		},
		{
			name:     "GHTKN_CLIPBOARD any non-true stays disabled",
			env:      map[string]string{"GHTKN_CLIPBOARD": "1"},
			cfg:      &pubconfig.Config{},
			wantClip: bp(false),
		},
		{
			name:     "GHTKN_CLIPBOARD overrides the file value",
			env:      map[string]string{"GHTKN_CLIPBOARD": "false"},
			cfg:      &pubconfig.Config{Clipboard: &pubconfig.Clipboard{Enable: bp(true)}},
			wantClip: bp(false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config.ApplyEnvOverrides(tt.cfg, func(k string) string { return tt.env[k] })

			gotBackend := ""
			if tt.cfg.Backend != nil {
				gotBackend = tt.cfg.Backend.Type
			}
			if gotBackend != tt.wantBackend {
				t.Errorf("backend.type = %q, want %q", gotBackend, tt.wantBackend)
			}
			if tt.cfg.MinExpiration != tt.wantMinExp {
				t.Errorf("min_expiration = %q, want %q", tt.cfg.MinExpiration, tt.wantMinExp)
			}
			var gotOpen *bool
			if tt.cfg.OpenBrowser != nil {
				gotOpen = tt.cfg.OpenBrowser.Enable
			}
			assertEnable(t, "open_browser", gotOpen, tt.wantOpen)
			var gotClip *bool
			if tt.cfg.Clipboard != nil {
				gotClip = tt.cfg.Clipboard.Enable
			}
			assertEnable(t, "clipboard", gotClip, tt.wantClip)
		})
	}
}

func assertEnable(t *testing.T, field string, got, want *bool) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Errorf("%s enable: got=%v want=%v (nil mismatch)", field, got, want)
	case *got != *want:
		t.Errorf("%s enable = %v, want %v", field, *got, *want)
	}
}

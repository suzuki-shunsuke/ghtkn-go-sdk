package ghtkn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
)

func Test_loadConfig(t *testing.T) {
	t.Parallel()

	// A config file whose backend.type is "text" and that also sets open_browser and
	// min_expiration, so tests can prove env overrides win over file values.
	const cfgYAML = `apps:
  - name: example
    client_id: Iv1.example
backend:
  type: text
min_expiration: 30m
open_browser:
  enable: true
`
	writeConfig := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "ghtkn.yaml")
		if err := os.WriteFile(path, []byte(cfgYAML), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	tests := []struct {
		name            string
		env             map[string]string
		writeFile       bool
		wantBackend     string
		wantMinExp      string
		wantOpenBrowser *bool // nil means "not set"
		wantClipboard   *bool
	}{
		{
			name:            "file only",
			writeFile:       true,
			wantBackend:     "text",
			wantMinExp:      "30m",
			wantOpenBrowser: ptr(true),
		},
		{
			name:            "GHTKN_BACKEND overrides the file",
			writeFile:       true,
			env:             map[string]string{"GHTKN_BACKEND": "agent"},
			wantBackend:     "agent",
			wantMinExp:      "30m",
			wantOpenBrowser: ptr(true),
		},
		{
			name:            "all env overrides win over the file",
			writeFile:       true,
			env:             map[string]string{"GHTKN_BACKEND": "agent", "GHTKN_MIN_EXPIRATION": "1h", "GHTKN_OPEN_BROWSER": "false", "GHTKN_CLIPBOARD": "true"},
			wantBackend:     "agent",
			wantMinExp:      "1h",
			wantOpenBrowser: ptr(false),
			wantClipboard:   ptr(true),
		},
		{
			name:        "no config file, no env -> empty",
			writeFile:   false,
			wantBackend: "",
			wantMinExp:  "",
		},
		{
			name:        "no config file, env still applies",
			writeFile:   false,
			env:         map[string]string{"GHTKN_BACKEND": "agent"},
			wantBackend: "agent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := map[string]string{}
			for k, v := range tt.env {
				env[k] = v
			}
			if tt.writeFile {
				env["GHTKN_CONFIG"] = writeConfig(t)
			} else {
				env["GHTKN_CONFIG"] = filepath.Join(t.TempDir(), "absent.yaml")
			}
			getEnv := func(k string) string { return env[k] }

			cfg, err := loadConfig(getEnv, "linux")
			if err != nil {
				t.Fatalf("loadConfig() unexpected error: %v", err)
			}

			if got := backendType(cfg); got != tt.wantBackend {
				t.Errorf("backend.type = %q, want %q", got, tt.wantBackend)
			}
			if cfg.MinExpiration != tt.wantMinExp {
				t.Errorf("min_expiration = %q, want %q", cfg.MinExpiration, tt.wantMinExp)
			}
			assertBoolPtr(t, "open_browser.enable", openBrowserEnable(cfg), tt.wantOpenBrowser)
			assertBoolPtr(t, "clipboard.enable", clipboardEnable(cfg), tt.wantClipboard)
		})
	}
}

func ptr[T any](v T) *T { return &v }

func backendType(cfg *config.Config) string {
	if cfg.Backend == nil {
		return ""
	}
	return cfg.Backend.Type
}

func openBrowserEnable(cfg *config.Config) *bool {
	if cfg.OpenBrowser == nil {
		return nil
	}
	return cfg.OpenBrowser.Enable
}

func clipboardEnable(cfg *config.Config) *bool {
	if cfg.Clipboard == nil {
		return nil
	}
	return cfg.Clipboard.Enable
}

func assertBoolPtr(t *testing.T, field string, got, want *bool) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Errorf("%s = %s, want %s", field, fmtBoolPtr(got), fmtBoolPtr(want))
	case *got != *want:
		t.Errorf("%s = %v, want %v", field, *got, *want)
	}
}

func fmtBoolPtr(b *bool) string {
	if b == nil {
		return "<nil>"
	}
	if *b {
		return "true"
	}
	return "false"
}

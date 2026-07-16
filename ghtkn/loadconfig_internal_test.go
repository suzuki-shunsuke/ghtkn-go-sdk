package ghtkn

import (
	"maps"
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
		invalidYAML     bool // write a malformed config file to exercise the Read error path
		wantErr         bool
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
			wantOpenBrowser: new(true),
		},
		{
			name:            "GHTKN_BACKEND overrides the file",
			writeFile:       true,
			env:             map[string]string{"GHTKN_BACKEND": "agent"},
			wantBackend:     "agent",
			wantMinExp:      "30m",
			wantOpenBrowser: new(true),
		},
		{
			name:            "all env overrides win over the file",
			writeFile:       true,
			env:             map[string]string{"GHTKN_BACKEND": "agent", "GHTKN_MIN_EXPIRATION": "1h", "GHTKN_OPEN_BROWSER": "false", "GHTKN_CLIPBOARD": "true"},
			wantBackend:     "agent",
			wantMinExp:      "1h",
			wantOpenBrowser: new(false),
			wantClipboard:   new(true),
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
		{
			name:      "unparsable boolean env errors",
			writeFile: false,
			env:       map[string]string{"GHTKN_OPEN_BROWSER": "yes"},
			wantErr:   true,
		},
		{
			name:        "malformed config file errors",
			invalidYAML: true,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := map[string]string{}
			maps.Copy(env, tt.env)
			switch {
			case tt.invalidYAML:
				path := filepath.Join(t.TempDir(), "ghtkn.yaml")
				if err := os.WriteFile(path, []byte("apps: [ this is not valid yaml"), 0o600); err != nil {
					t.Fatal(err)
				}
				env["GHTKN_CONFIG"] = path
			case tt.writeFile:
				env["GHTKN_CONFIG"] = writeConfig(t)
			default:
				env["GHTKN_CONFIG"] = filepath.Join(t.TempDir(), "absent.yaml")
			}
			getEnv := func(k string) string { return env[k] }

			cfg, err := loadConfig(getEnv, "linux")
			if tt.wantErr {
				if err == nil {
					t.Fatal("loadConfig() expected an error, got nil")
				}
				return
			}
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

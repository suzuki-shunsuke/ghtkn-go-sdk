package ghtkn

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_checkBoolEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{name: "true", input: "true", want: true},
		{name: "1", input: "1", want: true},
		{name: "false", input: "false", want: false},
		{name: "0", input: "0", want: false},
		{name: "invalid value", input: "yes", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := checkBoolEnv(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkBoolEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("checkBoolEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_enabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *InputEnabled
		env     map[string]string
		want    bool
		wantErr bool
	}{
		{
			name:  "Envs: first set variable wins (true)",
			input: &InputEnabled{Envs: []string{"A", "B"}},
			env:   map[string]string{"A": "true", "B": "false"},
			want:  true,
		},
		{
			name:  "Envs: first set variable wins (false)",
			input: &InputEnabled{Envs: []string{"A"}},
			env:   map[string]string{"A": "false"},
			want:  false,
		},
		{
			name:  "Envs: skips unset variables and uses the next one",
			input: &InputEnabled{Envs: []string{"A", "B"}},
			env:   map[string]string{"B": "1"},
			want:  true,
		},
		{
			name:    "Envs: invalid value errors",
			input:   &InputEnabled{Envs: []string{"A"}},
			env:     map[string]string{"A": "yes"},
			wantErr: true,
		},
		{
			name:  "Envs take precedence over GHTKN_ENABLE",
			input: &InputEnabled{Envs: []string{"A"}},
			env:   map[string]string{"A": "false", "GHTKN_ENABLE": "true"},
			want:  false,
		},
		{
			name: "falls back to GHTKN_ENABLE (true)",
			env:  map[string]string{"GHTKN_ENABLE": "true"},
			want: true,
		},
		{
			name: "falls back to GHTKN_ENABLE (false)",
			env:  map[string]string{"GHTKN_ENABLE": "false"},
			want: false,
		},
		{
			name:    "GHTKN_ENABLE invalid value errors",
			env:     map[string]string{"GHTKN_ENABLE": "maybe"},
			wantErr: true,
		},
		{
			name:    "no env set and config path cannot be resolved",
			env:     map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := enabled(func(k string) string { return tt.env[k] }, tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("enabled() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_enabled_configFile covers the config-file-existence branch by pointing
// GHTKN_CONFIG at a real path under a temp directory.
func Test_enabled_configFile(t *testing.T) {
	t.Parallel()

	t.Run("enabled when the config file exists", func(t *testing.T) {
		t.Parallel()

		p := filepath.Join(t.TempDir(), "ghtkn.yaml")
		if err := os.WriteFile(p, []byte("apps: []\n"), 0o600); err != nil {
			t.Fatalf("write config file: %v", err)
		}
		env := map[string]string{"GHTKN_CONFIG": p}
		got, err := enabled(func(k string) string { return env[k] }, nil)
		if err != nil {
			t.Fatalf("enabled() error = %v", err)
		}
		if !got {
			t.Error("enabled() = false, want true")
		}
	})

	t.Run("disabled when the config file does not exist", func(t *testing.T) {
		t.Parallel()

		env := map[string]string{"GHTKN_CONFIG": filepath.Join(t.TempDir(), "does-not-exist.yaml")}
		got, err := enabled(func(k string) string { return env[k] }, nil)
		if err != nil {
			t.Fatalf("enabled() error = %v", err)
		}
		if got {
			t.Error("enabled() = true, want false")
		}
	})
}

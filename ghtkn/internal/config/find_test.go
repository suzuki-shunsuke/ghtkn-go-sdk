package config_test

import (
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestGetPath(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name    string
		want    string
		envs    map[string]string
		goos    string
		wantErr bool
	}{
		// Linux/macOS tests
		{
			name: "Linux: standard XDG config path",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "/home/user/.config",
			},
			goos:    "linux",
			want:    filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "macOS: custom XDG config path",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "/custom/config/dir",
			},
			goos:    "darwin",
			want:    filepath.Join("/custom", "config", "dir", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: fallback to HOME when XDG not set",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "",
				"HOME":            "/home/user",
			},
			goos:    "linux",
			want:    filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: error when both XDG and HOME are empty",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "",
				"HOME":            "",
			},
			goos:    "linux",
			want:    "",
			wantErr: true,
		},
		{
			name: "macOS: XDG config with app field set",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "/home/user/.config",
			},
			goos:    "darwin",
			want:    filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: root config path",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "/",
			},
			goos:    "linux",
			want:    filepath.Join("/", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: relative path",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "relative/config",
			},
			goos:    "linux",
			want:    filepath.Join("relative", "config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "macOS: path with spaces",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "/path with spaces/config",
			},
			goos:    "darwin",
			want:    filepath.Join("/path with spaces", "config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: HOME fallback with spaces",
			envs: map[string]string{
				"XDG_CONFIG_HOME": "",
				"HOME":            "/home with spaces/user",
			},
			goos:    "linux",
			want:    filepath.Join("/home with spaces", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},

		// Windows tests
		{
			name: "Windows: with AppData",
			envs: map[string]string{
				"APPDATA": filepath.Join("C:", "Users", "testuser", "AppData", "Roaming"),
			},
			goos:    "windows",
			want:    filepath.Join("C:", "Users", "testuser", "AppData", "Roaming", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Windows: without AppData",
			envs: map[string]string{
				"APPDATA": "",
			},
			goos:    "windows",
			want:    "",
			wantErr: true,
		},
		{
			name: "Windows: custom AppData path",
			envs: map[string]string{
				"APPDATA": filepath.Join("D:", "CustomAppData"),
			},
			goos:    "windows",
			want:    filepath.Join("D:", "CustomAppData", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(key string) string {
				return tt.envs[key]
			}
			got, err := config.GetPath(getEnv, tt.goos)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

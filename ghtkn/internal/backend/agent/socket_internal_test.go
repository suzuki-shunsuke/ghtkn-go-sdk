package agent

import (
	"path/filepath"
	"testing"
)

func TestSocketPath(t *testing.T) {
	t.Parallel()
	data := []struct {
		name    string
		env     map[string]string
		goos    string
		want    string
		wantErr bool
	}{
		{
			name: "explicit socket override",
			env:  map[string]string{"GHTKN_AGENT_SOCKET": "/tmp/custom.sock"},
			goos: "linux",
			want: "/tmp/custom.sock",
		},
		{
			name: "override beats xdg",
			env:  map[string]string{"GHTKN_AGENT_SOCKET": "/o.sock", "XDG_RUNTIME_DIR": "/run/user/1000"},
			goos: "linux",
			want: "/o.sock",
		},
		{
			name: "xdg runtime dir",
			env:  map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"},
			goos: "linux",
			want: "/run/user/1000/ghtkn/socket",
		},
		{
			name: "xdg cache home fallback",
			env:  map[string]string{"XDG_CACHE_HOME": "/home/me/.cache"},
			goos: "linux",
			want: "/home/me/.cache/ghtkn/agent.sock",
		},
		{
			name: "home fallback",
			env:  map[string]string{"HOME": "/home/me"},
			goos: "linux",
			want: "/home/me/.cache/ghtkn/agent.sock",
		},
		{
			name:    "nothing set",
			env:     map[string]string{},
			goos:    "linux",
			wantErr: true,
		},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(k string) string { return d.env[k] }
			got, err := socketPath(getEnv, d.goos)
			if d.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != filepath.FromSlash(d.want) {
				t.Fatalf("socketPath = %q, want %q", got, d.want)
			}
		})
	}
}

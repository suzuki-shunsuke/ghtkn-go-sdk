package agent

import (
	"errors"
	"path/filepath"
)

// goosWindows is the runtime.GOOS value for Windows.
const goosWindows = "windows"

// socketPath resolves the path of the agent's Unix domain socket. GHTKN_AGENT_SOCKET
// takes precedence; otherwise it prefers $XDG_RUNTIME_DIR/ghtkn/socket and falls back
// to $XDG_CACHE_HOME/ghtkn/agent.sock, then $HOME/.cache/ghtkn/agent.sock. This
// mirrors how the agent server resolves the socket path.
func socketPath(getEnv func(string) string, goos string) (string, error) {
	if s := getEnv("GHTKN_AGENT_SOCKET"); s != "" {
		return s, nil
	}
	if dir := getEnv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "ghtkn", "socket"), nil
	}
	if dir := getEnv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "ghtkn", "agent.sock"), nil
	}
	if goos == goosWindows {
		if d := getEnv("LocalAppData"); d != "" {
			return filepath.Join(d, "cache", "ghtkn", "agent.sock"), nil
		}
		return "", errors.New("GHTKN_AGENT_SOCKET or LocalAppData is required to use the agent backend on Windows")
	}
	if home := getEnv("HOME"); home != "" {
		return filepath.Join(home, ".cache", "ghtkn", "agent.sock"), nil
	}
	return "", errors.New("GHTKN_AGENT_SOCKET, XDG_RUNTIME_DIR, XDG_CACHE_HOME, or HOME is required to use the agent backend")
}

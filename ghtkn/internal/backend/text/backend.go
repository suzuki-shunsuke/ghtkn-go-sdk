// Package text provides a plaintext file backend for GitHub access tokens.
// Tokens are stored unencrypted under the user's cache directory, protected
// only by file permissions. It targets environments where the OS keyring is
// unavailable, such as containers and VMs.
package text

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// The access token is saved in plaintext to ${XDG_CACHE_HOME}/ghtkn/tokens/<client-id>.
// The directory can be overridden with the GHTKN_TEXT_BACKEND_DIR environment variable.
// The file permissions are 0600. No encryption is performed.
// See https://github.com/suzuki-shunsuke/design-docs/blob/main/ghtkn/backend/README.md

// Backend stores access tokens as plaintext files under dir.
type Backend struct {
	dir string
}

// New creates a text backend. The storage directory is GHTKN_TEXT_BACKEND_DIR
// if set, otherwise ${XDG_CACHE_HOME}/ghtkn/tokens (falling back to $HOME/.cache).
// It returns an error if none of these are set.
func New() (*Backend, error) {
	dir, err := tokenDir(os.Getenv)
	if err != nil {
		return nil, err
	}
	return &Backend{
		dir: dir,
	}, nil
}

// tokenDir resolves the directory that stores token files. GHTKN_TEXT_BACKEND_DIR
// takes precedence; otherwise it is ${cache dir}/ghtkn/tokens.
func tokenDir(getEnv func(string) string) (string, error) {
	dir := getEnv("GHTKN_TEXT_BACKEND_DIR")
	if dir != "" {
		return dir, nil
	}
	cacheDir, err := cacheDir(getEnv)
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "ghtkn", "tokens"), nil
}

// cacheDir resolves the base cache directory, honoring XDG_CACHE_HOME and
// falling back to $HOME/.cache, mirroring how the config path is resolved.
func cacheDir(getEnv func(string) string) (string, error) {
	if d := getEnv("XDG_CACHE_HOME"); d != "" {
		return d, nil
	}
	if home := getEnv("HOME"); home != "" {
		return filepath.Join(home, ".cache"), nil
	}
	return "", errors.New("XDG_CACHE_HOME or HOME is required to use the text backend")
}

// Get reads the raw token stored for clientID.
// It returns (nil, nil) when no token file exists.
func (b *Backend) Get(_ context.Context, clientID string) ([]byte, error) {
	bt, err := os.ReadFile(filepath.Join(b.dir, clientID))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read a token file: %w", err)
	}
	return bt, nil
}

// Set writes the raw token for clientID atomically with file permission 0600.
// It writes to a temporary file in the same directory and renames it into place,
// so a concurrent writer can at worst lose its write but never corrupt the file.
func (b *Backend) Set(_ context.Context, clientID, token string) error {
	if err := os.MkdirAll(b.dir, 0o700); err != nil {
		return fmt.Errorf("create the token directory: %w", err)
	}
	tmp, err := os.CreateTemp(b.dir, clientID+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create a temporary file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(token); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write a token to a temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close the temporary file: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(b.dir, clientID)); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename the temporary file: %w", err)
	}
	return nil
}

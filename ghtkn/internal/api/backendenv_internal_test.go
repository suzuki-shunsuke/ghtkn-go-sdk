package api

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// oneAppConfigReader is a ConfigReader that returns a single app and no backend, so
// the backend can only come from the environment.
type oneAppConfigReader struct{}

func (r *oneAppConfigReader) Read(cfg *pubconfig.Config, _ string) error {
	cfg.Apps = []*pubconfig.App{{Name: "app1", ClientID: "Iv1.x"}}
	return nil
}

// TestTokenManager_Revoke_backendFromEnv verifies that Revoke resolves the backend
// from GHTKN_BACKEND, not from the config file alone. Revoke reads the config itself,
// and the backend type is applied to it as an environment override, so reading the
// file without folding the environment in would revoke from the default keyring while
// the token the user asked to revoke sits in the configured backend and stays live.
//
// The text backend stands in for a non-default backend because it is a plain file:
// the token can be seeded and the revoked token observed without a keyring or a
// running agent.
func TestTokenManager_Revoke_backendFromEnv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const token = "gho_from_text_backend" //nolint:gosec // G101: a fake token for the test
	if err := os.WriteFile(filepath.Join(dir, "Iv1.x"), []byte(`{"access_token":"`+token+`","expiration_date":"2999-01-01T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	getEnv := func(k string) string {
		switch k {
		case "GHTKN_BACKEND":
			return "text"
		case "GHTKN_TEXT_BACKEND_DIR":
			return dir
		default:
			return ""
		}
	}
	revoker := &mockRevoker{}
	tm := New(&Input{
		// Backend is deliberately left nil so the backend is resolved the way it is in
		// production.
		Revoker:      revoker,
		ConfigReader: &oneAppConfigReader{},
		Logger:       log.NewLogger(),
		Getenv:       getEnv,
		GOOS:         "linux",
	})

	if err := tm.Revoke(t.Context(), slog.New(slog.DiscardHandler), &pubapi.InputRevoke{
		AppNames:       []string{"app1"},
		ConfigFilePath: filepath.Join(dir, "ghtkn.yaml"),
	}); err != nil {
		t.Fatalf("Revoke() error: %v", err)
	}

	if len(revoker.revoked) != 1 || len(revoker.revoked[0]) != 1 || revoker.revoked[0][0] != token {
		t.Fatalf("revoked %v, want the token stored in the backend GHTKN_BACKEND selects (%q)", revoker.revoked, token)
	}
	// The revoked token is deleted from the backend it was read from.
	if _, err := os.Stat(filepath.Join(dir, "Iv1.x")); !os.IsNotExist(err) {
		t.Fatalf("the revoked token file must be deleted, stat err = %v", err)
	}
}

package ghtkn_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/socket"
)

// shortTempDir returns a temp directory under /tmp that is short enough to
// hold a Unix socket path within macOS's 104-byte sun_path limit.
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "g")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestNew(t *testing.T) {
	t.Setenv(socket.EnvSock, "")
	if c := ghtkn.New(); c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestClient_Get_socketMode(t *testing.T) {
	expires := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	// Spin up a fake daemon on a Unix socket.
	sockPath := filepath.Join(shortTempDir(t), "s")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	var gotApp, gotAuth string
	mux := http.NewServeMux()
	mux.HandleFunc(socket.PathToken, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		req := &socket.TokenRequest{}
		_ = json.NewDecoder(r.Body).Decode(req)
		gotApp = req.App
		_ = json.NewEncoder(w).Encode(&socket.TokenResponse{
			AccessToken:    "ghs_xxx",
			Login:          "octocat",
			ExpirationDate: expires,
		})
	})
	srv := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: mux, ReadHeaderTimeout: time.Second},
	}
	srv.Start()
	t.Cleanup(srv.Close)

	t.Setenv(socket.EnvSock, sockPath)
	t.Setenv(socket.EnvCapToken, "cap-token")

	client := ghtkn.New()
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	token, app, err := client.Get(t.Context(), logger, &ghtkn.InputGet{AppName: "my-app"})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	wantToken := &ghtkn.AccessToken{
		AccessToken:    "ghs_xxx",
		Login:          "octocat",
		ExpirationDate: expires,
	}
	if diff := cmp.Diff(wantToken, token); diff != "" {
		t.Errorf("token mismatch (-want +got):\n%s", diff)
	}
	if app == nil || app.Name != "my-app" {
		t.Errorf("app = %+v, want Name=my-app", app)
	}
	if gotApp != "my-app" {
		t.Errorf("daemon received app = %q, want my-app", gotApp)
	}
	if gotAuth != "Bearer cap-token" {
		t.Errorf("daemon received Authorization = %q, want Bearer cap-token", gotAuth)
	}
}

func TestClient_Get_socketMode_appFromEnv(t *testing.T) {
	expires := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	sockPath := filepath.Join(shortTempDir(t), "s")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	var gotApp string
	mux := http.NewServeMux()
	mux.HandleFunc(socket.PathToken, func(w http.ResponseWriter, r *http.Request) {
		req := &socket.TokenRequest{}
		_ = json.NewDecoder(r.Body).Decode(req)
		gotApp = req.App
		_ = json.NewEncoder(w).Encode(&socket.TokenResponse{
			AccessToken:    "ghs_xxx",
			ExpirationDate: expires,
		})
	})
	srv := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: mux, ReadHeaderTimeout: time.Second},
	}
	srv.Start()
	t.Cleanup(srv.Close)

	t.Setenv(socket.EnvSock, sockPath)
	t.Setenv(socket.EnvCapToken, "")
	t.Setenv("GHTKN_APP", "env-app")

	client := ghtkn.New()
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	if _, _, err := client.Get(t.Context(), logger, nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if gotApp != "env-app" {
		t.Errorf("daemon received app = %q, want env-app", gotApp)
	}
}

func TestClient_Setters_socketModeNoOp(t *testing.T) {
	t.Setenv(socket.EnvSock, "/nonexistent.sock")
	client := ghtkn.New()
	// Should not panic in socket mode even though the keyring backend is nil.
	client.SetLogger(&ghtkn.Logger{})
	client.SetDeviceCodeUI(nil)
	client.SetBrowser(nil)
}

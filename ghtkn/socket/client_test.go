package socket_test

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/socket"
)

// startUnixServer starts an httptest server bound to a Unix socket and returns
// the socket path. The server is closed on test cleanup.
//
// The socket lives in /tmp rather than t.TempDir() because macOS limits Unix
// socket paths to 104 bytes and t.TempDir() paths under /var/folders are too
// long when combined with verbose test names.
func startUnixServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "g")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "s")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	srv := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler, ReadHeaderTimeout: time.Second},
	}
	srv.Start()
	t.Cleanup(srv.Close)
	return sockPath
}

func TestFetchToken(t *testing.T) {
	t.Parallel()
	expires := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		capToken     string
		req          *socket.TokenRequest
		handler      http.HandlerFunc
		want         *socket.TokenResponse
		wantErr      bool
		wantErrAs    any
		wantStatus   int
		wantErrCode  string
		wantPath     string
		wantMethod   string
		wantApp      string
		wantAuthHdr  string
		wantContent  string
		nilRequest   bool
		emptyCapHdr  bool
		notValidJSON bool
	}{
		{
			name:     "success",
			capToken: "cap-123",
			req:      &socket.TokenRequest{App: "my-app"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != socket.PathToken {
					t.Errorf("path = %q, want %q", r.URL.Path, socket.PathToken)
				}
				if r.Method != http.MethodPost {
					t.Errorf("method = %q, want POST", r.Method)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer cap-123" {
					t.Errorf("Authorization = %q", got)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q", got)
				}
				body, _ := io.ReadAll(r.Body)
				gotReq := &socket.TokenRequest{}
				if err := json.Unmarshal(body, gotReq); err != nil {
					t.Errorf("decode body: %v", err)
				}
				if gotReq.App != "my-app" {
					t.Errorf("app = %q, want my-app", gotReq.App)
				}
				_ = json.NewEncoder(w).Encode(&socket.TokenResponse{
					AccessToken:    "ghs_xxx",
					Login:          "octocat",
					ExpirationDate: expires,
				})
			},
			want: &socket.TokenResponse{
				AccessToken:    "ghs_xxx",
				Login:          "octocat",
				ExpirationDate: expires,
			},
		},
		{
			name:     "no capability token omits Authorization header",
			capToken: "",
			req:      &socket.TokenRequest{App: "my-app"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "" {
					t.Errorf("Authorization should be empty, got %q", got)
				}
				_ = json.NewEncoder(w).Encode(&socket.TokenResponse{
					AccessToken:    "ghs_xxx",
					ExpirationDate: expires,
				})
			},
			want: &socket.TokenResponse{
				AccessToken:    "ghs_xxx",
				ExpirationDate: expires,
			},
		},
		{
			name:     "structured error response",
			capToken: "cap-123",
			req:      &socket.TokenRequest{App: "my-app"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(&socket.ErrorResponse{
					Code:    "forbidden",
					Message: "app not allowed",
				})
			},
			wantErr:     true,
			wantStatus:  http.StatusForbidden,
			wantErrCode: "forbidden",
		},
		{
			name:     "non-json error body",
			capToken: "cap-123",
			req:      &socket.TokenRequest{App: "my-app"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("boom"))
			},
			wantErr:    true,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "nil request",
			capToken:   "cap-123",
			req:        nil,
			handler:    func(_ http.ResponseWriter, _ *http.Request) { t.Error("handler should not be called") },
			wantErr:    true,
			nilRequest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sockPath := startUnixServer(t, tt.handler)
			got, err := socket.FetchToken(t.Context(), sockPath, tt.capToken, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantStatus != 0 {
					var sockErr *socket.Error
					if !errors.As(err, &sockErr) {
						t.Fatalf("error is not *socket.Error: %v", err)
					}
					if sockErr.Status != tt.wantStatus {
						t.Errorf("status = %d, want %d", sockErr.Status, tt.wantStatus)
					}
					if tt.wantErrCode != "" && sockErr.Code != tt.wantErrCode {
						t.Errorf("code = %q, want %q", sockErr.Code, tt.wantErrCode)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  *socket.Error
		want string
	}{
		{"with code", &socket.Error{Status: 403, Code: "forbidden", Message: "no"}, "ghtkn socket: 403 forbidden: no"},
		{"message only", &socket.Error{Status: 500, Message: "boom"}, "ghtkn socket: 500: boom"},
		{"status only", &socket.Error{Status: 502}, "ghtkn socket: 502"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

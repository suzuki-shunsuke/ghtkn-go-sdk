package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// fakeAgent listens on a Unix socket and serves one request per connection using
// handler, which receives the decoded request and returns the response to send.
type fakeAgent struct {
	socket   string
	listener net.Listener
	mu       sync.Mutex
	requests []*agentapi.Request
	// legacy makes the fake answer like a pre-versioning agent: it leaves
	// Response.ProtocolVersion unset, the way an agent that predates the field does.
	legacy bool
}

func startFakeAgent(t *testing.T, handler func(*agentapi.Request) *agentapi.Response) *fakeAgent {
	t.Helper()
	return startAgent(t, false, handler)
}

// startLegacyFakeAgent starts a fake agent that answers like a pre-versioning one: it
// never sets Response.ProtocolVersion.
func startLegacyFakeAgent(t *testing.T, handler func(*agentapi.Request) *agentapi.Response) *fakeAgent {
	t.Helper()
	return startAgent(t, true, handler)
}

func startAgent(t *testing.T, legacy bool, handler func(*agentapi.Request) *agentapi.Response) *fakeAgent {
	t.Helper()
	// Keep the socket path well under the platform's sun_path limit (104 bytes on
	// macOS): the default per-test TempDir embeds the (long) test name and can overflow.
	dir, err := os.MkdirTemp("", "gh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) }) //nolint:errcheck
	socket := filepath.Join(dir, "a.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeAgent{socket: socket, listener: listener, legacy: legacy}
	t.Cleanup(func() { listener.Close() }) //nolint:errcheck
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			f.serve(conn, handler)
		}
	}()
	return f
}

func (f *fakeAgent) serve(conn net.Conn, handler func(*agentapi.Request) *agentapi.Response) {
	defer conn.Close() //nolint:errcheck
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return
	}
	req := &agentapi.Request{}
	if err := json.Unmarshal(line, req); err != nil {
		return
	}
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	resp := handler(req)
	if !f.legacy {
		// A current agent stamps its protocol version on every response.
		resp.ProtocolVersion = agentapi.ProtocolVersion
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = conn.Write(append(b, '\n'))
}

func (f *fakeAgent) reqs() []*agentapi.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*agentapi.Request(nil), f.requests...)
}

func TestBackend_getHit(t *testing.T) {
	t.Parallel()
	value := `{"access_token":"abc","expiration_date":"2026-01-01T00:00:00Z"}`
	f := startFakeAgent(t, func(req *agentapi.Request) *agentapi.Response {
		if req.Command != agentapi.CommandGet || req.StartDeviceFlow {
			return &agentapi.Response{Error: "unexpected request"}
		}
		return &agentapi.Response{OK: true, Token: json.RawMessage(value)}
	})
	got, err := (&Backend{socket: f.socket}).Get(t.Context(), "Iv1.x")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(value, string(got)); diff != "" {
		t.Fatalf("token (-want +got):\n%s", diff)
	}
}

func TestBackend_getMiss(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespNotFound}
	})
	got, err := (&Backend{socket: f.socket}).Get(t.Context(), "Iv1.absent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("miss must return nil, got %q", got)
	}
}

// TestBackend_getWarning verifies that a security warning on the response is surfaced to
// the configured writer (stderr in production) so the user sees it, while the token is
// still returned.
func TestBackend_getWarning(t *testing.T) {
	t.Parallel()
	value := `{"access_token":"abc","expiration_date":"2026-01-01T00:00:00Z"}`
	const warning = "a still-valid refresh token failed to refresh"
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{OK: true, Token: json.RawMessage(value), Warning: warning}
	})
	var buf bytes.Buffer
	got, err := (&Backend{socket: f.socket, warn: &buf}).Get(t.Context(), "Iv1.x")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(value, string(got)); diff != "" {
		t.Fatalf("token (-want +got):\n%s", diff)
	}
	if !strings.Contains(buf.String(), warning) {
		t.Fatalf("the warning must be surfaced to the user; got %q", buf.String())
	}
}

// TestBackend_getWarning_customHook verifies that a consumer-provided AgentWarning log
// hook is honored, so the agent's warning can be re-routed or reformatted instead of the
// default stderr line.
func TestBackend_getWarning_customHook(t *testing.T) {
	t.Parallel()
	value := `{"access_token":"abc","expiration_date":"2026-01-01T00:00:00Z"}`
	const warning = "a still-valid refresh token failed to refresh"
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{OK: true, Token: json.RawMessage(value), Warning: warning}
	})
	var got string
	logger := &publog.Logger{
		AgentWarning: func(_ *slog.Logger, _ io.Writer, message string) {
			got = message
		},
	}
	if _, err := (&Backend{socket: f.socket, logger: logger}).Get(t.Context(), "Iv1.x"); err != nil {
		t.Fatal(err)
	}
	if got != warning {
		t.Fatalf("the custom AgentWarning hook must receive the warning; got %q, want %q", got, warning)
	}
}

// TestBackend_getProbeShape guards that Get is a pure probe: it must not ask the agent
// to start a device flow.
func TestBackend_getProbeShape(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespNotFound}
	})
	if _, err := (&Backend{socket: f.socket}).Get(t.Context(), "Iv1.x"); err != nil {
		t.Fatal(err)
	}
	reqs := f.reqs()
	if len(reqs) != 1 || reqs[0].Command != agentapi.CommandGet || reqs[0].StartDeviceFlow {
		t.Fatalf("unexpected request: %+v", reqs)
	}
}

func TestBackend_serverError(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: "boom"}
	})
	if _, err := (&Backend{socket: f.socket}).Get(t.Context(), "Iv1.x"); err == nil {
		t.Fatal("a server error response must produce an error")
	}
}

func TestBackend_agentNotRunning(t *testing.T) {
	t.Parallel()
	socket := filepath.Join(t.TempDir(), "absent.sock")
	if _, err := (&Backend{socket: socket}).Get(t.Context(), "Iv1.x"); !agentapi.IsNotRunning(err) {
		t.Fatalf("Get err = %v, want ErrAgentNotRunning", err)
	}
}

func TestBackend_locked(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespLocked}
	})
	if _, err := (&Backend{socket: f.socket}).Get(t.Context(), "Iv1.x"); !errors.Is(err, agentapi.ErrAgentLocked) {
		t.Fatalf("Get err = %v, want ErrAgentLocked", err)
	}
}

// TestBackend_setUnsupported guards that the agent backend rejects a token push: the
// agent mints and stores tokens itself.
func TestBackend_setUnsupported(t *testing.T) {
	t.Parallel()
	if err := (&Backend{socket: "unused"}).Set(t.Context(), "Iv1.x", "{}"); err == nil {
		t.Fatal("Set must return an error on the agent backend")
	}
}

// TestBackend_beginAndPoll drives the server-side device flow: Begin returns the
// one-time code and Poll returns the token once the agent reports it minted.
func TestBackend_beginAndPoll(t *testing.T) {
	t.Parallel()
	value := `{"access_token":"minted","expiration_date":"2026-01-01T00:00:00Z"}`
	var polls int
	var mu sync.Mutex
	f := startFakeAgent(t, func(req *agentapi.Request) *agentapi.Response {
		if req.StartDeviceFlow {
			return &agentapi.Response{
				OK:              true,
				Pending:         true,
				UserCode:        "ABCD-1234",
				VerificationURI: "https://github.com/login/device",
				ExpiresIn:       900,
			}
		}
		// Plain polls: report pending twice, then hand back the token.
		mu.Lock()
		polls++
		n := polls
		mu.Unlock()
		if n < 3 {
			return &agentapi.Response{OK: true, Pending: true}
		}
		return &agentapi.Response{OK: true, Token: json.RawMessage(value)}
	})
	// Run under synctest so Poll's real 5s ticker advances instantly: the single
	// bubble goroutine alternates a real socket round trip (the fakeAgent runs
	// outside the bubble and answers at once) with a durable wait on the ticker,
	// which lets the fake clock jump. The number of polls is driven by the handler
	// counter above, not by timing, so the assertions are unchanged.
	synctest.Test(t, func(t *testing.T) {
		b := &Backend{socket: f.socket}
		ctx := t.Context()

		token, dc, err := b.Begin(ctx, "Iv1.x", 0)
		if err != nil {
			t.Fatal(err)
		}
		if token != nil {
			t.Fatalf("Begin must not return a token when it starts a flow, got %q", token)
		}
		want := &pubdeviceflow.DeviceCodeResponse{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device", ExpiresIn: 900}
		if diff := cmp.Diff(want, dc); diff != "" {
			t.Fatalf("device code (-want +got):\n%s", diff)
		}

		got, err := b.Poll(ctx, "Iv1.x", 0)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(value, string(got)); diff != "" {
			t.Fatalf("polled token (-want +got):\n%s", diff)
		}
	})
}

// TestBackend_pollFlowFailed reports an error when the agent's flow ends without a
// token (a plain GET returns not-found while no flow is in progress).
func TestBackend_pollFlowFailed(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespNotFound}
	})
	synctest.Test(t, func(t *testing.T) {
		b := &Backend{socket: f.socket}
		if _, err := b.Poll(t.Context(), "Iv1.x", 0); err == nil {
			t.Fatal("Poll must error when the flow ends without a token")
		}
	})
}

// TestBackend_getActiveSendsMinExpiration guards that GetActive forwards its freshness
// requirement to the agent and stays a pure probe (no device flow started).
func TestBackend_getActiveSendsMinExpiration(t *testing.T) {
	t.Parallel()
	value := `{"access_token":"abc","expiration_date":"2026-01-01T00:00:00Z"}`
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{OK: true, Token: json.RawMessage(value)}
	})
	const minExpiration = 30 * time.Minute
	if _, err := (&Backend{socket: f.socket}).GetActive(t.Context(), "Iv1.x", minExpiration); err != nil {
		t.Fatal(err)
	}
	reqs := f.reqs()
	if len(reqs) != 1 {
		t.Fatalf("want 1 request, got %d: %+v", len(reqs), reqs)
	}
	if reqs[0].Command != agentapi.CommandGet || reqs[0].StartDeviceFlow {
		t.Fatalf("GetActive must send a plain GET probe: %+v", reqs[0])
	}
	if reqs[0].MinExpiration != minExpiration {
		t.Fatalf("MinExpiration = %v, want %v", reqs[0].MinExpiration, minExpiration)
	}
}

// TestBackend_beginReturnsExistingToken covers the concurrency case: another client
// already minted a token, so the agent hands it back on the StartDeviceFlow request
// instead of starting a flow. Begin returns the token and a nil device code.
func TestBackend_beginReturnsExistingToken(t *testing.T) {
	t.Parallel()
	value := `{"access_token":"existing","expiration_date":"2026-01-01T00:00:00Z"}`
	f := startFakeAgent(t, func(req *agentapi.Request) *agentapi.Response {
		if !req.StartDeviceFlow {
			return &agentapi.Response{Error: "unexpected plain GET"}
		}
		return &agentapi.Response{OK: true, Token: json.RawMessage(value)}
	})
	b := &Backend{socket: f.socket}
	token, dc, err := b.Begin(t.Context(), "Iv1.x", 0)
	if err != nil {
		t.Fatal(err)
	}
	if dc != nil {
		t.Fatalf("device code must be nil when a token already exists, got %+v", dc)
	}
	if diff := cmp.Diff(value, string(token)); diff != "" {
		t.Fatalf("token (-want +got):\n%s", diff)
	}
}

// TestBackend_revokeTokens covers the batch revoke command: the request carries all
// client IDs, and the failure lists come back to the caller; a server error and the
// locked agent map to an error / agentapi.ErrAgentLocked.
func TestBackend_revokeTokens(t *testing.T) {
	t.Parallel()
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
			return &agentapi.Response{OK: true, RevokeFailed: []string{"Iv1.b"}, CleanupFailed: []string{"Iv1.c"}}
		})
		revokeFailed, cleanupFailed, err := (&Backend{socket: f.socket}).RevokeTokens(t.Context(), []string{"Iv1.a", "Iv1.b", "Iv1.c"})
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff([]string{"Iv1.b"}, revokeFailed); diff != "" {
			t.Fatalf("revokeFailed (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"Iv1.c"}, cleanupFailed); diff != "" {
			t.Fatalf("cleanupFailed (-want +got):\n%s", diff)
		}
		reqs := f.reqs()
		if len(reqs) != 1 || reqs[0].Command != agentapi.CommandRevoke {
			t.Fatalf("unexpected request: %+v", reqs)
		}
		if diff := cmp.Diff([]string{"Iv1.a", "Iv1.b", "Iv1.c"}, reqs[0].ClientIDs); diff != "" {
			t.Fatalf("request client IDs (-want +got):\n%s", diff)
		}
	})
	t.Run("error", func(t *testing.T) {
		t.Parallel()
		f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
			return &agentapi.Response{Error: "boom"}
		})
		if _, _, err := (&Backend{socket: f.socket}).RevokeTokens(t.Context(), []string{"Iv1.x"}); err == nil {
			t.Fatal("a server error response must produce an error")
		}
	})
	t.Run("locked", func(t *testing.T) {
		t.Parallel()
		f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
			return &agentapi.Response{Error: agentapi.RespLocked}
		})
		if _, _, err := (&Backend{socket: f.socket}).RevokeTokens(t.Context(), []string{"Iv1.x"}); !errors.Is(err, agentapi.ErrAgentLocked) {
			t.Fatalf("RevokeTokens err = %v, want ErrAgentLocked", err)
		}
	})
}

func TestBackend_deleteOK(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response { return &agentapi.Response{OK: true} })
	if err := (&Backend{socket: f.socket}).Delete(t.Context(), "Iv1.x"); err != nil {
		t.Fatal(err)
	}
	reqs := f.reqs()
	if len(reqs) != 1 || reqs[0].Command != agentapi.CommandDelete || reqs[0].ClientID != "Iv1.x" {
		t.Fatalf("unexpected request: %+v", reqs)
	}
}

func TestBackend_deleteMiss(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespNotFound}
	})
	if err := (&Backend{socket: f.socket}).Delete(t.Context(), "Iv1.absent"); err != nil {
		t.Fatalf("Delete() on miss must return nil, got %v", err)
	}
}

func TestBackend_deleteLocked(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespLocked}
	})
	if err := (&Backend{socket: f.socket}).Delete(t.Context(), "Iv1.x"); !errors.Is(err, agentapi.ErrAgentLocked) {
		t.Fatalf("Delete err = %v, want ErrAgentLocked", err)
	}
}

func TestBackend_deleteNotRunning(t *testing.T) {
	t.Parallel()
	socket := filepath.Join(t.TempDir(), "absent.sock")
	if err := (&Backend{socket: socket}).Delete(t.Context(), "Iv1.x"); !agentapi.IsNotRunning(err) {
		t.Fatalf("Delete err = %v, want ErrAgentNotRunning", err)
	}
}

// TestBackend_obsoleteAgent verifies that an agent which predates protocol versioning
// is refused instead of trusted. Such an agent ignores min_expiration and
// start_device_flow, so it would answer a freshness-checked GET with whatever it has
// cached (an expired token reads as valid) and never start the server-side device
// flow. Upgrading ghtkn does not update an already-running agent, so this is what a
// user who forgot to restart it hits.
func TestBackend_obsoleteAgent(t *testing.T) {
	t.Parallel()
	stale := `{"access_token":"expired","expiration_date":"2000-01-01T00:00:00Z"}`
	f := startLegacyFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		// A pre-versioning agent knows GET and answers it with its cached token, no
		// matter how the current client meant the request.
		return &agentapi.Response{OK: true, Token: json.RawMessage(stale)}
	})
	b := &Backend{socket: f.socket}

	if _, err := b.GetActive(t.Context(), "Iv1.x", time.Hour); !errors.Is(err, agentapi.ErrObsoleteAgent) {
		t.Fatalf("GetActive err = %v, want ErrObsoleteAgent", err)
	}
	if _, _, err := b.Begin(t.Context(), "Iv1.x", 0); !errors.Is(err, agentapi.ErrObsoleteAgent) {
		t.Fatalf("Begin err = %v, want ErrObsoleteAgent", err)
	}
	if _, err := b.Poll(t.Context(), "Iv1.x", 0); !errors.Is(err, agentapi.ErrObsoleteAgent) {
		t.Fatalf("Poll err = %v, want ErrObsoleteAgent", err)
	}
	if _, _, err := b.RevokeTokens(t.Context(), []string{"Iv1.x"}); !errors.Is(err, agentapi.ErrObsoleteAgent) {
		t.Fatalf("RevokeTokens err = %v, want ErrObsoleteAgent", err)
	}
}

// TestBackend_obsoleteAgentReported verifies that an agent which does know versioning
// but is older than the client (it answers RespObsoleteAgent) is reported the same way.
func TestBackend_obsoleteAgentReported(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespObsoleteAgent}
	})
	if _, err := (&Backend{socket: f.socket}).GetActive(t.Context(), "Iv1.x", 0); !errors.Is(err, agentapi.ErrObsoleteAgent) {
		t.Fatalf("GetActive err = %v, want ErrObsoleteAgent", err)
	}
}

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// fakeAgent listens on a Unix socket and serves one request per connection using
// handler, which receives the decoded request and returns the response to send.
type fakeAgent struct {
	socket   string
	listener net.Listener
	requests []*agentapi.Request
}

func startFakeAgent(t *testing.T, handler func(*agentapi.Request) *agentapi.Response) *fakeAgent {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "agent.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeAgent{socket: socket, listener: listener}
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
	f.requests = append(f.requests, req)
	b, err := json.Marshal(handler(req))
	if err != nil {
		return
	}
	_, _ = conn.Write(append(b, '\n'))
}

func TestBackend_setGetRoundTrip(t *testing.T) {
	t.Parallel()
	// The agent echoes back whatever was SET, keyed by client ID.
	store := map[string]json.RawMessage{}
	f := startFakeAgent(t, func(req *agentapi.Request) *agentapi.Response {
		switch req.Command {
		case agentapi.CommandSet:
			store[req.ClientID] = req.Token
			return &agentapi.Response{OK: true}
		case agentapi.CommandGet:
			tok, ok := store[req.ClientID]
			if !ok {
				return &agentapi.Response{Error: agentapi.RespNotFound}
			}
			return &agentapi.Response{OK: true, Token: tok}
		default:
			return &agentapi.Response{Error: "unknown command"}
		}
	})
	b := &Backend{socket: f.socket}
	ctx := context.Background()

	value := `{"access_token":"abc","expiration_date":"2026-01-01T00:00:00Z","login":"me"}`
	if err := b.Set(ctx, "Iv1.x", value); err != nil {
		t.Fatal(err)
	}
	got, err := b.Get(ctx, "Iv1.x")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(value, string(got)); diff != "" {
		t.Fatalf("token round-trip (-want +got):\n%s", diff)
	}
}

func TestBackend_getMiss(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespNotFound}
	})
	got, err := (&Backend{socket: f.socket}).Get(context.Background(), "Iv1.absent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("miss must return nil, got %q", got)
	}
}

func TestBackend_serverError(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: "boom"}
	})
	if _, err := (&Backend{socket: f.socket}).Get(context.Background(), "Iv1.x"); err == nil {
		t.Fatal("a server error response must produce an error")
	}
}

func TestBackend_agentNotRunning(t *testing.T) {
	t.Parallel()
	socket := filepath.Join(t.TempDir(), "absent.sock")
	b := &Backend{socket: socket}
	ctx := context.Background()
	if _, err := b.Get(ctx, "Iv1.x"); !agentapi.IsNotRunning(err) {
		t.Fatalf("Get err = %v, want ErrAgentNotRunning", err)
	}
	if err := b.Set(ctx, "Iv1.x", "{}"); !agentapi.IsNotRunning(err) {
		t.Fatalf("Set err = %v, want ErrAgentNotRunning", err)
	}
}

// TestBackend_setRequestShape guards the wire contract: the request the client
// emits must match the agent server's Request fields.
func TestBackend_setRequestShape(t *testing.T) {
	t.Parallel()
	f := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response { return &agentapi.Response{OK: true} })
	value := `{"access_token":"abc"}`
	if err := (&Backend{socket: f.socket}).Set(context.Background(), "Iv1.x", value); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 1 {
		t.Fatalf("got %d requests, want 1", len(f.requests))
	}
	req := f.requests[0]
	if req.Command != agentapi.CommandSet || req.ClientID != "Iv1.x" {
		t.Fatalf("unexpected request: %+v", req)
	}
	if diff := cmp.Diff(value, string(req.Token)); diff != "" {
		t.Fatalf("token sent (-want +got):\n%s", diff)
	}
}

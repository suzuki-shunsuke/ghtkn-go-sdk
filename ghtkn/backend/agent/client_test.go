package agent_test

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

// startFakeAgent listens on a Unix socket and serves one request per connection
// using handler, which receives the decoded request and returns the response to send.
func startFakeAgent(t *testing.T, handler func(*agentapi.Request) *agentapi.Response) string {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "agent.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() }) //nolint:errcheck
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			serveOne(conn, handler)
		}
	}()
	return socket
}

func serveOne(conn net.Conn, handler func(*agentapi.Request) *agentapi.Response) {
	defer conn.Close() //nolint:errcheck
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return
	}
	req := &agentapi.Request{}
	if err := json.Unmarshal(line, req); err != nil {
		return
	}
	b, err := json.Marshal(handler(req))
	if err != nil {
		return
	}
	_, _ = conn.Write(append(b, '\n'))
}

func TestSend_roundTrip(t *testing.T) {
	t.Parallel()
	socket := startFakeAgent(t, func(req *agentapi.Request) *agentapi.Response {
		if req.Command != agentapi.CommandGet || req.ClientID != "Iv1.x" {
			return &agentapi.Response{Error: "unexpected request"}
		}
		return &agentapi.Response{OK: true, Token: json.RawMessage(`{"access_token":"abc"}`)}
	})
	resp, err := agentapi.Send(t.Context(), socket, &agentapi.Request{Command: agentapi.CommandGet, ClientID: "Iv1.x"})
	if err != nil {
		t.Fatal(err)
	}
	want := &agentapi.Response{OK: true, Token: json.RawMessage(`{"access_token":"abc"}`)}
	if diff := cmp.Diff(want, resp); diff != "" {
		t.Fatalf("response (-want +got):\n%s", diff)
	}
}

func TestSend_notFound(t *testing.T) {
	t.Parallel()
	socket := startFakeAgent(t, func(*agentapi.Request) *agentapi.Response {
		return &agentapi.Response{Error: agentapi.RespNotFound}
	})
	resp, err := agentapi.Send(t.Context(), socket, &agentapi.Request{Command: agentapi.CommandGet, ClientID: "Iv1.x"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK || resp.Error != agentapi.RespNotFound {
		t.Fatalf("want not-found response, got %+v", resp)
	}
}

func TestSend_agentNotRunning(t *testing.T) {
	t.Parallel()
	socket := filepath.Join(t.TempDir(), "absent.sock")
	_, err := agentapi.Send(t.Context(), socket, &agentapi.Request{Command: agentapi.CommandStatus})
	if !agentapi.IsNotRunning(err) {
		t.Fatalf("err = %v, want ErrAgentNotRunning", err)
	}
}

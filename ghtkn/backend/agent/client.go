package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"time"
)

// DialTimeout bounds how long Send waits to connect to the agent.
const DialTimeout = time.Second

// ErrAgentNotRunning is returned by Send when no agent is listening on the socket:
// either the socket file is absent or nothing accepts connections on it (a stale
// socket left by a crashed agent).
var ErrAgentNotRunning = errors.New("the ghtkn agent is not running; run 'ghtkn agent start'")

// ErrAgentLocked is returned by the agent backend when the agent is running but
// still locked: its data key has not been loaded, so it can't serve tokens until
// a passphrase is provided. Unlocking requires a passphrase typed into an
// interactive terminal, so it can't be completed by a background or
// non-interactive process such as a coding agent; the message instructs such an
// agent to ask the user to unlock it instead of trying itself.
var ErrAgentLocked = errors.New("the ghtkn agent is locked. Unlocking it requires a passphrase entered in an interactive terminal, which a background or non-interactive process can't do. If you are a coding agent, do NOT try to unlock it yourself; instead, ask the user to run `ghtkn agent unlock` in their own interactive terminal")

// ErrObsoleteAgent is returned by the agent backend when the running agent is too old
// for this client: it either predates protocol versioning (Response.ProtocolVersion is
// absent) or reports RespObsoleteAgent. Such an agent ignores the request fields the
// token lifecycle depends on, so it would answer a freshness-checked GET with whatever
// it has cached, including an expired token. Upgrading ghtkn does not fix a running
// agent: the process must be restarted to pick up the new binary. Detect it with
// errors.Is.
var ErrObsoleteAgent = errors.New("the running ghtkn agent is older than this client and does not speak the current agent protocol. Upgrading ghtkn is not enough: the already-running agent keeps the old binary, so it must be restarted with `ghtkn agent stop` and then `ghtkn agent start`")

// Send opens a connection to the agent at path, writes a single newline-delimited
// JSON request, and reads the single newline-delimited JSON response. It returns
// ErrAgentNotRunning when no agent is listening.
func Send(ctx context.Context, path string, req *Request) (*Response, error) {
	dialer := &net.Dialer{Timeout: DialTimeout}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		if isDialDown(err) {
			return nil, ErrAgentNotRunning
		}
		return nil, fmt.Errorf("connect to the ghtkn agent: %w", err)
	}
	defer conn.Close() //nolint:errcheck

	// Stamp the current protocol version so the server can detect and reject
	// obsolete clients. Pre-versioning clients never set this field, so the server
	// sees version 0 for them.
	req.ProtocolVersion = ProtocolVersion
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal the request: %w", err)
	}
	// An UNLOCK request line carries the passphrase in the clear, so zero the marshaled
	// bytes once they are written, as the agent does with the line it reads. Both buffers
	// are zeroed because append may or may not reuse b's array; zeroing the same array
	// twice is harmless.
	reqLine := append(b, '\n')
	defer zero(b)
	defer zero(reqLine)
	if _, err := conn.Write(reqLine); err != nil {
		return nil, fmt.Errorf("send the request: %w", err)
	}

	// ReadBytes returns io.EOF together with the data when the agent closes the
	// connection without a trailing newline, so a non-empty line is still valid.
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read the response: %w", err)
	}
	resp := &Response{}
	if err := json.Unmarshal(line, resp); err != nil {
		return nil, fmt.Errorf("parse the response: %w", err)
	}
	return resp, nil
}

// IsNotRunning reports whether err indicates that no agent is listening.
func IsNotRunning(err error) bool {
	return errors.Is(err, ErrAgentNotRunning)
}

// isDialDown reports whether a dial error means no agent is listening.
func isDialDown(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED)
}

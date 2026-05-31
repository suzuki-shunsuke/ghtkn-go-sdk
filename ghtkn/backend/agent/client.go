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

	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal the request: %w", err)
	}
	if _, err := conn.Write(append(b, '\n')); err != nil {
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

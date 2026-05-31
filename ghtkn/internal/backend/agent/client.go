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

// dialTimeout bounds how long a single request waits to connect to the agent.
const dialTimeout = time.Second

// errAgentNotRunning is returned when no agent is listening on the socket.
var errAgentNotRunning = errors.New("the ghtkn agent is not running; run 'ghtkn agent start'")

// roundTrip sends a single newline-delimited JSON request to the agent at socket
// and reads the single newline-delimited JSON response. It returns errAgentNotRunning
// when no agent is listening.
func roundTrip(ctx context.Context, socket string, req *request) (*response, error) {
	dialer := &net.Dialer{Timeout: dialTimeout}
	conn, err := dialer.DialContext(ctx, "unix", socket)
	if err != nil {
		if isAgentDown(err) {
			return nil, errAgentNotRunning
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
	resp := &response{}
	if err := json.Unmarshal(line, resp); err != nil {
		return nil, fmt.Errorf("parse the response: %w", err)
	}
	return resp, nil
}

// isAgentDown reports whether a dial error means no agent is listening: either the
// socket file is absent or nothing accepts connections on it (a stale socket).
func isAgentDown(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED)
}

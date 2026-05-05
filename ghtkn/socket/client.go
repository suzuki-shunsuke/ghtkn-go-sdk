package socket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

// httpScheme is a placeholder host used when building URLs for the Unix socket
// transport. The transport ignores the host, but net/http requires a valid URL.
const unixHTTPBase = "http://unix"

// NewClient returns an *http.Client whose Transport dials the Unix socket at
// sockPath. The returned client should be used with URLs that have any host
// (the host is ignored); helpers in this package construct such URLs as
// "http://unix" + path.
func NewClient(sockPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}
}

// Error is returned by socket helpers when the daemon responds with a non-2xx
// status. It carries both the HTTP status and the structured ErrorResponse
// fields when the body could be decoded.
type Error struct {
	Status  int
	Code    string
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("ghtkn socket: %d %s: %s", e.Status, e.Code, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("ghtkn socket: %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("ghtkn socket: %d", e.Status)
}

// FetchToken posts a TokenRequest to the daemon at sockPath, presenting
// capToken as a Bearer credential, and returns the decoded TokenResponse. A
// non-2xx response is reported as an *Error.
func FetchToken(ctx context.Context, sockPath, capToken string, req *TokenRequest) (*TokenResponse, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal token request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, unixHTTPBase+PathToken, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if capToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+capToken)
	}
	resp, err := NewClient(sockPath).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send token request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, decodeError(resp.StatusCode, respBody)
	}
	tokenResp := &TokenResponse{}
	if err := json.Unmarshal(respBody, tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return tokenResp, nil
}

// decodeError builds an *Error from a non-2xx response body. If the body is
// not a JSON ErrorResponse, the raw body is used as the message.
func decodeError(status int, body []byte) error {
	errResp := &ErrorResponse{}
	if err := json.Unmarshal(body, errResp); err == nil && (errResp.Code != "" || errResp.Message != "") {
		return &Error{Status: status, Code: errResp.Code, Message: errResp.Message}
	}
	return &Error{Status: status, Message: string(bytes.TrimSpace(body))}
}

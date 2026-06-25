// Package revoke calls GitHub's credential revocation API to revoke leaked
// GitHub credentials (e.g. GitHub App User Access Tokens or Personal Access
// Tokens). The endpoint is unauthenticated -- authenticated requests are
// rejected with 403 -- and accepts up to 1000 credentials per request.
//
// https://docs.github.com/en/rest/credentials/revoke
package revoke

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

const (
	// revokeURL is GitHub's unauthenticated credential revocation endpoint.
	revokeURL = "https://api.github.com/credentials/revoke"
	// apiVersion is the X-GitHub-Api-Version sent with the request.
	apiVersion = "2026-03-10"
	// maxCredentialsPerRequest is the maximum number of credentials GitHub
	// accepts in a single revocation request.
	maxCredentialsPerRequest = 1000
)

// httpClient is the subset of *http.Client used by the revoke client. It is an
// interface so tests can inject a fake.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client revokes GitHub credentials through the credential revocation API.
type Client struct {
	httpClient httpClient
}

// New creates a revoke client. When httpClient is nil, http.DefaultClient is used.
func New(c httpClient) *Client {
	if c == nil {
		c = http.DefaultClient
	}
	return &Client{httpClient: c}
}

// Revoke revokes the given credentials. It splits them into batches of at most
// maxCredentialsPerRequest. It is a no-op when tokens is empty.
func (c *Client) Revoke(ctx context.Context, tokens []string) error {
	for start := 0; start < len(tokens); start += maxCredentialsPerRequest {
		end := min(start+maxCredentialsPerRequest, len(tokens))
		if err := c.revoke(ctx, tokens[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// revoke sends a single revocation request for up to maxCredentialsPerRequest tokens.
func (c *Client) revoke(ctx context.Context, tokens []string) error {
	body, err := json.Marshal(map[string][]string{"credentials": tokens})
	if err != nil {
		return fmt.Errorf("marshal a request body as JSON: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revokeURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create a request to revoke credentials: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send a request to revoke credentials: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// A successful revocation returns 202 Accepted.
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		return slogerr.With( //nolint:wrapcheck
			errors.New("failed to revoke credentials"),
			"status_code", resp.StatusCode,
			"body", string(b))
	}
	return nil
}

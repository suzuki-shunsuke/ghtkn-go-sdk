package revoke

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// roundTripFunc lets a function act as an http.RoundTripper.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_Revoke(t *testing.T) {
	t.Parallel()

	t.Run("posts the credentials and succeeds on 202", func(t *testing.T) {
		t.Parallel()

		var gotBody map[string][]string
		var gotMethod, gotURL, gotVersion, gotAccept string
		c := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotURL = req.URL.String()
			gotVersion = req.Header.Get("X-GitHub-Api-Version")
			gotAccept = req.Header.Get("Accept")
			b, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(b, &gotBody)
			return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(""))}, nil
		})})

		if err := c.Revoke(t.Context(), []string{"ghu_a", "ghu_b"}); err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}
		if gotMethod != http.MethodPost {
			t.Errorf("method = %q, want POST", gotMethod)
		}
		if gotURL != revokeURL {
			t.Errorf("url = %q, want %q", gotURL, revokeURL)
		}
		if gotVersion != apiVersion {
			t.Errorf("api version = %q, want %q", gotVersion, apiVersion)
		}
		if gotAccept != "application/vnd.github+json" {
			t.Errorf("accept = %q", gotAccept)
		}
		if diff := cmp.Diff(map[string][]string{"credentials": {"ghu_a", "ghu_b"}}, gotBody); diff != "" {
			t.Errorf("body mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("non-202 status is an error", func(t *testing.T) {
		t.Parallel()

		c := New(&http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader("forbidden"))}, nil
		})})
		if err := c.Revoke(t.Context(), []string{"ghu_a"}); err == nil {
			t.Error("Revoke() expected an error, got nil")
		}
	})

	t.Run("empty tokens is a no-op without any request", func(t *testing.T) {
		t.Parallel()

		called := false
		c := New(&http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			called = true
			return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(""))}, nil
		})})
		if err := c.Revoke(t.Context(), nil); err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}
		if called {
			t.Error("Revoke() sent a request for an empty token list")
		}
	})

	t.Run("splits into batches of at most maxCredentialsPerRequest", func(t *testing.T) {
		t.Parallel()

		var batchSizes []int
		c := New(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			var body map[string][]string
			b, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(b, &body)
			batchSizes = append(batchSizes, len(body["credentials"]))
			return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(""))}, nil
		})})

		tokens := make([]string, maxCredentialsPerRequest+1)
		for i := range tokens {
			tokens[i] = "t"
		}
		if err := c.Revoke(t.Context(), tokens); err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}
		if diff := cmp.Diff([]int{maxCredentialsPerRequest, 1}, batchSizes); diff != "" {
			t.Errorf("batch sizes mismatch (-want +got):\n%s", diff)
		}
	})
}

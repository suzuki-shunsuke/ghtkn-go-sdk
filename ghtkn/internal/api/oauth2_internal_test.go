package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	pubapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"golang.org/x/oauth2"
)

type mockTokenSourceClient struct {
	token *pubapi.AccessToken
	err   error
	calls int
}

func (m *mockTokenSourceClient) Get(_ context.Context, _ *slog.Logger, _ *pubapi.InputGet) (*pubapi.AccessToken, *pubconfig.App, error) {
	m.calls++
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.token, nil, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestTokenManager_TokenSource(t *testing.T) {
	t.Parallel()

	tm := &TokenManager{}
	logger := newTestLogger()
	input := &pubapi.InputGet{ConfigFilePath: "/path/to/config.yaml"}

	ts := tm.TokenSource(logger, input)
	if ts == nil {
		t.Fatal("TokenSource() returned nil")
	}
	if ts.mutex == nil {
		t.Error("mutex is nil")
	}
	if ts.tm == nil {
		t.Error("tm is nil")
	}
	if ts.logger != logger {
		t.Error("logger was not propagated")
	}
	if ts.input != input {
		t.Error("input was not propagated")
	}
}

func TestTokenSource_Token(t *testing.T) {
	t.Parallel()

	// Token() reads time.Now() to decide whether a cached token is still valid, so the
	// expiry-sensitive sub-tests run in a synctest bubble where time.Now() is fixed and
	// future/past are expressed relative to it.

	t.Run("cached non-expired token is returned without calling client", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			cached := &oauth2.Token{AccessToken: "cached", Expiry: time.Now().Add(time.Hour)}
			client := &mockTokenSourceClient{}
			ts := &TokenSource{
				token:  cached,
				mutex:  &sync.Mutex{},
				tm:     client,
				logger: newTestLogger(),
			}

			got, err := ts.Token()
			if err != nil {
				t.Fatal(err)
			}
			if got != cached {
				t.Errorf("expected cached token, got %v", got)
			}
			if client.calls != 0 {
				t.Errorf("client.calls = %d, want 0", client.calls)
			}
		})
	})

	t.Run("no cached token fetches from client", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			future := time.Now().Add(time.Hour)
			client := &mockTokenSourceClient{
				token: &pubapi.AccessToken{AccessToken: "new", ExpirationDate: future},
			}
			ts := &TokenSource{
				mutex:  &sync.Mutex{},
				tm:     client,
				logger: newTestLogger(),
			}

			got, err := ts.Token()
			if err != nil {
				t.Fatal(err)
			}
			if got.AccessToken != "new" || !got.Expiry.Equal(future) {
				t.Errorf("got AccessToken=%q Expiry=%v, want AccessToken=%q Expiry=%v", got.AccessToken, got.Expiry, "new", future)
			}
			if client.calls != 1 {
				t.Errorf("client.calls = %d, want 1", client.calls)
			}
			if ts.token == nil || ts.token.AccessToken != "new" {
				t.Error("token was not cached")
			}
		})
	})

	t.Run("cached expired token triggers refetch", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			future := time.Now().Add(time.Hour)
			expired := &oauth2.Token{AccessToken: "old", Expiry: time.Now().Add(-time.Hour)}
			client := &mockTokenSourceClient{
				token: &pubapi.AccessToken{AccessToken: "new", ExpirationDate: future},
			}
			ts := &TokenSource{
				token:  expired,
				mutex:  &sync.Mutex{},
				tm:     client,
				logger: newTestLogger(),
			}

			got, err := ts.Token()
			if err != nil {
				t.Fatal(err)
			}
			if got.AccessToken != "new" || !got.Expiry.Equal(future) {
				t.Errorf("got AccessToken=%q Expiry=%v, want AccessToken=%q Expiry=%v", got.AccessToken, got.Expiry, "new", future)
			}
			if client.calls != 1 {
				t.Errorf("client.calls = %d, want 1", client.calls)
			}
		})
	})

	t.Run("client error is returned and token stays nil", func(t *testing.T) {
		t.Parallel()

		client := &mockTokenSourceClient{err: errors.New("boom")}
		ts := &TokenSource{
			mutex:  &sync.Mutex{},
			tm:     client,
			logger: newTestLogger(),
		}

		got, err := ts.Token()
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		if got != nil {
			t.Errorf("expected nil token, got %v", got)
		}
		if ts.token != nil {
			t.Error("token should remain nil on error")
		}
	})

	t.Run("fetched token is cached across calls", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			client := &mockTokenSourceClient{
				token: &pubapi.AccessToken{AccessToken: "new", ExpirationDate: time.Now().Add(time.Hour)},
			}
			ts := &TokenSource{
				mutex:  &sync.Mutex{},
				tm:     client,
				logger: newTestLogger(),
			}

			first, err := ts.Token()
			if err != nil {
				t.Fatal(err)
			}
			second, err := ts.Token()
			if err != nil {
				t.Fatal(err)
			}
			if first != second {
				t.Error("expected the same cached token instance on the second call")
			}
			if client.calls != 1 {
				t.Errorf("client.calls = %d, want 1", client.calls)
			}
		})
	})
}

func TestIsExpired(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		token *oauth2.Token
		want  bool
	}{
		{
			name:  "zero expiry never expires",
			token: &oauth2.Token{AccessToken: "x"},
			want:  false,
		},
		{
			name:  "expiry before now is expired",
			token: &oauth2.Token{AccessToken: "x", Expiry: now.Add(-time.Hour)},
			want:  true,
		},
		{
			name:  "expiry after now is not expired",
			token: &oauth2.Token{AccessToken: "x", Expiry: now.Add(time.Hour)},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isExpired(tt.token, now); got != tt.want {
				t.Errorf("isExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

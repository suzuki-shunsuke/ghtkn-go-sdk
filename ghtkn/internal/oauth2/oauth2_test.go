package oauth2_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/oauth2"
	oauth2lib "golang.org/x/oauth2"
)

// mockClient implements the oauth2.Client interface for testing
type mockClient struct {
	token string
	err   error
	calls int
	mutex sync.Mutex
}

func (m *mockClient) Get() (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calls++
	return m.token, m.err
}

func (m *mockClient) getCalls() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.calls
}

func TestNewTokenSource(t *testing.T) {
	t.Parallel()

	client := &mockClient{token: "test-token"}
	ts := oauth2.NewTokenSource(client)

	if ts == nil {
		t.Error("NewTokenSource() returned nil")
	}
}

func TestTokenSource_Token(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		clientToken   string
		clientError   error
		expectedToken string
		wantErr       bool
	}{
		{
			name:          "successful token retrieval",
			clientToken:   "ghp_test_token_123",
			clientError:   nil,
			expectedToken: "ghp_test_token_123",
			wantErr:       false,
		},
		{
			name:        "client returns error",
			clientToken: "",
			clientError: errors.New("failed to get token"),
			wantErr:     true,
		},
		{
			name:          "empty token from client",
			clientToken:   "",
			clientError:   nil,
			expectedToken: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &mockClient{
				token: tt.clientToken,
				err:   tt.clientError,
			}
			ts := oauth2.NewTokenSource(client)

			token, err := ts.Token()
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenSource.Token() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if token == nil {
					t.Error("TokenSource.Token() returned nil token")
					return
				}
				if token.AccessToken != tt.expectedToken {
					t.Errorf("TokenSource.Token().AccessToken = %v, want %v", token.AccessToken, tt.expectedToken)
				}
			}
		})
	}
}

func TestTokenSource_Token_Caching(t *testing.T) {
	t.Parallel()

	client := &mockClient{
		token: "cached-token",
		err:   nil,
	}
	ts := oauth2.NewTokenSource(client)

	// First call should retrieve token from client
	token1, err := ts.Token()
	if err != nil {
		t.Errorf("First TokenSource.Token() error = %v", err)
		return
	}
	if client.getCalls() != 1 {
		t.Errorf("Expected 1 client call, got %d", client.getCalls())
	}

	// Second call should return cached token without calling client
	token2, err := ts.Token()
	if err != nil {
		t.Errorf("Second TokenSource.Token() error = %v", err)
		return
	}
	if client.getCalls() != 1 {
		t.Errorf("Expected 1 client call after caching, got %d", client.getCalls())
	}

	// Tokens should be the same instance (cached)
	if token1 != token2 {
		t.Error("Expected cached token to be the same instance")
	}
	if token1.AccessToken != "cached-token" {
		t.Errorf("Token.AccessToken = %v, want %v", token1.AccessToken, "cached-token")
	}
}

func TestTokenSource_Token_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	client := &mockClient{
		token: "concurrent-token",
		err:   nil,
	}
	ts := oauth2.NewTokenSource(client)

	const numGoroutines = 10
	results := make(chan *oauth2lib.Token, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch multiple goroutines to test concurrent access
	for range numGoroutines {
		go func() {
			token, err := ts.Token()
			if err != nil {
				errors <- err
			} else {
				results <- token
			}
		}()
	}

	// Collect results
	var tokens []*oauth2lib.Token
	for range numGoroutines {
		select {
		case token := <-results:
			tokens = append(tokens, token)
		case err := <-errors:
			t.Errorf("Concurrent TokenSource.Token() error = %v", err)
		}
	}

	if len(tokens) != numGoroutines {
		t.Errorf("Expected %d tokens, got %d", numGoroutines, len(tokens))
	}

	// All tokens should be the same instance (cached)
	firstToken := tokens[0]
	for i, token := range tokens {
		if token != firstToken {
			t.Errorf("Token %d is not the same instance as first token", i)
		}
		if token.AccessToken != "concurrent-token" {
			t.Errorf("Token %d AccessToken = %v, want %v", i, token.AccessToken, "concurrent-token")
		}
	}

	// Client should only be called once despite concurrent access
	if client.getCalls() != 1 {
		t.Errorf("Expected 1 client call with concurrent access, got %d", client.getCalls())
	}
}

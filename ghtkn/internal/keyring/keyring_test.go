//nolint:cyclop,funlen
package keyring_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
)

// Mock is a mock implementation of the API interface for testing.
// It stores secrets in memory instead of using the system keyring.
type mockBackend struct {
	secrets map[string]string
}

// newMockBackend creates a new mock API instance with the provided initial secrets.
// If secrets is nil, an empty map will be created when needed.
func newMockBackend(secrets map[string]string) keyring.API {
	return &mockBackend{
		secrets: secrets,
	}
}

// mockKey generates a unique key for storing secrets by combining service and user.
// The format is "service:user".
func mockKey(service, user string) string {
	return service + ":" + user
}

// Get retrieves a secret from the mock keyring.
// Returns keyring.ErrNotFound if the secret doesn't exist.
func (m *mockBackend) Get(service, user string) (string, bool, error) {
	k := mockKey(service, user)
	s, ok := m.secrets[k]
	if !ok {
		return "", false, nil
	}
	return s, true, nil
}

// Set stores a secret in the mock keyring.
// Creates the internal map if it doesn't exist.
func (m *mockBackend) Set(service, user, password string) error {
	if m.secrets == nil {
		m.secrets = map[string]string{}
	}
	m.secrets[mockKey(service, user)] = password
	return nil
}

// TestParseDate tests the ParseDate function with various inputs.
func TestParseDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "valid RFC3339 date",
			input: "2024-01-15T10:30:00Z",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "valid RFC3339 date with timezone",
			input: "2024-06-20T15:45:30+09:00",
			want:  time.Date(2024, 6, 20, 15, 45, 30, 0, time.FixedZone("", 9*60*60)),
		},
		{
			name:  "valid RFC3339 date with negative timezone",
			input: "2024-12-31T23:59:59-05:00",
			want:  time.Date(2024, 12, 31, 23, 59, 59, 0, time.FixedZone("", -5*60*60)),
		},
		{
			name:  "valid RFC3339 date with nanoseconds",
			input: "2024-03-10T08:15:30.123456789Z",
			want:  time.Date(2024, 3, 10, 8, 15, 30, 123456789, time.UTC),
		},
		{
			name:    "invalid format - not RFC3339",
			input:   "2024-01-15 10:30:00",
			wantErr: true,
		},
		{
			name:    "invalid format - missing time",
			input:   "2024-01-15",
			wantErr: true,
		},
		{
			name:    "invalid format - missing timezone",
			input:   "2024-01-15T10:30:00",
			wantErr: true,
		},
		{
			name:    "invalid date string",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "2024-13-01T10:30:00Z",
			wantErr: true,
		},
		{
			name:    "invalid day",
			input:   "2024-02-30T10:30:00Z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := keyring.ParseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatDate tests the FormatDate function.
func TestFormatDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Time
		want  string
	}{
		{
			name:  "UTC time",
			input: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			want:  "2024-01-15T10:30:00Z",
		},
		{
			name:  "time with positive timezone",
			input: time.Date(2024, 6, 20, 15, 45, 30, 0, time.FixedZone("", 9*60*60)),
			want:  "2024-06-20T15:45:30+09:00",
		},
		{
			name:  "time with negative timezone",
			input: time.Date(2024, 12, 31, 23, 59, 59, 0, time.FixedZone("", -5*60*60)),
			want:  "2024-12-31T23:59:59-05:00",
		},
		{
			name:  "time with nanoseconds (truncated by RFC3339)",
			input: time.Date(2024, 3, 10, 8, 15, 30, 123456789, time.UTC),
			want:  "2024-03-10T08:15:30Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := keyring.FormatDate(tt.input)
			if got != tt.want {
				t.Errorf("FormatDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeyring_Get tests the Get method of Keyring.
func TestKeyring_Get(t *testing.T) {
	t.Parallel()

	expirationTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	testToken := &keyring.AccessToken{
		AccessToken:    "ghp_test_token_123",
		ExpirationDate: expirationTime,
		Login:          "testuser",
	}
	tokenJSON, _ := json.Marshal(testToken)

	tests := []struct {
		name    string
		service string
		key     string
		secrets map[string]string
		want    *keyring.AccessToken
		wantErr bool
	}{
		{
			name:    "token found",
			service: "github.com",
			key:     "testuser",
			secrets: map[string]string{
				"github.com:testuser": string(tokenJSON),
			},
			want: testToken,
		},
		{
			name:    "token not found",
			service: "github.com",
			key:     "nonexistent",
			secrets: map[string]string{
				"github.com:testuser": string(tokenJSON),
			},
			want: nil,
		},
		{
			name:    "invalid JSON",
			service: "github.com",
			key:     "invalid",
			secrets: map[string]string{
				"github.com:invalid": "invalid json",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty secrets",
			service: "github.com",
			key:     "testuser",
			secrets: nil,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &keyring.Input{
				API: newMockBackend(tt.secrets),
			}
			kr := keyring.New(input)

			got, err := kr.Get(tt.service, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Keyring.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				if got != nil {
					t.Errorf("Keyring.Get() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("Keyring.Get() = nil, want %v", tt.want)
				return
			}

			if got.AccessToken != tt.want.AccessToken {
				t.Errorf("Keyring.Get() AccessToken = %v, want %v", got.AccessToken, tt.want.AccessToken)
			}
			if !got.ExpirationDate.Equal(tt.want.ExpirationDate) {
				t.Errorf("Keyring.Get() ExpirationDate = %v, want %v", got.ExpirationDate, tt.want.ExpirationDate)
			}
			if got.Login != tt.want.Login {
				t.Errorf("Keyring.Get() Login = %v, want %v", got.Login, tt.want.Login)
			}
		})
	}
}

// TestKeyring_Set tests the Set method of Keyring.
func TestKeyring_Set(t *testing.T) {
	t.Parallel()

	expirationTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	testToken := &keyring.AccessToken{
		AccessToken:    "ghp_test_token_123",
		ExpirationDate: expirationTime,
		Login:          "testuser",
	}

	tests := []struct {
		name    string
		service string
		key     string
		token   *keyring.AccessToken
		wantErr bool
	}{
		{
			name:    "valid token",
			service: "github.com",
			key:     "testuser",
			token:   testToken,
		},
		{
			name:    "empty service",
			service: "",
			key:     "testuser",
			token:   testToken,
		},
		{
			name:    "empty key",
			service: "github.com",
			key:     "",
			token:   testToken,
		},
		{
			name:    "nil token",
			service: "github.com",
			key:     "testuser",
			token:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &keyring.Input{
				API: newMockBackend(nil),
			}
			kr := keyring.New(input)

			err := kr.Set(tt.service, tt.key, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("Keyring.Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.token != nil {
				// Verify the token was stored correctly by retrieving it
				got, err := kr.Get(tt.service, tt.key)
				if err != nil {
					t.Errorf("Keyring.Get() after Set() error = %v", err)
					return
				}

				if got == nil {
					t.Error("Keyring.Get() after Set() returned nil")
					return
				}

				if got.AccessToken != tt.token.AccessToken {
					t.Errorf("Keyring.Set() then Get() AccessToken = %v, want %v", got.AccessToken, tt.token.AccessToken)
				}
				if !got.ExpirationDate.Equal(tt.token.ExpirationDate) {
					t.Errorf("Keyring.Set() then Get() ExpirationDate = %v, want %v", got.ExpirationDate, tt.token.ExpirationDate)
				}
				if got.Login != tt.token.Login {
					t.Errorf("Keyring.Set() then Get() Login = %v, want %v", got.Login, tt.token.Login)
				}
			}
		})
	}
}

// TestNew tests the New function.
func TestNew(t *testing.T) {
	t.Parallel()

	input := &keyring.Input{
		API: newMockBackend(nil),
	}

	kr := keyring.New(input)
	if kr == nil {
		t.Error("New() returned nil")
	}
}

// TestNewInput tests the NewInput function.
func TestNewInput(t *testing.T) {
	t.Parallel()

	input := keyring.NewInput()
	if input == nil {
		t.Error("NewInput() returned nil")
		return
	}

	if input.API == nil {
		t.Error("NewInput().API is nil")
	}
}

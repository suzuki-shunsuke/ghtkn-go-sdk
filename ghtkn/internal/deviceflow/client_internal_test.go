package deviceflow

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
)

func TestClient_Create(t *testing.T) { //nolint:gocognit,cyclop,funlen
	t.Parallel()
	tests := []struct {
		name        string
		clientID    string
		handler     http.HandlerFunc
		want        *AccessToken
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty client ID",
			clientID:    "",
			handler:     nil,
			want:        nil,
			wantErr:     true,
			errContains: "client id is required",
		},
		{
			name:     "successful flow",
			clientID: "test-client-id",
			handler: func() http.HandlerFunc {
				callCount := 0
				return func(w http.ResponseWriter, r *http.Request) {
					callCount++
					switch r.URL.Path {
					case "/login/device/code":
						// Device code request
						resp := deviceflow.DeviceCodeResponse{
							DeviceCode:      "device123",
							UserCode:        "USER-CODE",
							VerificationURI: "https://github.com/login/device",
							ExpiresIn:       10,
							Interval:        1,
						}
						json.NewEncoder(w).Encode(resp) //nolint:errcheck
					case "/login/oauth/access_token":
						// Access token request
						if callCount <= 2 {
							// First call returns pending
							resp := deviceflow.AccessToken{
								Error: "authorization_pending",
							}
							json.NewEncoder(w).Encode(resp) //nolint:errcheck
							return
						}
						// Second call returns success
						resp := deviceflow.AccessToken{
							AccessToken: "gho_testtoken123",
							ExpiresIn:   28800,
						}
						json.NewEncoder(w).Encode(resp) //nolint:errcheck
					}
				}
			}(),
			want: &AccessToken{
				AccessToken: "gho_testtoken123",
				// ExpirationDate will be set dynamically
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var server *httptest.Server
			if tt.handler != nil {
				server = httptest.NewServer(tt.handler)
				defer server.Close()
			}

			fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			input := newMockInput()
			input.Now = func() time.Time { return fixedTime }
			input.Stderr = &bytes.Buffer{}
			if server != nil {
				input.Client = newTestDeviceFlow(server, input.Now)
			}
			client := NewClient(input)

			logger := slog.New(slog.DiscardHandler)

			got, err := client.Create(t.Context(), logger, &InputCreate{ClientID: tt.clientID})
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if tt.wantErr {
				t.Fatalf("expected error but got nil")
				return
			}

			// Compare without ExpirationDate first
			if got.AccessToken != tt.want.AccessToken {
				t.Errorf("AccessToken = %v, want %v", got.AccessToken, tt.want.AccessToken)
			}

			// Check that ExpirationDate is set
			if got.ExpirationDate.IsZero() {
				t.Error("ExpirationDate should be set")
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	input := newMockInput()
	client := NewClient(input)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.input.Now == nil {
		t.Error("now function not set")
	}

	if client.input.Stderr == nil {
		t.Error("stderr not set")
	}
}

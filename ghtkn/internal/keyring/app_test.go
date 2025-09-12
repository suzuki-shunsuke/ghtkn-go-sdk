package keyring_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
)

func TestKeyring_GetApp(t *testing.T) {
	t.Parallel()

	service := "test-service"
	appID := 123

	tests := []struct {
		name    string
		secrets map[string]string
		wantApp *keyring.App
		wantErr bool
	}{
		{
			name: "app found",
			secrets: map[string]string{
				"test-service:apps/123": `{"client_id":"test-client-id"}`,
			},
			wantApp: &keyring.App{
				ClientID: "test-client-id",
			},
		},
		{
			name:    "app not found",
			secrets: map[string]string{},
			wantApp: nil,
		},
		{
			name: "invalid JSON",
			secrets: map[string]string{
				"test-service:apps/123": "invalid json",
			},
			wantErr: true,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := keyring.New(&keyring.Input{
				API: newMockBackend(tt.secrets),
			})

			got, err := store.GetApp(logger, service, appID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantApp == nil && got != nil {
				t.Errorf("Get() = %v, want nil", got)
				return
			}

			if tt.wantApp != nil {
				if got == nil {
					t.Errorf("Get() = nil, want %v", tt.wantApp)
					return
				}

				if got.ClientID != tt.wantApp.ClientID {
					t.Errorf("Get() ClientID = %v, want %v", got.ClientID, tt.wantApp.ClientID)
				}
			}
		})
	}
}

// TestAppStore_Set tests the Set method of AppStore.
func TestKeyring_SetApp(t *testing.T) {
	t.Parallel()

	service := "test-service"
	appID := 456
	app := &keyring.App{
		ClientID: "test-client-id-456",
	}

	store := keyring.New(&keyring.Input{
		API: newMockBackend(nil),
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	err := store.SetApp(logger, service, appID, app)
	if err != nil {
		t.Errorf("Set() error = %v", err)
		return
	}

	// Verify the app was stored correctly
	got, err := store.GetApp(logger, service, appID)
	if err != nil {
		t.Errorf("Get() after Set() error = %v", err)
		return
	}

	if got == nil {
		t.Error("Get() after Set() returned nil")
		return
	}

	if got.ClientID != app.ClientID {
		t.Errorf("Get() after Set() ClientID = %v, want %v", got.ClientID, app.ClientID)
	}
}

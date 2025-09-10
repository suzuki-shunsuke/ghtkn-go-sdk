package keyring

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// AppStore manages access tokens in the system keychain.
// It provides methods to get, set, and remove tokens securely.
type AppStore struct {
	input *Input
}

// NewAppStore creates a new AppStore instance with the specified service name.
// The keyService parameter is used as the service identifier in the system keychain.
func NewAppStore(input *Input) *AppStore {
	return &AppStore{
		input: input,
	}
}

type App struct {
	ClientID string `json:"client_id"`
}

// apps/<app id> => {"client_id": "..."}

func keyApp(appID int) string {
	return fmt.Sprintf("apps/%d", appID)
}

// Get retrieves an access token from the keyring.
// The key parameter identifies the token to retrieve.
// Returns the token or an error if the token cannot be found or unmarshaled.
func (as *AppStore) Get(service string, appID int) (*App, error) {
	s, err := as.input.API.Get(service, keyApp(appID))
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get an App in keyring: %w", err)
	}
	app := &App{}
	if err := json.Unmarshal([]byte(s), app); err != nil {
		return nil, fmt.Errorf("unmarshal the app as JSON: %w", err)
	}
	return app, nil
}

// Set stores an access token in the keyring.
// The key parameter identifies where to store the token.
// Returns an error if the token cannot be marshaled or stored.
func (as *AppStore) Set(service string, appID int, app *App) error {
	s, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("marshal the app as JSON: %w", err)
	}
	if err := as.input.API.Set(service, keyApp(appID), string(s)); err != nil {
		return fmt.Errorf("set an App in keyring: %w", err)
	}
	return nil
}

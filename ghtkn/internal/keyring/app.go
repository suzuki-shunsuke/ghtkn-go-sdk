package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type App struct {
	ClientID string `json:"client_id"`
}

func (a *App) Validate() error {
	if a.ClientID == "" {
		return errors.New("client id is required")
	}
	return nil
}

// apps/<app id> => {"client_id": "..."}

func keyApp(appID int) string {
	return fmt.Sprintf("apps/%d", appID)
}

// Get retrieves an App config from the keyring.
// The key parameter identifies the app to retrieve.
// Returns the app or an error if the app cannot be found or unmarshaled.
func (kr *Keyring) GetApp(logger *slog.Logger, service string, appID int) (*App, error) {
	key := keyApp(appID)
	s, exist, err := kr.input.API.Get(service, key)
	if err != nil {
		return nil, fmt.Errorf("get an App in keyring: %w", err)
	}
	if !exist {
		return nil, nil
	}
	app := &App{}
	if err := json.Unmarshal([]byte(s), app); err != nil {
		// TODO customize logger
		slogerr.WithError(logger, err).With("app_id", appID).Debug("unmarshal the app as JSON")
		// Delete the invalid app
		if _, err := kr.input.API.Delete(service, key); err != nil {
			// TODO customize logger
			slogerr.WithError(logger, err).With("app_id", appID).Debug("delete an invalid App from keyring")
		}
		return nil, nil
	}
	// Validate and delete the invalid app
	if err := app.Validate(); err != nil {
		// TODO customize logger
		slogerr.WithError(logger, err).With("app_id", appID).Debug("the app is invalid")
		if _, err := kr.input.API.Delete(service, key); err != nil {
			// TODO customize logger
			slogerr.WithError(logger, err).With("app_id", appID).Debug("delete an invalid App from keyring")
		}
		return nil, nil
	}
	return app, nil
}

// Set stores an App config in the keyring.
// The key parameter identifies where to store the app config.
// Returns an error if the app config cannot be marshaled or stored.
func (kr *Keyring) SetApp(logger *slog.Logger, service string, appID int, app *App) error {
	s, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("marshal the app as JSON: %w", err)
	}
	if err := kr.input.API.Set(service, keyApp(appID), string(s)); err != nil {
		return fmt.Errorf("set an App in keyring: %w", err)
	}
	_, logins, err := kr.getLogins(service)
	if err != nil {
		slogerr.WithError(logger, err).Debug("get logins from keyring")
		return nil
	}
	// Delete access tokens with different client id
	for _, login := range logins {
		key := &AccessTokenKey{Login: login, AppID: appID}
		token, err := kr.GetAccessToken(logger, service, key)
		if err != nil {
			slogerr.WithError(logger, err).With("login", login).Debug("get access token from keyring")
			continue
		}
		if token.ClientID == app.ClientID {
			continue
		}
		if _, err := kr.DeleteAccessToken(service, key); err != nil {
			slogerr.WithError(logger, err).With("login", login).Debug("delete access token from keyring")
			continue
		}
	}
	return nil
}

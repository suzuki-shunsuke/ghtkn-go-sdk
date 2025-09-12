// Package keyring provides secure storage for GitHub access tokens.
// It wraps the zalando/go-keyring library to store and retrieve tokens from the system keychain.
package keyring

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

const DefaultServiceKey = "github.com/suzuki-shunsuke/ghtkn"

// dateFormat defines the standard format for date strings in the keyring.
const dateFormat = time.RFC3339

// ParseDate parses a date string in RFC3339 format.
// It returns a time.Time value or an error if the string cannot be parsed.
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse a date string: %w", err)
	}
	return t, nil
}

// FormatDate formats a time value as an RFC3339 string.
// This is the standard format used for expiration dates in the keyring.
func FormatDate(t time.Time) string {
	return t.Format(dateFormat)
}

// AccessToken represents a GitHub App access token stored in the keyring.
// It contains the token value, expiration information, and user login details.
type AccessToken struct {
	AccessToken    string    `json:"access_token"`    // The OAuth access token for GitHub API authentication
	ExpirationDate time.Time `json:"expiration_date"` // RFC3339 formatted expiration timestamp
	Login          string    `json:"login"`           // The GitHub user login associated with the token
	ClientID       string    `json:"client_id"`
}

func (t *AccessToken) Validate() error {
	if t.AccessToken == "" {
		return fmt.Errorf("access token is required")
	}
	if t.ExpirationDate.IsZero() {
		return fmt.Errorf("expiration date is required")
	}
	if t.Login == "" {
		return fmt.Errorf("login is required")
	}
	if t.ClientID == "" {
		return fmt.Errorf("client id is required")
	}
	return nil
}

type AccessTokenKey struct {
	Login string
	AppID int
}

func (k *AccessTokenKey) String() string {
	return fmt.Sprintf("access_tokens/%s/%d", k.Login, k.AppID)
}

// Keyring manages access tokens in the system keychain.
// It provides methods to get, set, and remove tokens securely.
type Keyring struct {
	input *Input
}

type Input struct {
	API API
}

// New creates a new Keyring instance with the specified service name.
// The keyService parameter is used as the service identifier in the system keychain.
func New(input *Input) *Keyring {
	return &Keyring{
		input: input,
	}
}

type API interface {
	Get(service, user string) (string, bool, error)
	Set(service, user, password string) error
	Delete(service, user string) (bool, error)
}

func NewInput() *Input {
	return &Input{
		API: NewAPI(),
	}
}

// Get retrieves an access token from the keyring.
// The key parameter identifies the token to retrieve.
// Returns the token or an error if the token cannot be found or unmarshaled.
func (kr *Keyring) GetAccessToken(logger *slog.Logger, service string, key *AccessTokenKey) (*AccessToken, error) {
	s, exist, err := kr.input.API.Get(service, key.String())
	if err != nil {
		return nil, fmt.Errorf("get a GitHub Access token in keyring: %w", err)
	}
	if !exist {
		return nil, nil
	}
	token := &AccessToken{}
	if err := json.Unmarshal([]byte(s), token); err != nil {
		slogerr.WithError(logger, err).With("login", key.Login).With("app_id", key.AppID).Debug("unmarshal the token as JSON")
		// Delete the invalid token
		if _, err := kr.DeleteAccessToken(service, key); err != nil {
			// TODO customize logger
			slogerr.WithError(logger, err).With("login", key.Login).With("app_id", key.AppID).Debug("delete an invalid AccessToken from keyring")
		}
		return nil, nil
	}
	// Validate and delete the invalid token
	if err := token.Validate(); err != nil {
		slogerr.WithError(logger, err).With("login", key.Login).With("app_id", key.AppID).Debug("the token is invalid")
		if _, err := kr.input.API.Delete(service, key.String()); err != nil {
			// TODO customize logger
			slogerr.WithError(logger, err).With("login", key.Login).With("app_id", key.AppID).Debug("delete an invalid AccessToken from keyring")
		}
		return nil, nil
	}
	return token, nil
}

func (kr *Keyring) getLogins(service string) (string, []string, error) {
	loginsStr, _, err := kr.input.API.Get(service, "logins")
	if err != nil {
		return "", nil, fmt.Errorf("get logins from keyring: %w", err)
	}
	if loginsStr == "" {
		return "", nil, nil
	}
	return loginsStr, strings.Split(loginsStr, ","), nil
}

// SetAccessToken stores an access token in the keyring.
// The key parameter identifies where to store the token.
// Returns an error if the token cannot be marshaled or stored.
func (kr *Keyring) SetAccessToken(logger *slog.Logger, service string, key *AccessTokenKey, token *AccessToken) error {
	s, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal the token as JSON: %w", err)
	}
	if err := kr.input.API.Set(service, key.String(), string(s)); err != nil {
		return fmt.Errorf("set a GitHub Access token in keyring: %w", err)
	}
	loginsStr, logins, err := kr.getLogins(service)
	if err != nil {
		slogerr.WithError(logger, err).Debug("get logins from keyring")
		return nil
	}
	loginsM := make(map[string]struct{}, len(logins)+1)
	if _, ok := loginsM[key.Login]; !ok {
		return nil
	}
	if loginsStr == "" {
		loginsStr = key.Login
	} else {
		loginsStr += "," + key.Login
	}
	if err := kr.input.API.Set(service, "logins", loginsStr); err != nil {
		slogerr.WithError(logger, err).Debug("update logins in keyring")
		return nil
	}
	return nil
}

func (kr *Keyring) DeleteAccessToken(service string, key *AccessTokenKey) (bool, error) {
	return kr.input.API.Delete(service, key.String())
}

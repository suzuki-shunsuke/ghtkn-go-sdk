// Package keyring provides the public access token type stored by ghtkn.
package keyring

import (
	"errors"
	"time"
)

// AccessToken represents a GitHub App access token stored in the keyring.
// It contains the token value, expiration information, and user login details.
type AccessToken struct {
	AccessToken    string    `json:"access_token"`    // The OAuth access token for GitHub API authentication
	ExpirationDate time.Time `json:"expiration_date"` // RFC3339 formatted expiration timestamp
	Login          string    `json:"login"`           // The GitHub user login associated with the token
}

func (at *AccessToken) Validate() error {
	if at.AccessToken == "" {
		return errors.New("access_token is required")
	}
	if at.ExpirationDate.IsZero() {
		return errors.New("expiration_date is required")
	}
	if at.Login == "" {
		return errors.New("login is required")
	}
	return nil
}

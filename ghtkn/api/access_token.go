package api

import (
	"errors"
	"time"
)

// AccessToken represents a GitHub App access token: the token a caller receives from
// Get, and the form the backends (keyring, text file, agent) persist.
//
// It never carries a refresh token. Only the ghtkn agent obtains refresh tokens, and it
// keeps them server-side: it stores them in its own encrypted store and strips them from
// the token it hands back, so a refresh token never reaches an SDK caller.
type AccessToken struct {
	AccessToken    string    `json:"access_token"`    // The OAuth access token for GitHub API authentication
	ExpirationDate time.Time `json:"expiration_date"` // RFC3339 formatted expiration timestamp
}

func (at *AccessToken) Validate() error {
	if at.AccessToken == "" {
		return errors.New("access_token is required")
	}
	if at.ExpirationDate.IsZero() {
		return errors.New("expiration_date is required")
	}
	return nil
}

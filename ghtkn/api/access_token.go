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
	AccessToken string `json:"access_token"` // The OAuth access token for GitHub API authentication
	// ExpirationDate is when the token expires. The zero time means it never expires,
	// which is what a GitHub App with user-token expiration disabled issues.
	ExpirationDate time.Time `json:"expiration_date"`
}

func (at *AccessToken) Validate() error {
	if at.AccessToken == "" {
		return errors.New("access_token is required")
	}
	// ExpirationDate is not required: the zero time is a valid value meaning the token
	// never expires (a GitHub App with user-token expiration disabled). The expiry checks
	// read the zero time as never-expiring.
	return nil
}

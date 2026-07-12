package api

import (
	"errors"
	"time"
)

// AccessToken represents a GitHub App access token stored in the keyring.
// It contains the token value and expiration information.
type AccessToken struct {
	AccessToken                string    `json:"access_token"`                  // The OAuth access token for GitHub API authentication
	ExpirationDate             time.Time `json:"expiration_date"`               // RFC3339 formatted expiration timestamp
	RefreshToken               string    `json:"refresh_token"`                 // The OAuth refresh token for GitHub API authentication
	RefreshTokenExpirationDate time.Time `json:"refresh_token_expiration_date"` // RFC3339 formatted expiration timestamp
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

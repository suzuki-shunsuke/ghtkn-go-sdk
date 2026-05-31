package keyring_test

import (
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
)

func TestAccessToken_Validate(t *testing.T) {
	t.Parallel()

	exp := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		token   *keyring.AccessToken
		wantErr bool
	}{
		{
			name:    "valid",
			token:   &keyring.AccessToken{AccessToken: "token", ExpirationDate: exp, Login: "octocat"},
			wantErr: false,
		},
		{
			name:    "missing access_token",
			token:   &keyring.AccessToken{ExpirationDate: exp, Login: "octocat"},
			wantErr: true,
		},
		{
			name:    "missing expiration_date",
			token:   &keyring.AccessToken{AccessToken: "token", Login: "octocat"},
			wantErr: true,
		},
		{
			name:    "missing login",
			token:   &keyring.AccessToken{AccessToken: "token", ExpirationDate: exp},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.token.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

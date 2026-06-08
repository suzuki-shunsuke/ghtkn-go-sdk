package api_test

import (
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
)

func TestAccessToken_Validate(t *testing.T) {
	t.Parallel()

	exp := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		token   *api.AccessToken
		wantErr bool
	}{
		{
			name:    "valid",
			token:   &api.AccessToken{AccessToken: "token", ExpirationDate: exp},
			wantErr: false,
		},
		{
			name:    "missing access_token",
			token:   &api.AccessToken{ExpirationDate: exp},
			wantErr: true,
		},
		{
			name:    "missing expiration_date",
			token:   &api.AccessToken{AccessToken: "token"},
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

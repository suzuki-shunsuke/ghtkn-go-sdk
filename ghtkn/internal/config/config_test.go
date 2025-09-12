package config_test

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestConfig_Validate(t *testing.T) { //nolint:funlen
	t.Parallel()
}

func TestApp_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		app     *config.App
		wantErr bool
	}{
		{
			name: "valid app",
			app: &config.App{
				Name:     "test-app",
				ClientID: "xxx",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			app: &config.App{
				Name:     "",
				ClientID: "xxx",
			},
			wantErr: true,
		},
		{
			name: "zero app_id",
			app: &config.App{
				Name:     "test-app",
				ClientID: "",
			},
			wantErr: true,
		},
		{
			name: "both empty",
			app: &config.App{
				Name:     "",
				ClientID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.app.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("App.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReader_Read(t *testing.T) { //nolint:funlen
	t.Parallel()
}

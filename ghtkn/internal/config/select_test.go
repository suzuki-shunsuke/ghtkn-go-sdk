package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

func TestConfig_SelectUser(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name string
		cfg  *config.Config
		key  string
		want *config.User
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cfg.SelectUser(tt.key)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SelectUser() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

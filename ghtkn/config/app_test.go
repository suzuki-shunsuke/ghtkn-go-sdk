package config_test

import (
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
)

func TestResolveApp(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Apps: []*config.App{
		{Name: "first", ClientID: "Iv1.first", GitOwner: "owner-a"},
		{Name: "second", ClientID: "Iv1.second", GitOwner: "owner-b"},
	}}

	tests := []struct {
		name  string
		cfg   *config.Config
		key   string
		owner string
		want  string // expected app Name, or "" when the result is nil
	}{
		{name: "owner match wins over key", cfg: cfg, key: "first", owner: "owner-b", want: "second"},
		{name: "key match", cfg: cfg, key: "second", owner: "", want: "second"},
		{name: "key not found is nil", cfg: cfg, key: "missing", owner: "", want: ""},
		{name: "both empty is the first app", cfg: cfg, key: "", owner: "", want: "first"},
		{name: "owner not found falls through to key", cfg: cfg, key: "first", owner: "owner-x", want: "first"},
		{name: "owner not found and empty key is the first app", cfg: cfg, key: "", owner: "owner-x", want: "first"},
		{name: "nil config is nil", cfg: nil, key: "", owner: "", want: ""},
		{name: "no apps is nil", cfg: &config.Config{}, key: "", owner: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := config.ResolveApp(tt.cfg, tt.key, tt.owner)
			gotName := ""
			if got != nil {
				gotName = got.Name
			}
			if gotName != tt.want {
				t.Errorf("ResolveApp() = %q, want %q", gotName, tt.want)
			}
		})
	}
}

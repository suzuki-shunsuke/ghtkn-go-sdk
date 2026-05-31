package ghtkn

import (
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
)

func TestFromKeyringAccessToken(t *testing.T) {
	t.Parallel()

	exp := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	if got := fromKeyringAccessToken(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}

	got := fromKeyringAccessToken(&keyring.AccessToken{
		AccessToken:    "token",
		ExpirationDate: exp,
		Login:          "octocat",
	})
	want := &AccessToken{AccessToken: "token", ExpirationDate: exp, Login: "octocat"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}

func TestFromConfigApp(t *testing.T) {
	t.Parallel()

	if got := fromConfigApp(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}

	got := fromConfigApp(&config.App{Name: "app", ClientID: "cid", GitOwner: "owner"})
	want := &AppConfig{Name: "app", ClientID: "cid", GitOwner: "owner"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}

func TestFromDeviceCodeResponse(t *testing.T) {
	t.Parallel()

	if got := fromDeviceCodeResponse(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}

	got := fromDeviceCodeResponse(&deviceflow.DeviceCodeResponse{
		DeviceCode:      "dc",
		UserCode:        "uc",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       900,
		Interval:        5,
	})
	want := &DeviceCodeResponse{
		DeviceCode:      "dc",
		UserCode:        "uc",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       900,
		Interval:        5,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}

func TestToAPIInputGet(t *testing.T) {
	t.Parallel()

	if got := toAPIInputGet(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}

	got := toAPIInputGet(&InputGet{
		KeyringService: "svc",
		AppName:        "app",
		ConfigFilePath: "/path/to/config.yaml",
		AppOwner:       "owner",
		MinExpiration:  time.Hour,
	})
	want := &api.InputGet{
		KeyringService: "svc",
		AppName:        "app",
		ConfigFilePath: "/path/to/config.yaml",
		AppOwner:       "owner",
		MinExpiration:  time.Hour,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Error(diff)
	}
}

func TestToLogLogger(t *testing.T) {
	t.Parallel()

	if got := toLogLogger(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}

	called := map[string]bool{}
	l := &Logger{
		Expire:                            func(*slog.Logger, time.Time) { called["expire"] = true },
		FailedToOpenBrowser:               func(*slog.Logger, error) { called["browser"] = true },
		FailedToGetAccessTokenFromKeyring: func(*slog.Logger, error) { called["keyring"] = true },
		AccessTokenIsNotFoundInKeyring:    func(*slog.Logger) { called["notfound"] = true },
	}
	got := toLogLogger(l)
	if got == nil {
		t.Fatal("toLogLogger returned nil")
	}
	// The function fields should be carried over (verified by invoking them).
	got.Expire(nil, time.Time{})
	got.FailedToOpenBrowser(nil, nil)
	got.FailedToGetAccessTokenFromKeyring(nil, nil)
	got.AccessTokenIsNotFoundInKeyring(nil)
	for _, k := range []string{"expire", "browser", "keyring", "notfound"} {
		if !called[k] {
			t.Errorf("function field %q was not carried over", k)
		}
	}
}

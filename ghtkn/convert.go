package ghtkn

import (
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// fromKeyringAccessToken converts an internal keyring.AccessToken to the public AccessToken.
func fromKeyringAccessToken(t *keyring.AccessToken) *AccessToken {
	if t == nil {
		return nil
	}
	return &AccessToken{
		AccessToken:    t.AccessToken,
		ExpirationDate: t.ExpirationDate,
		Login:          t.Login,
	}
}

// fromConfigApp converts an internal config.App to the public AppConfig.
func fromConfigApp(a *config.App) *AppConfig {
	if a == nil {
		return nil
	}
	return &AppConfig{
		Name:     a.Name,
		ClientID: a.ClientID,
		GitOwner: a.GitOwner,
	}
}

// fromDeviceCodeResponse converts an internal deviceflow.DeviceCodeResponse to the public DeviceCodeResponse.
func fromDeviceCodeResponse(d *deviceflow.DeviceCodeResponse) *DeviceCodeResponse {
	if d == nil {
		return nil
	}
	return &DeviceCodeResponse{
		DeviceCode:      d.DeviceCode,
		UserCode:        d.UserCode,
		VerificationURI: d.VerificationURI,
		ExpiresIn:       d.ExpiresIn,
		Interval:        d.Interval,
	}
}

// toAPIInputGet converts the public InputGet to the internal api.InputGet.
func toAPIInputGet(in *InputGet) *api.InputGet {
	if in == nil {
		return nil
	}
	return &api.InputGet{
		KeyringService: in.KeyringService,
		AppName:        in.AppName,
		ConfigFilePath: in.ConfigFilePath,
		AppOwner:       in.AppOwner,
		MinExpiration:  in.MinExpiration,
	}
}

// toLogLogger converts the public Logger to the internal log.Logger.
func toLogLogger(l *Logger) *log.Logger {
	if l == nil {
		return nil
	}
	return &log.Logger{
		Expire:                            l.Expire,
		FailedToOpenBrowser:               l.FailedToOpenBrowser,
		FailedToGetAccessTokenFromKeyring: l.FailedToGetAccessTokenFromKeyring,
		AccessTokenIsNotFoundInKeyring:    l.AccessTokenIsNotFoundInKeyring,
	}
}

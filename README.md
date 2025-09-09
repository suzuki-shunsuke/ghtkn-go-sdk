# ghtkn-go-sdk

[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn-go-sdk/main/LICENSE)

Go SDK to enable your Go application to create GitHub User Access Tokens for GitHub Apps easily

## :warning: The status is still alpha

The API is still unstable.

## Usage

Please see [examples](examples) too.

1. Use ghtkn a configuration file and keyring.

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
client := ghtkn.New()
token, _, err := client.Get(context.Background(), logger, &ghtkn.InputGet{
	UseConfig:  true,
	UseKeyring: true,
})
```

You can customize the behaviour by the argument `ghtkn.InputGet`.

### Using logging libraries such as logrus, zap, and zerolog

This SDK uses slog.
If you want to use other libraries such as logrus, zap, zerolog, and so on, you can implement slog.Handler using those libraries.

The following libraries are useful:

- https://github.com/samber/slog-logrus
- https://github.com/samber/slog-zap
- https://github.com/samber/slog-zerolog

### Customize logger

You can customize logger.

```go
client.SetLogger(&log.Logger{
		Expire: func(logger *slog.Logger, exDate time.Time) {
			logger.Debug("access token expires", "expiration_date", keyring.FormatDate(exDate))
		},
		FailedToOpenBrowser: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to open the browser")
		},
		FailedToGetAccessTokenFromKeyring: func(logger *slog.Logger, err error) {
			slogerr.WithError(logger, err).Warn("failed to get access token from keyring")
		},
		AccessTokenIsNotFoundInKeyring: func(logger *slog.Logger) {
			logger.Info("access token is not found in keyring")
		},
	}
)
```

### Customize opening the browser

Coming soon.

### Customize showing the device code

```go
type UI struct {}
func (ui *UI) Show(deviceCode *DeviceCodeResponse, expirationDate time.Time) {
	fmt.Fprintf(d.stderr, "Please visit: %s\n", deviceCode.VerificationURI)
	fmt.Fprintf(d.stderr, "And enter code: %s\n", deviceCode.UserCode)
	fmt.Fprintf(d.stderr, "Expiration date: %s\n", expirationDate.Format(time.RFC3339))
}
client.SetDeviceCodeUI(&UI{})
```

### Pass a client id without configuration file and keyring

```go
token, _, err := client.Get(context.Background(), logger, &ghtkn.InputGet{
    ClientID: "xxx", // GitHub App client id
})
```

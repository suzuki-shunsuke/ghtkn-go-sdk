package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

func main() {
	// Create a GitHub App User Access Token by Client ID without configuration file and Keyring.
	// Usage:
	//   env CLIENT_ID=$YOUR_CLIENT_ID go run main.go
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := ghtkn.New()
	client.SetLogger(&ghtkn.Logger{
		Expire: func(logger *slog.Logger, exDate time.Time) {
			logger.Debug("access token expires", "expiration_date", exDate.Format(time.RFC3339))
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
	})
	token, _, err := client.Get(context.Background(), logger, &ghtkn.InputGet{
		UseConfig:  true,
		UseKeyring: true,
	})
	if err != nil {
		slogerr.WithError(logger, err).Error("failed to get token")
		return 1
	}
	fmt.Println("access token: ", token.AccessToken)
	fmt.Println("expiration date: ", token.ExpirationDate)
	return 0
}

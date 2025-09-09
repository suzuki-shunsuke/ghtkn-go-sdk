package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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
	token, _, err := client.Get(context.Background(), logger, &ghtkn.InputGet{
		ClientID: os.Getenv("CLIENT_ID"), // Please set your GitHub App Client ID
	})
	if err != nil {
		slogerr.WithError(logger, err).Error("failed to get token")
		return 1
	}
	fmt.Println("access token:", token.AccessToken)
	fmt.Println("expiration date:", token.ExpirationDate)
	return 0
}

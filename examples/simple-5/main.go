package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

func main() {
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

type clientIDReader struct{}

func (r *clientIDReader) Read(_ context.Context, _ *slog.Logger, app *ghtkn.AppConfig) (string, error) {
	fmt.Fprintln(os.Stderr, "Enter your GitHub App Client ID:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := scanner.Text()
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read client ID: %w", err)
	}
	return strings.TrimSpace(text), nil
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	client := ghtkn.New()
	client.SetClientIDReader(&clientIDReader{})
	token, _, err := client.Get(context.Background(), logger, &ghtkn.InputGet{})
	if err != nil {
		slogerr.WithError(logger, err).Error("failed to get token")
		return 1
	}
	fmt.Println("access token: ", token.AccessToken)
	fmt.Println("expiration date: ", token.ExpirationDate)
	return 0
}

package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

func main() {
	// Usage:
	//   go run main.go
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := core(logger); err != nil {
		slogerr.WithError(logger, err).Error("failed to get token")
		return 1
	}
	return 0
}

type UI struct{}

func (ui *UI) Show(_ context.Context, _ *slog.Logger, deviceCode *ghtkn.DeviceCodeResponse, expirationDate time.Time) error {
	fmt.Fprintf(os.Stderr, "Please access %s and enter code %s by %s\n", deviceCode.VerificationURI, deviceCode.UserCode, expirationDate.Format(time.RFC3339))
	return nil
}

type Browser struct{}

func (b *Browser) Open(ctx context.Context, logger *slog.Logger, url string) error {
	db := &ghtkn.DefaultBrowser{}
	fmt.Println("Please input enter key to continue...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return err
	}
	return db.Open(ctx, logger, url)
}

func core(logger *slog.Logger) error {
	client := ghtkn.New()
	client.SetDeviceCodeUI(&UI{})
	client.SetBrowser(&Browser{})

	token, _, err := client.Get(context.Background(), logger, &ghtkn.InputGet{
		UseConfig:  true,
		UseKeyring: ghtkn.Ptr(true),
	})
	if err != nil {
		return err
	}
	fmt.Println("access token: ", token.AccessToken)
	fmt.Println("expiration date: ", token.ExpirationDate)
	return nil
}

// Package get provides functionality to retrieve GitHub App access tokens.
// It serves both the standard 'get' command and the 'git-credential' helper command.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package ghtkn

import (
	"context"
	"log/slog"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/api"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

// Client manages the process of retrieving GitHub App access tokens.
// It coordinates between configuration reading, token caching, and token generation.
type Client struct {
	tm  tokenManager
	env *config.Env
}

// New creates a new Client instance with the provided input configuration.
func New() *Client {
	return &Client{
		tm:  api.New(api.NewInput()),
		env: config.NewEnv(os.Getenv, runtime.GOOS),
	}
}

type tokenManager interface {
	Get(ctx context.Context, logger *slog.Logger, input *api.InputGet) (*keyring.AccessToken, *config.App, error)
	SetLogger(logger *log.Logger)
	SetDeviceCodeUI(ui deviceflow.DeviceCodeUI)
	SetBrowser(browser deviceflow.Browser)
}

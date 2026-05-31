// Package api provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package api

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/spf13/afero"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/github"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	pubkeyring "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
)

// TokenManager manages the process of retrieving GitHub App access tokens.
// It coordinates between configuration reading, token caching, and token generation.
type TokenManager struct {
	input *Input
}

// New creates a new Controller instance with the provided input configuration.
func New(input *Input) *TokenManager {
	return &TokenManager{
		input: input,
	}
}

// Input contains all the dependencies and configuration needed by the Controller.
// It encapsulates file system access, configuration reading, token generation, and output handling.
// The IsGitCredential flag determines whether to format output for Git's credential helper protocol.
type Input struct {
	DeviceFlow   DeviceFlow       // Client for creating GitHub App tokens
	Keyring      Keyring          // Keyring for token storage
	Now          func() time.Time // Current time provider for testing
	Logger       *publog.Logger
	ConfigReader ConfigReader
	Getenv       func(string) string
	GOOS         string
	NewGitHub    func(ctx context.Context, token string) (GitHub, error)
}

// GitHub defines the interface for interacting with the GitHub API.
// It is used to retrieve authenticated user information needed for Git Credential Helper.
type GitHub interface {
	GetUser(ctx context.Context) (*github.User, error)
}

// NewInput creates a new Input instance with default production values.
// It sets up all necessary dependencies including file system, HTTP client, and keyring access.
func NewInput() *Input {
	ki := keyring.NewInput()
	return &Input{
		DeviceFlow:   deviceflow.NewClient(deviceflow.NewInput()),
		Keyring:      keyring.New(ki),
		Now:          time.Now,
		Logger:       log.NewLogger(),
		ConfigReader: config.NewReader(afero.NewOsFs()),
		Getenv:       os.Getenv,
		GOOS:         runtime.GOOS,
		NewGitHub: func(ctx context.Context, token string) (GitHub, error) {
			return github.New(ctx, token)
		},
	}
}

// Validate checks if the Input configuration is valid.
// It returns an error if the output format is neither empty nor "json".
func (i *Input) Validate() error {
	return nil
}

// DeviceFlow defines the interface for creating GitHub App access tokens.
type DeviceFlow interface {
	Create(ctx context.Context, logger *slog.Logger, clientID string) (*deviceflow.AccessToken, error)
	SetLogger(logger *publog.Logger)
	SetDeviceCodeUI(ui pubdeviceflow.DeviceCodeUI)
	SetBrowser(browser pubdeviceflow.Browser)
}

// Keyring defines the interface for storing and retrieving tokens from the system keyring.
type Keyring interface {
	Get(service, key string) (*pubkeyring.AccessToken, error)
	Set(service, key string, token *pubkeyring.AccessToken) error
}

// ConfigReader defines the interface for reading configuration files.
type ConfigReader interface {
	Read(cfg *pubconfig.Config, configFilePath string) error
}

type PasswordReader interface {
	Read(ctx context.Context, logger *slog.Logger, app *pubconfig.App) (string, error)
}

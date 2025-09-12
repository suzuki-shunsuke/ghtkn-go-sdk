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
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/github"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
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
	DeviceFlow     DeviceFlow       // Client for creating GitHub App tokens
	Keyring        Keyring          // Keyring for token storage
	Now            func() time.Time // Current time provider for testing
	Logger         *log.Logger
	ConfigReader   ConfigReader
	Getenv         func(string) string
	GOOS           string
	NewGitHub      func(ctx context.Context, token string) GitHub
	ClientIDReader PasswordReader
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
		NewGitHub: func(ctx context.Context, token string) GitHub {
			return github.New(ctx, token)
		},
		ClientIDReader: NewPasswordReader(os.Stderr),
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
	SetLogger(logger *log.Logger)
	SetDeviceCodeUI(ui deviceflow.DeviceCodeUI)
	SetBrowser(browser deviceflow.Browser)
}

// Keyring defines the interface for storing and retrieving tokens from the system keyring.
type Keyring interface {
	GetAccessToken(logger *slog.Logger, service string, key *keyring.AccessTokenKey) (*keyring.AccessToken, error)
	SetAccessToken(logger *slog.Logger, service string, key *keyring.AccessTokenKey, token *keyring.AccessToken) error
	DeleteAccessToken(service string, key *keyring.AccessTokenKey) (bool, error)
	GetApp(logger *slog.Logger, service string, appID int) (*keyring.App, error)
	SetApp(logger *slog.Logger, service string, appID int, app *keyring.App) error
}

// ConfigReader defines the interface for reading configuration files.
type ConfigReader interface {
	Read(cfg *config.Config, configFilePath string) error
}

type PasswordReader interface {
	Read(ctx context.Context, logger *slog.Logger, app *config.App) (string, error)
}

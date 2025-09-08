// Package api provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/apptoken"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/keyring"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
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
	AppTokenClient AppTokenClient   // Client for creating GitHub App tokens
	Keyring        Keyring          // Keyring for token storage
	Now            func() time.Time // Current time provider for testing
	Logger         *log.Logger
	ConfigReader   ConfigReader
}

// NewInput creates a new Input instance with default production values.
// It sets up all necessary dependencies including file system, HTTP client, and keyring access.
func NewInput() *Input {
	return &Input{
		AppTokenClient: apptoken.NewClient(apptoken.NewInput()),
		Keyring:        keyring.New(keyring.NewInput()),
		Now:            time.Now,
		Logger:         log.NewLogger(),
	}
}

// Validate checks if the Input configuration is valid.
// It returns an error if the output format is neither empty nor "json".
func (i *Input) Validate() error {
	return nil
}

// AppTokenClient defines the interface for creating GitHub App access tokens.
type AppTokenClient interface {
	Create(ctx context.Context, logger *slog.Logger, clientID string) (*apptoken.AccessToken, error)
	SetLogger(logger *log.Logger)
}

// Keyring defines the interface for storing and retrieving tokens from the system keyring.
type Keyring interface {
	Get(service, key string) (*keyring.AccessToken, error)
	Set(service, key string, token *keyring.AccessToken) error
}

// ConfigReader defines the interface for reading configuration files.
type ConfigReader interface {
	Read(cfg *config.Config, configFilePath string) error
}

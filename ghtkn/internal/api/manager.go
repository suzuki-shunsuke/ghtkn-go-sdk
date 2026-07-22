// Package api provides functionality to retrieve GitHub App access tokens.
// It handles token retrieval from the keyring cache and token generation/renewal when needed.
package api

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/api"
	pubconfig "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/config"
	pubdeviceflow "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/backend"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
	publog "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/log"
	"github.com/suzuki-shunsuke/go-revoke-github-access-token/revoke"
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
	DeviceFlow   deviceFlow // Client for creating GitHub App tokens
	Backend      Backend    // Keyring for token storage
	Revoker      revoker    // Client for revoking credentials
	Logger       *publog.Logger
	ConfigReader configReader
	Getenv       func(string) string
	GOOS         string
}

// NewInput creates a new Input instance with default production values.
// It sets up all necessary dependencies including file system, HTTP client, and keyring access.
//
// The storage backend is not built here: its type can come from the config file
// (backend.type), which isn't read until Get/Revoke, so the backend is resolved
// lazily then. Backend is left nil and built on demand by resolveBackend.
func NewInput(getEnv func(string) string) (*Input, error) {
	return &Input{
		DeviceFlow:   deviceflow.NewClient(deviceflow.NewInput()),
		Revoker:      revoke.New(nil),
		Logger:       log.NewLogger(),
		ConfigReader: config.NewReader(),
		Getenv:       getEnv,
		GOOS:         runtime.GOOS,
	}, nil
}

// resolveBackend returns the storage backend to use. An injected backend
// (Input.Backend, e.g. set by a test or an SDK consumer) is honored as is.
// Otherwise the backend is built from cfg's backend.type, defaulting to the OS
// keyring. cfg must be the effective config (see loadConfig): GHTKN_BACKEND is folded
// into backend.type upstream, so a cfg read straight from the file selects the wrong
// backend.
func (tm *TokenManager) resolveBackend(logger *slog.Logger, cfg *pubconfig.Config) (Backend, error) {
	if tm.input.Backend != nil {
		return tm.input.Backend, nil
	}
	return backend.New(resolveBackendType(cfg.Backend), tm.input.Getenv, tm.input.Logger, logger)
}

// Validate checks if the Input configuration is valid.
// It returns an error if the output format is neither empty nor "json".
func (i *Input) Validate() error {
	return nil
}

// deviceFlow defines the interface for creating GitHub App access tokens.
type deviceFlow interface {
	Create(ctx context.Context, logger *slog.Logger, input *deviceflow.InputCreate) (*deviceflow.AccessToken, error)
	Show(ctx context.Context, logger *slog.Logger, input *deviceflow.InputCreate, deviceCode *pubdeviceflow.DeviceCodeResponse) error
	SetLogger(logger *publog.Logger)
	SetOnetimeCodeUI(ui pubdeviceflow.OnetimeCodeUI)
	SetBrowser(browser pubdeviceflow.Browser)
	SetCopyOnetimeCodeToClipboard(f pubdeviceflow.CopyTextToClipboard)
}

// Backend defines the interface for storing and retrieving tokens from the system keyring.
type Backend interface {
	Get(ctx context.Context, clientID string) (*api.AccessToken, error)
	Set(ctx context.Context, clientID string, token *api.AccessToken) error
	Delete(ctx context.Context, clientID string) error
	// SupportsDeviceFlow reports whether the backend owns the token lifecycle
	// server-side (the agent). When true, expiration-aware reads, device-flow token
	// creation, and revocation are driven through the backend below instead of the
	// client-side equivalents (GetActive, BeginDeviceFlow/PollDeviceFlow, RevokeToken).
	SupportsDeviceFlow() bool
	GetActive(ctx context.Context, clientID string, minExpiration time.Duration) (*api.AccessToken, error)
	BeginDeviceFlow(ctx context.Context, clientID string, minExpiration time.Duration) (*api.AccessToken, *pubdeviceflow.DeviceCodeResponse, error)
	PollDeviceFlow(ctx context.Context, clientID string, minExpiration time.Duration) (*api.AccessToken, error)
	RevokeTokens(ctx context.Context, clientIDs []string) (revokeFailed, cleanupFailed []string, err error)
}

// revoker defines the interface for revoking GitHub credentials.
type revoker interface {
	Revoke(ctx context.Context, tokens []string) error
}

// configReader defines the interface for reading configuration files.
type configReader interface {
	Read(cfg *pubconfig.Config, configFilePath string) error
}

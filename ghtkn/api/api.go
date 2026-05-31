// Package api provides the public request types for the ghtkn client.
package api

import "time"

// InputGet contains the input parameters for token retrieval operations.
// It provides configuration options for specifying which app to use,
// where to find configuration, and token expiration requirements.
type InputGet struct {
	KeyringService string        // Service name for keyring storage (defaults to the keyring's default service)
	AppName        string        // Name of the app to use (defaults to GHTKN_APP environment variable)
	ConfigFilePath string        // Path to configuration file (auto-detected if empty)
	AppOwner       string        // GitHub App Owner
	MinExpiration  time.Duration // Minimum time before token expiration to trigger renewal
}

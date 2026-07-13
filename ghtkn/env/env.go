// Package env is the single registry of the GHTKN_* environment variable names ghtkn
// reads. Both the SDK and the ghtkn CLI reference these constants instead of scattering
// string literals, so each name is defined in exactly one place.
package env

const (
	App              = "GHTKN_APP"
	AgentKey         = "GHTKN_AGENT_KEY"
	AgentSocket      = "GHTKN_AGENT_SOCKET"
	AgentTokenDir    = "GHTKN_AGENT_TOKEN_DIR"
	Backend          = "GHTKN_BACKEND"
	Clipboard        = "GHTKN_CLIPBOARD"
	Config           = "GHTKN_CONFIG"
	Enable           = "GHTKN_ENABLE"
	EnableDeviceFlow = "GHTKN_ENABLE_DEVICE_FLOW"
	GitApp           = "GHTKN_GIT_APP"
	GitHubToken      = "GHTKN_GITHUB_TOKEN"
	LogLevel         = "GHTKN_LOG_LEVEL"
	MinExpiration    = "GHTKN_MIN_EXPIRATION"
	OpenBrowser      = "GHTKN_OPEN_BROWSER"
	OutputFormat     = "GHTKN_OUTPUT_FORMAT"
	TextBackendDir   = "GHTKN_TEXT_BACKEND_DIR"
)

// All lists every GHTKN_* variable defined above. Iterate it (e.g. `ghtkn info`) so an
// environment dump can never omit a variable. Keep it in sync with the constants; the
// guard test in this package fails if they diverge.
var All = []string{ //nolint:gochecknoglobals // an intentional read-only registry
	App,
	AgentKey,
	AgentSocket,
	AgentTokenDir,
	Backend,
	Clipboard,
	Config,
	Enable,
	EnableDeviceFlow,
	GitApp,
	GitHubToken,
	LogLevel,
	MinExpiration,
	OpenBrowser,
	OutputFormat,
	TextBackendDir,
}

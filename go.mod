module github.com/suzuki-shunsuke/ghtkn-go-sdk

go 1.26.5

// replace github.com/suzuki-shunsuke/go-github-device-flow v0.0.1 => ../go-github-device-flow

require (
	github.com/google/go-cmp v0.7.0
	github.com/suzuki-shunsuke/go-github-device-flow v0.0.2-0.20260723200608-b91d9018984b
	github.com/suzuki-shunsuke/go-revoke-github-access-token v0.0.2-0.20260723202049-b59e884ad0ce
	github.com/suzuki-shunsuke/slog-error v0.2.2
	github.com/zalando/go-keyring v0.2.8
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sys v0.47.0
	golang.org/x/term v0.45.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
)

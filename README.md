# ghtkn-go-sdk

[![MIT LICENSE](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn-go-sdk/main/LICENSE) [![Go Reference](https://pkg.go.dev/badge/github.com/suzuki-shunsuke/ghtkn-go-sdk.svg)](https://pkg.go.dev/github.com/suzuki-shunsuke/ghtkn-go-sdk) | [Versioning Policy](https://github.com/suzuki-shunsuke/versioning-policy/blob/v0.1.0/POLICY.md)

Go SDK to enable your Go application to create GitHub User Access Tokens for GitHub Apps easily.

## Examples

- [Using configuration file and keyring](examples/simple-1/main.go)
- [Customizing Logging](examples/simple-4/main.go)
- [Customizing opening the browser](examples/simple-3/main.go)
- [Customizing showing the device code](examples/simple-3/main.go)

## Using logging libraries such as logrus, zap, and zerolog

This SDK uses [slog](https://pkg.go.dev/log/slog).
If you want to use other libraries such as logrus, zap, zerolog, and so on, you can implement slog.Handler using those libraries.

The following libraries are useful:

- https://github.com/samber/slog-logrus
- https://github.com/samber/slog-zap
- https://github.com/samber/slog-zerolog

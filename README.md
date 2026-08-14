# ghtkn-go-sdk

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/suzuki-shunsuke/ghtkn-go-sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/suzuki-shunsuke/ghtkn-go-sdk.svg)](https://pkg.go.dev/github.com/suzuki-shunsuke/ghtkn-go-sdk) | [Versioning Policy](https://github.com/suzuki-shunsuke/versioning-policy/blob/v0.1.0/POLICY.md)

Go SDK to enable your Go application to create GitHub User Access Tokens for GitHub Apps easily.

## Getting a token vs authenticating

`Client.Get` reads the token stored by ghtkn and never creates one.
When no valid token is stored it fails with `ghtkn.ErrDisableDeviceFlow`, because creating a token runs GitHub's OAuth Device Flow, which is interactive.
Ask the user to run `ghtkn auth` in their terminal, then try again.
Get is safe to call from a background or non-interactive process: it never blocks waiting for a user.

`Client.Auth` is the only method that runs the Device Flow, so a token is never created on a caller's behalf.
Call it only from a foreground, interactive context.
Most applications should call `Get` and leave authentication to the `ghtkn` CLI.

## Examples

- [Simple](examples/simple-1/main.go)
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

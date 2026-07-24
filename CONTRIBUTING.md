# Contributing

Please read the following document.

- https://github.com/suzuki-shunsuke/oss-contribution-guide

## Guides

- Keep the public API minimal. Exported identifiers are a compatibility promise, so every one that is not needed by a consumer makes the API harder to keep stable. Prefer unexported identifiers, and export something only when a consumer genuinely needs it. For example, when an error value is exported so callers can match it with `errors.Is`, do not also add an `IsXxx` helper unless a consumer actually branches on it.

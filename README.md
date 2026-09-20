# cli

## Setup

```sh
git config core.hooksPath .githooks
```

The `pre-push` hook runs the same gates as CI (`gofmt`, `go vet`, `golangci-lint`, `go test`, `govulncheck`) before every push.

## License

Apache License 2.0 — use it, ship it, embed it in your pipelines freely.

The broker it talks to is source-available under BSL 1.1; see
https://github.com/bidirekt/broker for its terms.

# cli

## Setup

```sh
git config core.hooksPath .githooks
```

The `pre-push` hook runs the same gates as CI (`gofmt`, `go vet`, `golangci-lint`, `go test`, `govulncheck`) before every push.

## E2E tests

The end-to-end suite in `tests/e2e` runs the CLI against a released broker image, `ghcr.io/bidirekt/broker:0.1.0-rc.1`. It needs Docker, bats, bats-support, bats-assert and gettext (for `envsubst`).

On macOS, Homebrew installs bats and gettext but has no bats-support or bats-assert, so clone those at the versions CI uses into one folder and point `BATS_LIB_PATH` at it:

```sh
brew install bats-core gettext
git clone --depth 1 -b v0.3.0 https://github.com/bats-core/bats-support "$HOME/.local/lib/bats/bats-support"
git clone --depth 1 -b v2.1.0 https://github.com/bats-core/bats-assert "$HOME/.local/lib/bats/bats-assert"
export BATS_LIB_PATH="$HOME/.local/lib/bats"
```

The suite runs the `bidirekt` it finds on `PATH` and never builds it. Build the working tree into the repository root, where `/bidirekt` is gitignored, and put the root first on `PATH`:

```sh
go build -o bidirekt ./cmd/bidirekt
export PATH="$PWD:$PATH"
```

Start the broker and its Postgres, then run the suite:

```sh
docker compose -f tests/e2e/compose.yaml up -d --wait
BIDIREKT_BROKER_URL=http://localhost:8080 bats tests/e2e
```

Each run uses its own participant and environment names, so running the suite again against the same broker passes. `docker compose -f tests/e2e/compose.yaml down` stops both containers and throws the data away (Postgres keeps it on tmpfs).

## License

Apache License 2.0 — use it, ship it, embed it in your pipelines freely.

The broker it talks to is source-available under BSL 1.1; see
https://github.com/bidirekt/broker for its terms.

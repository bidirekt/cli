# cli

[![CI](https://img.shields.io/github/actions/workflow/status/bidirekt/cli/ci.yml?branch=main&label=CI)](https://github.com/bidirekt/cli/actions/workflows/ci.yml) [![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/bidirekt/cli/badges/coverage.json)](https://github.com/bidirekt/cli/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/bidirekt/cli)](https://github.com/bidirekt/cli/releases) [![License](https://img.shields.io/github/license/bidirekt/cli)](LICENSE) [![Go](https://img.shields.io/github/go-mod/go-version/bidirekt/cli)](go.mod)

`bidirekt` is the command line for the Bidirekt broker. In Bidirekt, each side of an integration declares in a contract file what it provides and what it consumes. The CLI publishes these contracts to the broker, asks it with `can-i-deploy` whether a version can be deployed to an environment without breaking a consumer or a provider deployed there, and records each deployment. The broker compares the contracts, so no service has to be running for the check.

## Install

```sh
curl -sSfL https://raw.githubusercontent.com/bidirekt/cli/main/install.sh | sh
```

Each release is also on https://github.com/bidirekt/cli/releases, as one archive per system.

The CLI also ships as the Docker image `ghcr.io/bidirekt/cli`.

All install options: https://bidirekt.com/docs/start/install/

## Quickstart

Point the CLI at your broker, then create the environment and the participant once:

```sh
export BIDIREKT_BROKER_URL=http://localhost:8080
bidirekt create-environment production
bidirekt create-participant petstore_web
```

For every version, publish its contract files, ask whether it can be deployed, deploy it, and record the deployment:

```sh
bidirekt publish contracts/*.yaml --participant petstore_web --version 2.3.0
bidirekt can-i-deploy petstore_web --version 2.3.0 --environment production
# deploy petstore_web 2.3.0 to production
bidirekt record-deployment petstore_web --version 2.3.0 --environment production
```

- Docs: https://bidirekt.com/docs/
- CLI reference, every command and its flags: https://bidirekt.com/docs/reference/cli/

## Development

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening an issue or a pull request.

### Setup

```sh
git config core.hooksPath .githooks
```

The `pre-push` hook runs the Go gates of CI (`gofmt`, `go vet`, `golangci-lint`, `go-arch-lint`, `go test`, `govulncheck`) before every push.

### E2E tests

The end-to-end suite in `tests/e2e` runs the CLI against a released broker image, `ghcr.io/bidirekt/broker:0.1.0`. It needs Docker, bats, bats-support, bats-assert and gettext (for `envsubst`).

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

Before the first test, the suite checks that the broker at `BIDIREKT_BROKER_URL` is the version pinned in `tests/e2e/compose.yaml`, and runs no test if it isn't.

Each run uses its own participant and environment names, so running the suite again against the same broker passes. `docker compose -f tests/e2e/compose.yaml down` stops both containers and throws the data away (Postgres keeps it on tmpfs).

## License

Apache License 2.0 — use it, ship it, embed it in your pipelines freely.

The broker it talks to is source-available under BSL 1.1; see
https://github.com/bidirekt/broker for its terms.

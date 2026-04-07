# CLAUDE.md

## Build & Test Commands

- `make build` — build the binary to `./bin/gh-app-auth`
- `make test` — run all tests (`go test ./...`)
- `make lint` — run golangci-lint (`golangci-lint run ./...`)
- `make check` — run tests, lint, and validate goreleaser config
- `make build-all` — cross-compile for macOS and Linux (amd64/arm64)
- `make clean` — remove build artifacts

## Workflow

- Always run `make lint` before pushing to catch golangci-lint errors locally.
- CI runs tests, linting, and a GoReleaser dry-run on every PR and push to main.
- Releases are triggered by pushing a `v*` tag.

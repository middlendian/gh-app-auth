# gh-app-auth v1 Design Spec

## Overview

`gh-app-auth` is a GitHub CLI extension that generates GitHub App installation access tokens. It authenticates as a GitHub App using a private key, resolves the target installation, and outputs a short-lived token that can be used with `gh` or any tool that accepts `GH_TOKEN`.

**Language:** Go
**CLI framework:** Cobra (`spf13/cobra`)
**License:** GPLv3

## CLI Interface

### Subcommands

- `gh app-auth` — prints help text with available subcommands
- `gh app-auth token` — prints an installation access token to stdout (nothing else)
- `gh app-auth completion [bash|zsh|fish|powershell]` — outputs shell completion script

### Persistent Flags (root command)

- `--repo owner/repo` — explicitly specify the target repository (overrides git remote auto-discovery)

## Project Structure

```
gh-app-auth/
├── main.go                  # Entry point, wires up cobra root command
├── cmd/
│   ├── root.go              # Root command, persistent --repo flag, help
│   ├── token.go             # `token` subcommand
│   └── completion.go        # `completion` subcommand
├── internal/
│   ├── auth/
│   │   ├── jwt.go           # JWT generation from app ID + private key
│   │   └── token.go         # Installation token minting (GitHub API)
│   ├── config/
│   │   └── config.go        # Reads env vars, resolves private key
│   ├── github/
│   │   └── client.go        # Thin HTTP client for GitHub API
│   └── repo/
│       └── repo.go          # Git remote discovery, owner/repo parsing
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

## Configuration

All configuration is via environment variables for v1. The code is structured so a config file layer can be added later without rearchitecting.

| Variable | Required | Description |
|---|---|---|
| `GH_APP_ID` | Yes | The GitHub App's numeric ID |
| `GH_APP_PRIVATE_KEY` | One of these | PEM-encoded private key (inline value) |
| `GH_APP_PRIVATE_KEY_FILE` | One of these | Path to PEM-encoded private key file |
| `GH_APP_INSTALLATION_ID` | No | Explicit installation ID (skips auto-discovery) |

- At least one of `GH_APP_PRIVATE_KEY` or `GH_APP_PRIVATE_KEY_FILE` must be set.
- If both are set, the inline value (`GH_APP_PRIVATE_KEY`) takes precedence.

## Authentication Flow

### Step 1 — Load config

Read environment variables. Parse the private key PEM (from inline value or file). Validate that required fields are present. Return a config struct with app ID, parsed RSA private key, and optional installation ID.

### Step 2 — Generate JWT

- Build a JWT with claims: `iss` = app ID, `iat` = now - 60s, `exp` = now + 10min
- Sign with RS256 using the private key
- Entirely local — no network call

### Step 3 — Resolve installation ID

If `GH_APP_INSTALLATION_ID` is set, use it directly. Otherwise:

1. Determine `owner/repo` from the `--repo` flag or git remote auto-discovery (see below)
2. Call `GET /repos/{owner}/{repo}/installation` with the JWT as bearer token
3. Extract `id` from the response

### Step 4 — Mint installation token

1. Call `POST /app/installations/{installation_id}/access_tokens` with the JWT as bearer token
2. Extract `token` from the response
3. Print token to stdout

## Repo Auto-Discovery

When `--repo` is not provided and `GH_APP_INSTALLATION_ID` is not set, the tool discovers the target repo from the local git context:

1. List all git remotes (`git remote`)
2. If exactly one remote exists — use it regardless of name
3. If multiple remotes exist and one is named `origin` — use `origin`
4. If multiple remotes exist and none is named `origin` — error: "multiple remotes found, use --repo to specify"
5. Parse `owner/repo` from the remote URL (supports both HTTPS and SSH formats)
6. Strip `.git` suffix if present

**Failure modes:**
- Not in a git repo → error with guidance to use `--repo` or `GH_APP_INSTALLATION_ID`
- Remote URL is not a GitHub URL → error with guidance to use `--repo`

## Package Responsibilities

### `internal/config`
Reads env vars, resolves private key (inline vs file path), validates required fields. Returns a config struct.

### `internal/auth/jwt.go`
Takes app ID + private key, returns a signed JWT string. Pure function, no I/O beyond crypto.

### `internal/auth/token.go`
Takes a JWT and installation ID, calls GitHub API, returns the access token string.

### `internal/github/client.go`
Thin wrapper around `net/http` for GitHub API calls. Sets `Authorization: Bearer <jwt>`, `Accept: application/vnd.github+json`. Parses responses and extracts meaningful error messages. Base URL defaults to `https://api.github.com`.

### `internal/repo`
Git remote discovery. Shells out to `git remote` to list and resolve remotes. Parses `owner/repo` from HTTPS and SSH URLs.

### `cmd/root.go`
Cobra root command. Registers persistent `--repo` flag. Displays help by default.

### `cmd/token.go`
Orchestrates the full flow: load config → generate JWT → resolve installation → mint token → print to stdout.

### `cmd/completion.go`
Cobra's built-in shell completion generation.

## Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework, subcommands, flags, help, completions |
| `github.com/golang-jwt/jwt/v5` | JWT creation and RS256 signing |

No other external dependencies. Everything else uses Go stdlib.

## Error Handling

All errors print to stderr and exit non-zero. Error messages are clear and actionable:

- Missing env vars → names the missing variable and what it's for
- Malformed private key → "failed to parse private key: ..."
- Not in a git repo → "not in a git repo; use --repo or set GH_APP_INSTALLATION_ID"
- App not installed on repo → "GitHub App is not installed on owner/repo"
- API errors → includes HTTP status and GitHub's error message

## Out of Scope for v1

The following are planned for future versions. The v1 architecture is designed to accommodate them.

- **Token caching** — Cache installation tokens by installation ID, respect `expires_at` with ~5 min buffer. Natural location: a cache layer wrapping `internal/auth/token.go`.
- **Multiple app identities** — Config file or `--app` flag to select between apps. The `internal/config` package can grow a config file layer in front of the env vars.
- **`gh app-auth activate`** — Subcommand to inject `GH_TOKEN` into the current shell session.
- **PATH-shadowing wrapper** — Separate binary that intercepts `gh` calls and auto-injects tokens. Builds on the extension primitive.
- **GitHub Enterprise support** — Override the API base URL via env var (e.g. `GH_APP_AUTH_BASE_URL`).

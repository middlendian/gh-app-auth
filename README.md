# gh-app-auth

GitHub CLI extension for GitHub App authentication and automatic token generation.

## Install

```sh
gh extension install middlendian/gh-app-auth
```

## Usage

```sh
# Set required environment variables
export GH_APP_ID=12345
export GH_APP_PRIVATE_KEY_FILE=/path/to/private-key.pem

# Get an installation token for the current repo
export GH_TOKEN=$(gh app-auth token)

# Get a token for a specific repo
export GH_TOKEN=$(gh app-auth token --repo owner/repo)

# Or provide the installation ID directly
export GH_APP_INSTALLATION_ID=67890
export GH_TOKEN=$(gh app-auth token)
```

## Configuration

| Variable | Required | Description |
|---|---|---|
| `GH_APP_ID` | Yes | The GitHub App's numeric ID |
| `GH_APP_PRIVATE_KEY` | One of these | PEM-encoded private key (inline) |
| `GH_APP_PRIVATE_KEY_FILE` | One of these | Path to PEM-encoded private key file |
| `GH_APP_INSTALLATION_ID` | No | Explicit installation ID (skips auto-discovery) |

## Shell Completions

```sh
# Bash
source <(gh app-auth completion bash)

# Zsh
gh app-auth completion zsh > "${fpath[1]}/_gh-app-auth"

# Fish
gh app-auth completion fish | source
```

## License

GPLv3 — see [LICENSE](LICENSE).

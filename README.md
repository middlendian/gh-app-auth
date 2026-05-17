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

# Get a token for a specific installation ID (skips repo-slug lookup)
export GH_TOKEN=$(gh app-auth token --installation-id 67890)

# Or provide the installation ID via env var
export GH_APP_INSTALLATION_ID=67890
export GH_TOKEN=$(gh app-auth token)
```

### Enumerate every installation

When you need to fan a request out across every org or user the App is
installed on, list the installations first and pick out the IDs:

```sh
gh app-auth installations list
# [
#   { "id": 12345678, "account": { "login": "alice",    "type": "User" } },
#   { "id": 87654321, "account": { "login": "acmecorp", "type": "Organization" } }
# ]

# All org installation IDs:
gh app-auth installations list | jq -r '.[] | select(.account.type == "Organization") | .id'

# Mint a token for each org and do something with it:
for id in $(gh app-auth installations list | jq -r '.[].id'); do
  GH_TOKEN=$(gh app-auth token --installation-id "$id") gh api /installation/repositories
done
```

The output is intentionally slim — only `id`, `account.login`, and
`account.type` — so consumers aren't coupled to the full GitHub schema.

### Installation ID precedence (for `token`)

When `token` figures out which installation to use, it checks, in order:

1. `--installation-id <id>` flag
2. `GH_APP_INSTALLATION_ID` env var
3. `--repo owner/repo` flag (then `GET /repos/{owner}/{repo}/installation`)
4. Git remote discovery in the current directory (same lookup)

`--installation-id` and `--repo` are mutually exclusive — pass one or
the other.

## Configuration

| Variable | Required | Description |
|---|---|---|
| `GH_APP_ID` | Yes | The GitHub App's numeric ID |
| `GH_APP_PRIVATE_KEY` | One of these | PEM-encoded private key (inline) |
| `GH_APP_PRIVATE_KEY_FILE` | One of these | Path to PEM-encoded private key file |
| `GH_APP_INSTALLATION_ID` | No | Explicit installation ID (skips auto-discovery) |

## Git Credential Helper

`gh-app-auth` can act as a [Git credential helper](https://git-scm.com/docs/gitcredentials), so that `git clone`, `git fetch`, `git push`, and other operations authenticate automatically using your GitHub App.

```sh
git config --global credential.https://github.com.helper '!gh-app-auth git-credential'
```

With this configured, any HTTPS Git operation against `github.com` will request a short-lived installation token from your App. Git sends the repository path to the helper, so the correct installation is resolved automatically — no `--repo` flag needed.

The helper only responds to `https://github.com` requests. Other hosts and protocols are left to your existing credential helpers.

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

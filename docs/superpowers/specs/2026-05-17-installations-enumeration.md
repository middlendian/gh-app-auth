# Installation Enumeration & ID-First Token Minting

**Status:** implemented
**Date:** 2026-05-17

## Motivation

Today `gh-app-auth token` resolves a single installation through `--repo`
(or `GH_APP_INSTALLATION_ID`). Callers that need to fan a request out
across every org an App is installed on have no way to learn the
installation IDs in the first place — they're stuck with whatever single
installation `--repo` selects.

Consumer context: callers that need cross-installation enumeration.

## What ships

### 1. `gh-app-auth installations list`

Mints an App-level JWT, calls `GET /app/installations` (paginated via the
`Link: rel="next"` header), and prints a slim JSON array to stdout.

Each element exposes only the fields callers actually use:

```json
[
  {
    "id": 12345678,
    "account": {
      "login": "middlendian",
      "type": "Organization"
    }
  }
]
```

Slimming the payload keeps the public contract small. GitHub adds fields
to `/app/installations` regularly — re-emitting the upstream schema would
drag every consumer along. If a caller needs more fields later we can
grow this struct (additive) rather than discovering breakage in the
field.

Output is pretty-printed (`json.MarshalIndent` with two-space indent) so
piping through `jq` Just Works.

Pagination follows `Link: rel="next"` until the header is absent. The
client treats the `next` URL as a full URL (it may differ from the
base URL the client was constructed with), then aggregates all pages
before printing — there's no streaming requirement for the v1 caller.

Errors are reported the same way `token` reports them: non-zero exit
plus a message on stderr.

### 2. `gh-app-auth token --installation-id <int64>`

A new token-scoped flag that names a single installation directly. When
set, the repo-slug → installation-ID lookup (and therefore `--repo` and
git remote discovery) is skipped entirely.

Precedence is documented in the flag help:

1. `--installation-id`
2. `GH_APP_INSTALLATION_ID` env
3. `--repo` (then `/repos/{owner}/{repo}/installation` lookup)
4. git remote discovery (then the same lookup)

`--installation-id` and `--repo` are rejected as a combination with a
clear error. They specify the same thing two different ways; silent
precedence between them would be a footgun. The env var, by contrast,
is a fallback users opt into globally and is allowed to coexist with
either flag (the flag wins, matching today's behavior for
`GH_APP_INSTALLATION_ID` vs `--repo`).

## Layout

```
cmd/
  installations.go        # new — `installations` parent + `list` subcommand
  token.go                # extended — adds --installation-id, plumbs into resolver
internal/auth/
  installations.go        # new — Installation type + ListInstallations (paginated)
  installations_test.go   # new — httptest-driven coverage
internal/github/
  client.go               # extended — GetPaginated helper that returns next URL
```

The `installations` parent command leaves room to grow
(`installations get`, etc.) without restructuring later. A flat
`installations-list` would also have worked; the parent form was chosen
for symmetry with the GitHub API's own grouping and for future
extensibility.

## Out of scope

- No changes to `git-credential` mode.
- No caching. `token` doesn't cache; `installations list` doesn't either.
  Callers cache.
- No filtering by org/account inside `installations list`. Consumers
  filter — `jq '.[] | select(.account.login == "foo")'` is one line.

# gh-app-auth v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a `gh` CLI extension that generates GitHub App installation access tokens.

**Architecture:** Cobra-based CLI with subcommand structure. Internal packages handle config loading (env vars), JWT generation (RS256), GitHub API calls (net/http), and git remote discovery. The `token` subcommand orchestrates the full flow.

**Tech Stack:** Go 1.25, spf13/cobra, golang-jwt/jwt/v5

---

## File Map

| File | Responsibility |
|---|---|
| `main.go` | Entry point — calls `cmd.Execute()` |
| `cmd/root.go` | Cobra root command, persistent `--repo` flag, help text |
| `cmd/token.go` | `token` subcommand — orchestrates config → JWT → resolve → mint → print |
| `cmd/completion.go` | `completion` subcommand — shell completion scripts |
| `internal/config/config.go` | Reads env vars, parses private key, returns config struct |
| `internal/config/config_test.go` | Tests for config loading |
| `internal/auth/jwt.go` | Generates signed JWT from app ID + private key |
| `internal/auth/jwt_test.go` | Tests for JWT generation |
| `internal/github/client.go` | Thin HTTP client for GitHub API |
| `internal/github/client_test.go` | Tests using httptest |
| `internal/repo/repo.go` | Git remote discovery + URL parsing |
| `internal/repo/repo_test.go` | Tests using temp git repos |
| `internal/auth/token.go` | Resolves installation ID + mints token via GitHub client |
| `internal/auth/token_test.go` | Tests for token minting |

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`, `main.go`, `cmd/root.go`

- [ ] **Step 1: Initialize Go module and install dependencies**

```bash
cd /Users/greghaskins/code/middlendian/gh-app-auth
go mod init github.com/middlendian/gh-app-auth
go get github.com/spf13/cobra@latest
go get github.com/golang-jwt/jwt/v5@latest
```

- [ ] **Step 2: Create `cmd/root.go`**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var repo string

var rootCmd = &cobra.Command{
	Use:   "gh-app-auth",
	Short: "GitHub App authentication for gh CLI",
	Long:  "Generate GitHub App installation access tokens for use with gh CLI and other tools.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&repo, "repo", "", "target repository in owner/repo format")
}
```

- [ ] **Step 3: Create `main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/middlendian/gh-app-auth/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Verify it builds and prints help**

```bash
go build -o gh-app-auth .
./gh-app-auth
```

Expected: prints help text with "GitHub App authentication for gh CLI" and `--repo` flag in the output.

- [ ] **Step 5: Commit**

```bash
git add main.go cmd/root.go go.mod go.sum
git commit -m "scaffold: Go module, cobra root command with --repo flag"
```

---

### Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`

- [ ] **Step 1: Write failing test — missing app ID**

```go
package config

import (
	"testing"
)

func TestLoad_MissingAppID(t *testing.T) {
	t.Setenv("GH_APP_ID", "")
	t.Setenv("GH_APP_PRIVATE_KEY", "")
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing GH_APP_ID")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/ -run TestLoad_MissingAppID -v
```

Expected: FAIL — `Load` not defined.

- [ ] **Step 3: Write failing test — missing private key**

```go
func TestLoad_MissingPrivateKey(t *testing.T) {
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", "")
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing private key")
	}
}
```

- [ ] **Step 4: Write failing test — inline private key**

```go
import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func generateTestPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block))
}

func TestLoad_InlinePrivateKey(t *testing.T) {
	pemStr := generateTestPEM(t)
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", pemStr)
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppID != "12345" {
		t.Errorf("AppID = %q, want %q", cfg.AppID, "12345")
	}
	if cfg.PrivateKey == nil {
		t.Fatal("PrivateKey is nil")
	}
}
```

- [ ] **Step 5: Write failing test — private key from file**

```go
func TestLoad_PrivateKeyFile(t *testing.T) {
	pemStr := generateTestPEM(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(keyPath, []byte(pemStr), 0600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", "")
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", keyPath)
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PrivateKey == nil {
		t.Fatal("PrivateKey is nil")
	}
}
```

- [ ] **Step 6: Write failing test — inline takes precedence over file**

```go
func TestLoad_InlineTakesPrecedenceOverFile(t *testing.T) {
	pemStr := generateTestPEM(t)
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", pemStr)
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "/nonexistent/path.pem")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	_, err := Load()
	if err != nil {
		t.Fatalf("should succeed with inline key even if file path is bogus: %v", err)
	}
}
```

- [ ] **Step 7: Write failing test — optional installation ID**

```go
func TestLoad_InstallationID(t *testing.T) {
	pemStr := generateTestPEM(t)
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", pemStr)
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "67890")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InstallationID != "67890" {
		t.Errorf("InstallationID = %q, want %q", cfg.InstallationID, "67890")
	}
}
```

- [ ] **Step 8: Implement `internal/config/config.go`**

```go
package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type Config struct {
	AppID          string
	PrivateKey     *rsa.PrivateKey
	InstallationID string // optional; empty means auto-discover
}

func Load() (*Config, error) {
	appID := os.Getenv("GH_APP_ID")
	if appID == "" {
		return nil, fmt.Errorf("GH_APP_ID is required (the GitHub App's numeric ID)")
	}

	key, err := loadPrivateKey()
	if err != nil {
		return nil, err
	}

	return &Config{
		AppID:          appID,
		PrivateKey:     key,
		InstallationID: os.Getenv("GH_APP_INSTALLATION_ID"),
	}, nil
}

func loadPrivateKey() (*rsa.PrivateKey, error) {
	pemData := os.Getenv("GH_APP_PRIVATE_KEY")
	if pemData == "" {
		path := os.Getenv("GH_APP_PRIVATE_KEY_FILE")
		if path == "" {
			return nil, fmt.Errorf("GH_APP_PRIVATE_KEY or GH_APP_PRIVATE_KEY_FILE is required")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading private key file: %w", err)
		}
		pemData = string(data)
	}

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key: no PEM block found")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return key, nil
}
```

- [ ] **Step 9: Run all config tests**

```bash
go test ./internal/config/ -v
```

Expected: all PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/config/
git commit -m "feat: config package — load app credentials from env vars"
```

---

### Task 3: JWT Generation

**Files:**
- Create: `internal/auth/jwt.go`, `internal/auth/jwt_test.go`

- [ ] **Step 1: Write failing test — JWT structure and claims**

```go
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func TestGenerateJWT(t *testing.T) {
	key := generateTestKey(t)
	now := time.Now()

	token, err := GenerateJWT("12345", key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse and validate the token
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("unexpected claims type")
	}

	// iss must be the app ID as a string
	iss, err := claims.GetIssuer()
	if err != nil {
		t.Fatalf("get issuer: %v", err)
	}
	if iss != "12345" {
		t.Errorf("iss = %q, want %q", iss, "12345")
	}

	// iat should be ~60s in the past
	iat, err := claims.GetIssuedAt()
	if err != nil {
		t.Fatalf("get iat: %v", err)
	}
	drift := now.Sub(iat.Time)
	if drift < 50*time.Second || drift > 70*time.Second {
		t.Errorf("iat drift = %v, want ~60s", drift)
	}

	// exp should be ~10min from now
	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatalf("get exp: %v", err)
	}
	ttl := exp.Time.Sub(now)
	if ttl < 9*time.Minute || ttl > 11*time.Minute {
		t.Errorf("ttl = %v, want ~10m", ttl)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/auth/ -run TestGenerateJWT -v
```

Expected: FAIL — `GenerateJWT` not defined.

- [ ] **Step 3: Implement `internal/auth/jwt.go`**

```go
package auth

import (
	"crypto/rsa"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(appID string, key *rsa.PrivateKey) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(key)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/auth/ -run TestGenerateJWT -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/jwt.go internal/auth/jwt_test.go
git commit -m "feat: JWT generation with RS256 signing"
```

---

### Task 4: GitHub API Client

**Files:**
- Create: `internal/github/client.go`, `internal/github/client_test.go`

- [ ] **Step 1: Write failing test — successful GET request**

```go
package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-jwt" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-jwt")
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept = %q, want %q", got, "application/vnd.github+json")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result struct {
		ID int `json:"id"`
	}
	err := client.Get(context.Background(), "/test", "test-jwt", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != 42 {
		t.Errorf("ID = %d, want 42", result.ID)
	}
}
```

- [ ] **Step 2: Write failing test — successful POST request**

```go
func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"token": "ghs_abc123"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result struct {
		Token string `json:"token"`
	}
	err := client.Post(context.Background(), "/test", "test-jwt", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "ghs_abc123" {
		t.Errorf("Token = %q, want %q", result.Token, "ghs_abc123")
	}
}
```

- [ ] **Step 3: Write failing test — API error response**

```go
func TestClient_Get_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result struct{}
	err := client.Get(context.Background(), "/test", "test-jwt", &result)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	want := "GitHub API error (404): Not Found"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
go test ./internal/github/ -v
```

Expected: FAIL — package doesn't exist yet.

- [ ] **Step 5: Implement `internal/github/client.go`**

```go
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) Get(ctx context.Context, path string, jwt string, result any) error {
	return c.do(ctx, http.MethodGet, path, jwt, result)
}

func (c *Client) Post(ctx context.Context, path string, jwt string, result any) error {
	return c.do(ctx, http.MethodPost, path, jwt, result)
}

func (c *Client) do(ctx context.Context, method string, path string, jwt string, result any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("GitHub API error (%d)", resp.StatusCode)
		}
		return fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, apiErr.Message)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./internal/github/ -v
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/github/
git commit -m "feat: GitHub API client with GET, POST, and error handling"
```

---

### Task 5: Repo Auto-Discovery

**Files:**
- Create: `internal/repo/repo.go`, `internal/repo/repo_test.go`

- [ ] **Step 1: Write failing tests — URL parsing**

```go
package repo

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "HTTPS URL",
			url:  "https://github.com/owner/repo.git",
			want: "owner/repo",
		},
		{
			name: "HTTPS URL without .git",
			url:  "https://github.com/owner/repo",
			want: "owner/repo",
		},
		{
			name: "SSH URL",
			url:  "git@github.com:owner/repo.git",
			want: "owner/repo",
		},
		{
			name: "SSH URL without .git",
			url:  "git@github.com:owner/repo",
			want: "owner/repo",
		},
		{
			name:    "non-GitHub URL",
			url:     "https://gitlab.com/owner/repo.git",
			wantErr: true,
		},
		{
			name:    "non-GitHub SSH URL",
			url:     "git@gitlab.com:owner/repo.git",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRemoteURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/repo/ -run TestParseRemoteURL -v
```

Expected: FAIL — `ParseRemoteURL` not defined.

- [ ] **Step 3: Implement URL parsing in `internal/repo/repo.go`**

```go
package repo

import (
	"fmt"
	"os/exec"
	"strings"
)

func ParseRemoteURL(rawURL string) (string, error) {
	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@github.com:") {
		path := strings.TrimPrefix(rawURL, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		return path, nil
	}

	// HTTPS format: https://github.com/owner/repo.git
	if strings.HasPrefix(rawURL, "https://github.com/") {
		path := strings.TrimPrefix(rawURL, "https://github.com/")
		path = strings.TrimSuffix(path, ".git")
		return path, nil
	}

	return "", fmt.Errorf("remote is not a GitHub URL: %s; use --repo to specify", rawURL)
}
```

- [ ] **Step 4: Run URL parsing tests**

```bash
go test ./internal/repo/ -run TestParseRemoteURL -v
```

Expected: all PASS.

- [ ] **Step 5: Write failing tests — remote selection logic**

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T, remotes map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	for name, url := range remotes {
		run("remote", "add", name, url)
	}
	return dir
}

func TestDiscover_SingleRemote(t *testing.T) {
	dir := initTestRepo(t, map[string]string{
		"upstream": "https://github.com/owner/repo.git",
	})

	got, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "owner/repo" {
		t.Errorf("got %q, want %q", got, "owner/repo")
	}
}

func TestDiscover_MultipleRemotes_PrefersOrigin(t *testing.T) {
	dir := initTestRepo(t, map[string]string{
		"upstream": "https://github.com/other/repo.git",
		"origin":   "https://github.com/owner/repo.git",
	})

	got, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "owner/repo" {
		t.Errorf("got %q, want %q", got, "owner/repo")
	}
}

func TestDiscover_MultipleRemotes_NoOrigin(t *testing.T) {
	dir := initTestRepo(t, map[string]string{
		"upstream": "https://github.com/other/repo.git",
		"fork":     "https://github.com/owner/repo.git",
	})

	_, err := Discover(dir)
	if err == nil {
		t.Fatal("expected error when multiple remotes and no origin")
	}
}

func TestDiscover_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := Discover(dir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}
```

- [ ] **Step 6: Run remote selection tests to verify they fail**

```bash
go test ./internal/repo/ -run TestDiscover -v
```

Expected: FAIL — `Discover` not defined.

- [ ] **Step 7: Implement `Discover` in `internal/repo/repo.go`**

Add to the existing `repo.go`:

```go
func Discover(dir string) (string, error) {
	remotes, err := listRemotes(dir)
	if err != nil {
		return "", fmt.Errorf("not in a git repo; use --repo or set GH_APP_INSTALLATION_ID")
	}

	if len(remotes) == 0 {
		return "", fmt.Errorf("no git remotes found; use --repo or set GH_APP_INSTALLATION_ID")
	}

	var remoteName string
	if len(remotes) == 1 {
		for name := range remotes {
			remoteName = name
		}
	} else if url, ok := remotes["origin"]; ok {
		_ = url
		remoteName = "origin"
	} else {
		return "", fmt.Errorf("multiple remotes found and none is named 'origin'; use --repo to specify")
	}

	return ParseRemoteURL(remotes[remoteName])
}

func listRemotes(dir string) (map[string]string, error) {
	// Get remote names
	cmd := exec.Command("git", "remote")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	names := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(names) == 1 && names[0] == "" {
		return nil, fmt.Errorf("no remotes")
	}

	remotes := make(map[string]string, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		cmd := exec.Command("git", "remote", "get-url", name)
		cmd.Dir = dir
		urlOut, err := cmd.Output()
		if err != nil {
			continue
		}
		remotes[name] = strings.TrimSpace(string(urlOut))
	}
	return remotes, nil
}
```

- [ ] **Step 8: Run all repo tests**

```bash
go test ./internal/repo/ -v
```

Expected: all PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/repo/
git commit -m "feat: git remote discovery and GitHub URL parsing"
```

---

### Task 6: Installation Token Minting

**Files:**
- Create: `internal/auth/token.go`, `internal/auth/token_test.go`

- [ ] **Step 1: Write failing test — resolve installation ID from repo**

```go
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/middlendian/gh-app-auth/internal/github"
)

func TestGetInstallationID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/installation" {
			t.Errorf("path = %q, want /repos/owner/repo/installation", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"id": 67890})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	id, err := GetInstallationID(context.Background(), client, "test-jwt", "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 67890 {
		t.Errorf("id = %d, want 67890", id)
	}
}
```

- [ ] **Step 2: Write failing test — mint installation token**

```go
func TestMintToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/installations/67890/access_tokens" {
			t.Errorf("path = %q, want /app/installations/67890/access_tokens", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"token": "ghs_abc123"})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	token, err := MintToken(context.Background(), client, "test-jwt", 67890)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "ghs_abc123" {
		t.Errorf("token = %q, want %q", token, "ghs_abc123")
	}
}
```

- [ ] **Step 3: Write failing test — app not installed error**

```go
func TestGetInstallationID_NotInstalled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	_, err := GetInstallationID(context.Background(), client, "test-jwt", "owner/repo")
	if err == nil {
		t.Fatal("expected error for app not installed")
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
go test ./internal/auth/ -run "TestGetInstallationID|TestMintToken" -v
```

Expected: FAIL — functions not defined.

- [ ] **Step 5: Implement `internal/auth/token.go`**

```go
package auth

import (
	"context"
	"fmt"

	"github.com/middlendian/gh-app-auth/internal/github"
)

func GetInstallationID(ctx context.Context, client *github.Client, jwt string, repo string) (int64, error) {
	var result struct {
		ID int64 `json:"id"`
	}
	path := fmt.Sprintf("/repos/%s/installation", repo)
	if err := client.Get(ctx, path, jwt, &result); err != nil {
		return 0, fmt.Errorf("GitHub App is not installed on %s: %w", repo, err)
	}
	return result.ID, nil
}

func MintToken(ctx context.Context, client *github.Client, jwt string, installationID int64) (string, error) {
	var result struct {
		Token string `json:"token"`
	}
	path := fmt.Sprintf("/app/installations/%d/access_tokens", installationID)
	if err := client.Post(ctx, path, jwt, &result); err != nil {
		return "", fmt.Errorf("minting installation token: %w", err)
	}
	return result.Token, nil
}
```

- [ ] **Step 6: Run all auth tests**

```bash
go test ./internal/auth/ -v
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/auth/token.go internal/auth/token_test.go
git commit -m "feat: installation ID resolution and token minting"
```

---

### Task 7: Token Subcommand

**Files:**
- Create: `cmd/token.go`

- [ ] **Step 1: Implement `cmd/token.go`**

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/middlendian/gh-app-auth/internal/auth"
	"github.com/middlendian/gh-app-auth/internal/config"
	"github.com/middlendian/gh-app-auth/internal/github"
	repopkg "github.com/middlendian/gh-app-auth/internal/repo"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print an installation access token to stdout",
	RunE:  runToken,
}

func init() {
	rootCmd.AddCommand(tokenCmd)
}

func runToken(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	jwtToken, err := auth.GenerateJWT(cfg.AppID, cfg.PrivateKey)
	if err != nil {
		return fmt.Errorf("generating JWT: %w", err)
	}

	client := github.NewClient("https://api.github.com")

	installationID, err := resolveInstallationID(ctx, client, jwtToken, cfg)
	if err != nil {
		return err
	}

	token, err := auth.MintToken(ctx, client, jwtToken, installationID)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, token)
	return nil
}

func resolveInstallationID(ctx context.Context, client *github.Client, jwt string, cfg *config.Config) (int64, error) {
	if cfg.InstallationID != "" {
		id, err := strconv.ParseInt(cfg.InstallationID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid GH_APP_INSTALLATION_ID: %w", err)
		}
		return id, nil
	}

	repoSlug, err := resolveRepo()
	if err != nil {
		return 0, err
	}

	return auth.GetInstallationID(ctx, client, jwt, repoSlug)
}

func resolveRepo() (string, error) {
	if repo != "" {
		return repo, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	return repopkg.Discover(cwd)
}
```

**Note:** The import alias `repopkg` avoids collision with the `repo` variable (the `--repo` flag).

- [ ] **Step 2: Verify it compiles**

```bash
go build -o gh-app-auth .
```

Expected: builds successfully.

- [ ] **Step 3: Verify help output includes token subcommand**

```bash
./gh-app-auth --help
```

Expected: output includes `token` in the available commands list.

- [ ] **Step 4: Verify token subcommand help**

```bash
./gh-app-auth token --help
```

Expected: shows "Print an installation access token to stdout" and the inherited `--repo` flag.

- [ ] **Step 5: Commit**

```bash
git add cmd/token.go
git commit -m "feat: token subcommand — orchestrates full auth flow"
```

---

### Task 8: Completion Subcommand

**Files:**
- Create: `cmd/completion.go`

- [ ] **Step 1: Implement `cmd/completion.go`**

```go
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for gh app-auth.

To load completions:

Bash:
  $ source <(gh app-auth completion bash)

Zsh:
  $ gh app-auth completion zsh > "${fpath[1]}/_gh-app-auth"

Fish:
  $ gh app-auth completion fish | source

PowerShell:
  PS> gh app-auth completion powershell | Out-String | Invoke-Expression
`,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build -o gh-app-auth .
```

- [ ] **Step 3: Verify completion output**

```bash
./gh-app-auth completion bash | head -5
```

Expected: outputs bash completion script (starts with a comment or function definition).

- [ ] **Step 4: Verify help lists completion subcommand**

```bash
./gh-app-auth --help
```

Expected: both `token` and `completion` appear in available commands.

- [ ] **Step 5: Commit**

```bash
git add cmd/completion.go
git commit -m "feat: shell completion subcommand"
```

---

### Task 9: Build, Test, and Polish

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 2: Run `go vet`**

```bash
go vet ./...
```

Expected: no issues.

- [ ] **Step 3: Verify final binary build**

```bash
go build -o gh-app-auth .
```

- [ ] **Step 4: Verify error message when env vars are missing**

```bash
GH_APP_ID="" ./gh-app-auth token
```

Expected: error message mentioning `GH_APP_ID`.

- [ ] **Step 5: Update `README.md`**

```markdown
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
```

- [ ] **Step 6: Commit**

```bash
git add README.md
git commit -m "docs: update README with install, usage, and configuration"
```

- [ ] **Step 7: Run full test suite one final time**

```bash
go test ./... -v
```

Expected: all tests pass. Implementation complete.

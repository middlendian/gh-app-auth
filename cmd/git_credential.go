package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/middlendian/gh-app-auth/internal/auth"
	"github.com/middlendian/gh-app-auth/internal/config"
	"github.com/middlendian/gh-app-auth/internal/github"
	"github.com/spf13/cobra"
)

var gitCredentialCmd = &cobra.Command{
	Use:   "git-credential <get|store|erase>",
	Short: "Act as a Git credential helper",
	Long: `Implements the Git credential helper protocol.

Configure as a credential helper in your gitconfig:

  [credential "https://github.com"]
    helper = !gh-app-auth git-credential

Git will call this command with "get" when it needs credentials.
The helper reads the request from stdin, resolves the repository
from the URL path, and returns an installation access token.`,
	Args: cobra.ExactArgs(1),
	RunE: runGitCredential,
}

func init() {
	rootCmd.AddCommand(gitCredentialCmd)
}

// credentialRequest holds the key=value pairs Git sends on stdin.
type credentialRequest struct {
	Protocol string
	Host     string
	Path     string
}

func parseCredentialInput(cmd *cobra.Command) (*credentialRequest, error) {
	req := &credentialRequest{}
	scanner := bufio.NewScanner(cmd.InOrStdin())
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "protocol":
			req.Protocol = value
		case "host":
			req.Host = value
		case "path":
			req.Path = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading credential input: %w", err)
	}
	return req, nil
}

func runGitCredential(cmd *cobra.Command, args []string) error {
	operation := args[0]

	// store and erase are no-ops; we don't cache tokens.
	if operation != "get" {
		return nil
	}

	req, err := parseCredentialInput(cmd)
	if err != nil {
		return err
	}

	if req.Protocol != "https" {
		return fmt.Errorf("unsupported protocol %q; only https is supported", req.Protocol)
	}
	if req.Host != "github.com" {
		return fmt.Errorf("unsupported host %q; only github.com is supported", req.Host)
	}

	// Determine the target repo: --repo flag > stdin path > git discovery.
	repoSlug, err := resolveRepoFromCredential(req)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	token, err := mintTokenForRepo(ctx, repoSlug)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "protocol=https\nhost=github.com\nusername=x-access-token\npassword=%s\n", token)
	return nil
}

// resolveRepoFromCredential determines the owner/repo slug. Priority:
// 1. --repo flag (set globally)
// 2. path from credential input (e.g. "owner/repo.git")
// 3. git remote discovery (fallback)
func resolveRepoFromCredential(req *credentialRequest) (string, error) {
	if repo != "" {
		return repo, nil
	}

	if req.Path != "" {
		slug := strings.TrimSuffix(req.Path, ".git")
		// The path may include extra segments (e.g. "owner/repo/info/refs");
		// take only the first two components.
		parts := strings.SplitN(slug, "/", 3)
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + "/" + parts[1], nil
		}
	}

	return resolveRepo()
}

func mintTokenForRepo(ctx context.Context, repoSlug string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	jwtToken, err := auth.GenerateJWT(cfg.AppID, cfg.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("generating JWT: %w", err)
	}

	client := github.NewClient("https://api.github.com")

	var installationID int64
	if cfg.InstallationID != "" {
		id, err := parseInstallationID(cfg.InstallationID)
		if err != nil {
			return "", err
		}
		installationID = id
	} else {
		installationID, err = auth.GetInstallationID(ctx, client, jwtToken, repoSlug)
		if err != nil {
			return "", err
		}
	}

	return auth.MintToken(ctx, client, jwtToken, installationID)
}

func parseInstallationID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid GH_APP_INSTALLATION_ID: %w", err)
	}
	return id, nil
}

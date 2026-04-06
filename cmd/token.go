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

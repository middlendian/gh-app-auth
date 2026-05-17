package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/middlendian/gh-app-auth/internal/auth"
	"github.com/middlendian/gh-app-auth/internal/config"
	"github.com/middlendian/gh-app-auth/internal/github"
	"github.com/spf13/cobra"
)

var installationsCmd = &cobra.Command{
	Use:   "installations",
	Short: "Inspect the App's installations across GitHub",
}

var installationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List every installation the GitHub App is on",
	Long: `List every installation the GitHub App is on.

Mints an App-level JWT, calls GET /app/installations (paginated), and
prints a JSON array to stdout. Each element is slimmed to the fields
callers actually use: id, account.login, account.type.

The output is jq-friendly:

  gh-app-auth installations list | jq '.[] | select(.account.type == "Organization") | .id'`,
	RunE: runInstallationsList,
}

func init() {
	rootCmd.AddCommand(installationsCmd)
	installationsCmd.AddCommand(installationsListCmd)
}

func runInstallationsList(cmd *cobra.Command, args []string) error {
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

	installations, err := auth.ListInstallations(ctx, client, jwtToken)
	if err != nil {
		return err
	}

	// Guarantee `[]` rather than `null` when there are zero installations —
	// downstream jq pipelines hate null arrays.
	if installations == nil {
		installations = []auth.Installation{}
	}

	out, err := json.MarshalIndent(installations, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding installations: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, string(out))
	return nil
}

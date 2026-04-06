package cmd

import (
	"github.com/spf13/cobra"
)

var repo string

var rootCmd = &cobra.Command{
	Use:           "gh-app-auth",
	Short:         "GitHub App authentication for gh CLI",
	Long:          "Generate GitHub App installation access tokens for use with gh CLI and other tools.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&repo, "repo", "", "target repository in owner/repo format")
}

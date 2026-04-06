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

package auth

import (
	"context"
	"fmt"

	"github.com/middlendian/gh-app-auth/internal/github"
)

// Installation is the slim, caller-facing view of a GitHub App installation.
// Only the fields downstream consumers use are exposed — keeping the contract
// small insulates callers from churn in GitHub's full installation schema.
type Installation struct {
	ID      int64   `json:"id"`
	Account Account `json:"account"`
}

type Account struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// ListInstallations fetches every installation the App is on by paginating
// through GET /app/installations following Link: rel="next".
func ListInstallations(ctx context.Context, client *github.Client, jwt string) ([]Installation, error) {
	var all []Installation
	next := "/app/installations?per_page=100"
	for next != "" {
		var page []Installation
		nextURL, err := client.GetPaginated(ctx, next, jwt, &page)
		if err != nil {
			return nil, fmt.Errorf("listing installations: %w", err)
		}
		all = append(all, page...)
		next = nextURL
	}
	return all, nil
}

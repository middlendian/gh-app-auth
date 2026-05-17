package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Get(ctx context.Context, path string, jwt string, result any) error {
	_, err := c.do(ctx, http.MethodGet, path, jwt, result)
	return err
}

func (c *Client) Post(ctx context.Context, path string, jwt string, result any) error {
	_, err := c.do(ctx, http.MethodPost, path, jwt, result)
	return err
}

// GetPaginated performs a GET against urlOrPath and returns the "next" URL
// from the response's Link header (empty string if there is no next page).
// urlOrPath may be a path beginning with "/" — which is joined with the
// client's base URL — or a fully qualified URL (as found in a Link header).
func (c *Client) GetPaginated(ctx context.Context, urlOrPath string, jwt string, result any) (string, error) {
	return c.do(ctx, http.MethodGet, urlOrPath, jwt, result)
}

func (c *Client) do(ctx context.Context, method string, urlOrPath string, jwt string, result any) (string, error) {
	url := urlOrPath
	if strings.HasPrefix(url, "/") {
		url = c.baseURL + url
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return "", fmt.Errorf("GitHub API error (%d)", resp.StatusCode)
		}
		return "", fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, apiErr.Message)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	return parseNextLink(resp.Header.Get("Link")), nil
}

// parseNextLink extracts the URL whose rel="next" from a GitHub Link header.
// Returns "" when the header is empty or contains no "next" relation.
//
// GitHub format: <https://api.github.com/...?page=2>; rel="next", <...>; rel="last"
func parseNextLink(header string) string {
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ",") {
		segments := strings.Split(strings.TrimSpace(part), ";")
		if len(segments) < 2 {
			continue
		}
		urlPart := strings.TrimSpace(segments[0])
		if !strings.HasPrefix(urlPart, "<") || !strings.HasSuffix(urlPart, ">") {
			continue
		}
		for _, attr := range segments[1:] {
			if strings.TrimSpace(attr) == `rel="next"` {
				return urlPart[1 : len(urlPart)-1]
			}
		}
	}
	return ""
}

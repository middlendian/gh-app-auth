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
	} else if _, ok := remotes["origin"]; ok {
		remoteName = "origin"
	} else {
		return "", fmt.Errorf("multiple remotes found and none is named 'origin'; use --repo to specify")
	}

	return ParseRemoteURL(remotes[remoteName])
}

func listRemotes(dir string) (map[string]string, error) {
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

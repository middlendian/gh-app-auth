package repo

import (
	"os"
	"os/exec"
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

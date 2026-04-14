package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseCredentialInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		protocol string
		host     string
		path     string
	}{
		{
			name:     "standard get request",
			input:    "protocol=https\nhost=github.com\npath=owner/repo.git\n\n",
			protocol: "https",
			host:     "github.com",
			path:     "owner/repo.git",
		},
		{
			name:     "no path",
			input:    "protocol=https\nhost=github.com\n\n",
			protocol: "https",
			host:     "github.com",
			path:     "",
		},
		{
			name:     "empty input",
			input:    "\n",
			protocol: "",
			host:     "",
			path:     "",
		},
		{
			name:     "unknown keys ignored",
			input:    "protocol=https\nhost=github.com\nwwwauth=basic\n\n",
			protocol: "https",
			host:     "github.com",
			path:     "",
		},
		{
			name:     "lines without equals ignored",
			input:    "protocol=https\nbadline\nhost=github.com\n\n",
			protocol: "https",
			host:     "github.com",
			path:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(tt.input))

			req, err := parseCredentialInput(cmd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Protocol != tt.protocol {
				t.Errorf("protocol: got %q, want %q", req.Protocol, tt.protocol)
			}
			if req.Host != tt.host {
				t.Errorf("host: got %q, want %q", req.Host, tt.host)
			}
			if req.Path != tt.path {
				t.Errorf("path: got %q, want %q", req.Path, tt.path)
			}
		})
	}
}

func TestResolveRepoFromCredential(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "flag takes precedence",
			flag: "flagowner/flagrepo",
			path: "pathowner/pathrepo.git",
			want: "flagowner/flagrepo",
		},
		{
			name: "path with .git suffix",
			path: "owner/repo.git",
			want: "owner/repo",
		},
		{
			name: "path without .git suffix",
			path: "owner/repo",
			want: "owner/repo",
		},
		{
			name: "path with extra segments",
			path: "owner/repo/info/refs",
			want: "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := repo
			repo = tt.flag
			defer func() { repo = old }()

			req := &credentialRequest{Path: tt.path}
			got, err := resolveRepoFromCredential(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
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

func TestGitCredentialStoreAndErase(t *testing.T) {
	for _, op := range []string{"store", "erase"} {
		t.Run(op, func(t *testing.T) {
			err := runGitCredential(gitCredentialCmd, []string{op})
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", op, err)
			}
		})
	}
}

func TestGitCredentialUnsupportedProtocol(t *testing.T) {
	cmd := gitCredentialCmd
	cmd.SetIn(strings.NewReader("protocol=ssh\nhost=github.com\n\n"))
	cmd.SetArgs([]string{"get"})

	err := cmd.RunE(cmd, []string{"get"})
	if err == nil {
		t.Fatal("expected error for ssh protocol")
	}
	if !strings.Contains(err.Error(), "unsupported protocol") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitCredentialUnsupportedHost(t *testing.T) {
	cmd := gitCredentialCmd
	cmd.SetIn(strings.NewReader("protocol=https\nhost=gitlab.com\n\n"))
	cmd.SetArgs([]string{"get"})

	err := cmd.RunE(cmd, []string{"get"})
	if err == nil {
		t.Fatal("expected error for non-github host")
	}
	if !strings.Contains(err.Error(), "unsupported host") {
		t.Errorf("unexpected error: %v", err)
	}
}

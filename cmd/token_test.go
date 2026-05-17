package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/middlendian/gh-app-auth/internal/config"
	"github.com/middlendian/gh-app-auth/internal/github"
)

func TestRunToken_MutuallyExclusiveFlags(t *testing.T) {
	oldRepo := repo
	oldFlag := installationIDFlag
	repo = "owner/repo"
	installationIDFlag = 12345
	defer func() {
		repo = oldRepo
		installationIDFlag = oldFlag
	}()

	err := runToken(tokenCmd, nil)
	if err == nil {
		t.Fatal("expected error when both --installation-id and --repo are set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error %q does not mention mutual exclusion", err.Error())
	}
}

func TestResolveInstallationID_FlagWinsOverEnvAndRepo(t *testing.T) {
	oldFlag := installationIDFlag
	installationIDFlag = 999
	defer func() { installationIDFlag = oldFlag }()

	cfg := &config.Config{InstallationID: "1"}
	// client and jwt are unused on the flag path, but pass non-nil for safety.
	got, err := resolveInstallationID(context.Background(), github.NewClient("http://unused"), "jwt", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 999 {
		t.Errorf("got %d, want 999", got)
	}
}

func TestResolveInstallationID_EnvFallback(t *testing.T) {
	oldFlag := installationIDFlag
	installationIDFlag = 0
	defer func() { installationIDFlag = oldFlag }()

	cfg := &config.Config{InstallationID: "42"}
	got, err := resolveInstallationID(context.Background(), github.NewClient("http://unused"), "jwt", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestResolveInstallationID_InvalidEnv(t *testing.T) {
	oldFlag := installationIDFlag
	installationIDFlag = 0
	defer func() { installationIDFlag = oldFlag }()

	cfg := &config.Config{InstallationID: "not-a-number"}
	_, err := resolveInstallationID(context.Background(), github.NewClient("http://unused"), "jwt", cfg)
	if err == nil {
		t.Fatal("expected error for non-numeric env")
	}
}

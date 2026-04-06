package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func generateTestPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block))
}

func TestLoad_MissingAppID(t *testing.T) {
	t.Setenv("GH_APP_ID", "")
	t.Setenv("GH_APP_PRIVATE_KEY", "")
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing GH_APP_ID")
	}
}

func TestLoad_NonNumericAppID(t *testing.T) {
	t.Setenv("GH_APP_ID", "not-a-number")
	t.Setenv("GH_APP_PRIVATE_KEY", "")
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric GH_APP_ID")
	}
}

func TestLoad_MissingPrivateKey(t *testing.T) {
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", "")
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing private key")
	}
}

func TestLoad_InlinePrivateKey(t *testing.T) {
	pemStr := generateTestPEM(t)
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", pemStr)
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppID != "12345" {
		t.Errorf("AppID = %q, want %q", cfg.AppID, "12345")
	}
	if cfg.PrivateKey == nil {
		t.Fatal("PrivateKey is nil")
	}
}

func TestLoad_PrivateKeyFile(t *testing.T) {
	pemStr := generateTestPEM(t)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(keyPath, []byte(pemStr), 0600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", "")
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", keyPath)
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PrivateKey == nil {
		t.Fatal("PrivateKey is nil")
	}
}

func TestLoad_InlineTakesPrecedenceOverFile(t *testing.T) {
	pemStr := generateTestPEM(t)
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", pemStr)
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "/nonexistent/path.pem")
	t.Setenv("GH_APP_INSTALLATION_ID", "")

	_, err := Load()
	if err != nil {
		t.Fatalf("should succeed with inline key even if file path is bogus: %v", err)
	}
}

func TestLoad_InstallationID(t *testing.T) {
	pemStr := generateTestPEM(t)
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_APP_PRIVATE_KEY", pemStr)
	t.Setenv("GH_APP_PRIVATE_KEY_FILE", "")
	t.Setenv("GH_APP_INSTALLATION_ID", "67890")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InstallationID != "67890" {
		t.Errorf("InstallationID = %q, want %q", cfg.InstallationID, "67890")
	}
}

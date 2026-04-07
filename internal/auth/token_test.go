package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/middlendian/gh-app-auth/internal/github"
)

func TestGetInstallationID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/installation" {
			t.Errorf("path = %q, want /repos/owner/repo/installation", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 67890})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	id, err := GetInstallationID(context.Background(), client, "test-jwt", "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 67890 {
		t.Errorf("id = %d, want 67890", id)
	}
}

func TestMintToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/installations/67890/access_tokens" {
			t.Errorf("path = %q, want /app/installations/67890/access_tokens", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "ghs_abc123"})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	token, err := MintToken(context.Background(), client, "test-jwt", 67890)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "ghs_abc123" {
		t.Errorf("token = %q, want %q", token, "ghs_abc123")
	}
}

func TestGetInstallationID_NotInstalled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	_, err := GetInstallationID(context.Background(), client, "test-jwt", "owner/repo")
	if err == nil {
		t.Fatal("expected error for app not installed")
	}
}

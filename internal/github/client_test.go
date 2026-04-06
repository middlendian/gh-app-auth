package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-jwt" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-jwt")
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept = %q, want %q", got, "application/vnd.github+json")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result struct {
		ID int `json:"id"`
	}
	err := client.Get(context.Background(), "/test", "test-jwt", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != 42 {
		t.Errorf("ID = %d, want 42", result.ID)
	}
}

func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"token": "ghs_abc123"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result struct {
		Token string `json:"token"`
	}
	err := client.Post(context.Background(), "/test", "test-jwt", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "ghs_abc123" {
		t.Errorf("Token = %q, want %q", result.Token, "ghs_abc123")
	}
}

func TestClient_Get_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result struct{}
	err := client.Get(context.Background(), "/test", "test-jwt", &result)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	want := "GitHub API error (404): Not Found"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

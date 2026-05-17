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
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42})
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
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "ghs_abc123"})
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

func TestClient_GetPaginated_ReturnsNextLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://api.example.com/foo?page=2>; rel="next", <https://api.example.com/foo?page=5>; rel="last"`)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1}})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result []map[string]any
	next, err := client.GetPaginated(context.Background(), "/foo", "test-jwt", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "https://api.example.com/foo?page=2"; next != want {
		t.Errorf("next = %q, want %q", next, want)
	}
	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1", len(result))
	}
}

func TestClient_GetPaginated_NoLinkHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var result []map[string]any
	next, err := client.GetPaginated(context.Background(), "/foo", "test-jwt", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != "" {
		t.Errorf("next = %q, want empty", next)
	}
}

func TestClient_GetPaginated_AbsoluteURL(t *testing.T) {
	var gotURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer server.Close()

	client := NewClient("https://api.example.com")
	var result []map[string]any
	_, err := client.GetPaginated(context.Background(), server.URL+"/abs?page=3", "test-jwt", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "/abs?page=3"; gotURL != want {
		t.Errorf("server saw URL %q, want %q", gotURL, want)
	}
}

func TestParseNextLink(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "empty",
			header: "",
			want:   "",
		},
		{
			name:   "next and last",
			header: `<https://api.github.com/foo?page=2>; rel="next", <https://api.github.com/foo?page=5>; rel="last"`,
			want:   "https://api.github.com/foo?page=2",
		},
		{
			name:   "only last (no next)",
			header: `<https://api.github.com/foo?page=1>; rel="first", <https://api.github.com/foo?page=5>; rel="last"`,
			want:   "",
		},
		{
			name:   "next listed second",
			header: `<https://api.github.com/foo?page=1>; rel="prev", <https://api.github.com/foo?page=3>; rel="next"`,
			want:   "https://api.github.com/foo?page=3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNextLink(tt.header)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_Get_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
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

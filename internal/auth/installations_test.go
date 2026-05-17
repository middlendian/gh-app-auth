package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/middlendian/gh-app-auth/internal/github"
)

func TestListInstallations_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/app/installations") {
			t.Errorf("path = %q, want prefix /app/installations", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": 111,
				"account": map[string]any{
					"login": "alice",
					"type":  "User",
				},
				"extra_field_we_drop": "ignored",
			},
			{
				"id": 222,
				"account": map[string]any{
					"login": "acmecorp",
					"type":  "Organization",
				},
			},
		})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	got, err := ListInstallations(context.Background(), client, "test-jwt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []Installation{
		{ID: 111, Account: Account{Login: "alice", Type: "User"}},
		{ID: 222, Account: Account{Login: "acmecorp", Type: "Organization"}},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestListInstallations_FollowsPagination(t *testing.T) {
	var pagesServed int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch page {
		case "", "1":
			pagesServed++
			w.Header().Set("Link", fmt.Sprintf(`<%s/app/installations?page=2>; rel="next", <%s/app/installations?page=3>; rel="last"`, server.URL, server.URL))
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "account": map[string]any{"login": "one", "type": "User"}},
			})
		case "2":
			pagesServed++
			w.Header().Set("Link", fmt.Sprintf(`<%s/app/installations?page=3>; rel="next", <%s/app/installations?page=3>; rel="last"`, server.URL, server.URL))
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 2, "account": map[string]any{"login": "two", "type": "Organization"}},
			})
		case "3":
			pagesServed++
			// No Link header (or no rel=next) — end of pagination.
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 3, "account": map[string]any{"login": "three", "type": "Organization"}},
			})
		default:
			t.Errorf("unexpected page %q", page)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	got, err := ListInstallations(context.Background(), client, "test-jwt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pagesServed != 3 {
		t.Errorf("pagesServed = %d, want 3", pagesServed)
	}
	wantIDs := []int64{1, 2, 3}
	if len(got) != len(wantIDs) {
		t.Fatalf("len = %d, want %d", len(got), len(wantIDs))
	}
	for i, want := range wantIDs {
		if got[i].ID != want {
			t.Errorf("[%d].ID = %d, want %d", i, got[i].ID, want)
		}
	}
}

func TestListInstallations_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Bad credentials"})
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	_, err := ListInstallations(context.Background(), client, "test-jwt")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q does not mention status 401", err.Error())
	}
}

func TestListInstallations_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := github.NewClient(server.URL)
	_, err := ListInstallations(context.Background(), client, "test-jwt")
	if err == nil {
		t.Fatal("expected error")
	}
}

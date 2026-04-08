package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestTranslateVocationID(t *testing.T) {
	tests := []struct {
		name       string
		rubinotID  string
		expectedID string
	}{
		{"all vocations", "0", "0"},
		{"none", "1", "0"},
		{"sorcerer", "2", "4"},
		{"druid", "3", "2"},
		{"paladin", "4", "3"},
		{"knight", "5", "1"},
		{"monk", "9", "5"},
		{"unknown passes through", "99", "99"},
		{"empty passes through", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateVocationID(tt.rubinotID)
			if got != tt.expectedID {
				t.Fatalf("translateVocationID(%q) = %q, want %q", tt.rubinotID, got, tt.expectedID)
			}
		})
	}
}

func TestTranslatePath(t *testing.T) {
	UpdateWorldMappings([]WorldMapping{
		{ID: 1, Name: "Elysian"},
		{ID: 11, Name: "Auroria"},
		{ID: 15, Name: "Belaria"},
	})

	tests := []struct {
		name        string
		upstreamURL string
		wantPath    string
		wantErr     bool
	}{
		{
			"worlds list",
			"https://rubinot.com.br/api/worlds",
			"/v1/worlds",
			false,
		},
		{
			"single world",
			"https://rubinot.com.br/api/worlds/Auroria",
			"/v1/world/Auroria",
			false,
		},
		{
			"character search",
			"https://rubinot.com.br/api/characters/search?name=Bubble",
			"/v1/characters/Bubble",
			false,
		},
		{
			"character search with spaces",
			"https://rubinot.com.br/api/characters/search?name=Bubble+Gum",
			"/v1/characters/Bubble%20Gum",
			false,
		},
		{
			"character search url encoded",
			"https://rubinot.com.br/api/characters/search?name=Bubble%20Gum",
			"/v1/characters/Bubble%20Gum",
			false,
		},
		{
			"guilds list with world and page",
			"https://rubinot.com.br/api/guilds?world=11&page=2",
			"/v1/guilds/Auroria?page=2",
			false,
		},
		{
			"guilds list with world only",
			"https://rubinot.com.br/api/guilds?world=15",
			"/v1/guilds/Belaria",
			false,
		},
		{
			"guilds list unknown world id passes through",
			"https://rubinot.com.br/api/guilds?world=999",
			"/v1/guilds/999",
			false,
		},
		{
			"single guild",
			"https://rubinot.com.br/api/guilds/Red Rose",
			"/v1/guild/Red%20Rose",
			false,
		},
		{
			"highscores categories",
			"https://rubinot.com.br/api/highscores/categories",
			"/v1/highscores/categories",
			false,
		},
		{
			"highscores with params",
			"https://rubinot.com.br/api/highscores?world=11&category=experience&vocation=2",
			"/v1/highscores?category=experience&vocation=4&world=Auroria",
			false,
		},
		{
			"highscores unknown world passes through",
			"https://rubinot.com.br/api/highscores?world=999&category=experience&vocation=5",
			"/v1/highscores?category=experience&vocation=1&world=999",
			false,
		},
		{
			"killstats",
			"https://rubinot.com.br/api/killstats?world=1",
			"/v1/killstatistics/Elysian",
			false,
		},
		{
			"killstats unknown world",
			"https://rubinot.com.br/api/killstats?world=999",
			"/v1/killstatistics/999",
			false,
		},
		{
			"deaths with world and page",
			"https://rubinot.com.br/api/deaths?world=11&page=3",
			"/v1/deaths/Auroria?page=3",
			false,
		},
		{
			"deaths with world only",
			"https://rubinot.com.br/api/deaths?world=11",
			"/v1/deaths/Auroria",
			false,
		},
		{
			"bans with world and page",
			"https://rubinot.com.br/api/bans?world=15&page=2",
			"/v1/banishments/Belaria?page=2",
			false,
		},
		{
			"bans with world only",
			"https://rubinot.com.br/api/bans?world=15",
			"/v1/banishments/Belaria",
			false,
		},
		{
			"transfers with page",
			"https://rubinot.com.br/api/transfers?page=5",
			"/v1/transfers?page=5",
			false,
		},
		{
			"transfers without page",
			"https://rubinot.com.br/api/transfers",
			"/v1/transfers",
			false,
		},
		{
			"boosted",
			"https://rubinot.com.br/api/boosted",
			"/v1/boosted",
			false,
		},
		{
			"unrecognized path errors",
			"https://rubinot.com.br/api/unknown/endpoint",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := translatePath(tt.upstreamURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("translatePath(%q) expected error, got nil", tt.upstreamURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("translatePath(%q) unexpected error: %v", tt.upstreamURL, err)
			}
			if got != tt.wantPath {
				t.Fatalf("translatePath(%q) = %q, want %q", tt.upstreamURL, got, tt.wantPath)
			}
		})
	}
}

func TestTranslatePathEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		upstreamURL string
		wantErr     bool
	}{
		{"empty url", "", true},
		{"just path /api/worlds", "/api/worlds", false},
		{"character search missing name", "https://rubinot.com.br/api/characters/search", true},
		{"guilds missing world param", "https://rubinot.com.br/api/guilds", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := translatePath(tt.upstreamURL)
			if tt.wantErr && err == nil {
				t.Fatalf("translatePath(%q) expected error, got nil", tt.upstreamURL)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("translatePath(%q) unexpected error: %v", tt.upstreamURL, err)
			}
		})
	}
}

func TestIsRubinidataProvider(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{"empty env", "", false},
		{"rubinot value", "rubinot", false},
		{"rubinidata lowercase", "rubinidata", true},
		{"rubinidata uppercase", "RUBINIDATA", true},
		{"rubinidata mixed case", "RubiniData", true},
		{"other value", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("UPSTREAM_PROVIDER", tt.envValue)
			if got := IsRubinidataProvider(); got != tt.want {
				t.Fatalf("IsRubinidataProvider() = %v, want %v (env=%q)", got, tt.want, tt.envValue)
			}
		})
	}
}

func TestRubinidataClientFetchSuccess(t *testing.T) {
	UpdateWorldMappings([]WorldMapping{{ID: 11, Name: "Auroria"}})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Fatalf("expected X-API-Key header, got %q", r.Header.Get("X-API-Key"))
		}
		if r.URL.Path != "/v1/worlds" {
			t.Fatalf("expected /v1/worlds, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"worlds":[]}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "test-key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != `{"worlds":[]}` {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestRubinidataClientFetchSendsAPIKeyHeader(t *testing.T) {
	var receivedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "my-secret-key")
	_, _ = client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")

	if receivedKey != "my-secret-key" {
		t.Fatalf("expected API key %q, got %q", "my-secret-key", receivedKey)
	}
}

func TestRubinidataClientFetchRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "server error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", body)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRubinidataClientFetchExhaustedRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "server error")
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	_, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRubinidataClientFetchInvalidPath(t *testing.T) {
	client := NewRubinidataClient("http://localhost", "key")
	_, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/unknown/xyz")
	if err == nil {
		t.Fatal("expected error for unrecognized path")
	}
}

func TestRubinidataClientFetchCallsAdaptResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"raw":"data"}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != `{"raw":"data"}` {
		t.Fatalf("expected adaptResponse to pass through body, got %q", body)
	}
}

func TestRubinidataClientFetchContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Fetch(ctx, "https://rubinot.com.br/api/worlds")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRubinidataClientFetchNotFoundReturnsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error for 404: %v", err)
	}
	if body != `{"error":"not found"}` {
		t.Fatalf("expected 404 body to be returned, got %q", body)
	}
}

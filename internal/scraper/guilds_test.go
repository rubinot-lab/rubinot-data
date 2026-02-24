package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseGuildsHTMLListSynthetic(t *testing.T) {
	// FIXTURE: synthetic, must be replaced with real capture
	html := readFixture(t, "guilds", "list.html")
	result, err := parseGuildsHTML("Belaria", html)
	if err != nil {
		t.Fatalf("expected guilds list fixture to parse, got error: %v", err)
	}
	if len(result.Active) != 2 {
		t.Fatalf("expected 2 active guilds, got %d", len(result.Active))
	}
	if len(result.Formation) != 1 {
		t.Fatalf("expected 1 formation guild, got %d", len(result.Formation))
	}
}

func TestParseGuildsHTMLEmptyFixture(t *testing.T) {
	html := readFixture(t, "guilds", "empty.html")
	result, err := parseGuildsHTML("Belaria", html)
	if err != nil {
		t.Fatalf("expected guilds empty fixture to parse, got error: %v", err)
	}
	if len(result.Active) != 0 || len(result.Formation) != 0 {
		t.Fatalf("expected empty guild lists, got active=%d formation=%d", len(result.Active), len(result.Formation))
	}
}

func TestFetchGuildsHappy(t *testing.T) {
	guildsHTML := readFixture(t, "guilds", "list.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(
			w,
			`{"status":"ok","message":"","solution":{"response":%q,"status":200,"url":"https://www.rubinot.com.br/?subtopic=guilds&world=15"}}`,
			guildsHTML,
		)
	}))
	defer server.Close()

	result, sourceURL, err := FetchGuilds(context.Background(), "https://www.rubinot.com.br", "Belaria", 15, FetchOptions{
		FlareSolverrURL: server.URL,
		MaxTimeoutMs:    120000,
	})
	if err != nil {
		t.Fatalf("expected FetchGuilds to succeed, got error: %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/?subtopic=guilds" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Active) == 0 {
		t.Fatal("expected non-empty guild list")
	}
}

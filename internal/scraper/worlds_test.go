package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorldsHTMLOverviewFixture(t *testing.T) {
	html := readFixture(t, "worlds", "overview.html")
	result, err := parseWorldsHTML(html)
	if err != nil {
		t.Fatalf("expected parseWorldsHTML to succeed, got error: %v", err)
	}
	if result.TotalPlayersOnline <= 0 {
		t.Fatalf("expected total_players_online > 0, got %d", result.TotalPlayersOnline)
	}
	if len(result.Worlds) == 0 {
		t.Fatal("expected at least one world entry")
	}

	firstWorld := result.Worlds[0]
	if firstWorld.Name == "" || firstWorld.Location == "" || firstWorld.PVPType == "" {
		t.Fatalf("expected non-empty world fields, got %+v", firstWorld)
	}
}

func TestParseWorldsHTMLEmptyFixture(t *testing.T) {
	html := readFixture(t, "worlds", "empty.html")
	result, err := parseWorldsHTML(html)
	if err != nil {
		t.Fatalf("expected parseWorldsHTML to succeed, got error: %v", err)
	}
	if len(result.Worlds) != 0 {
		t.Fatalf("expected no worlds, got %d", len(result.Worlds))
	}
}

func TestFetchWorldsHappy(t *testing.T) {
	overviewHTML := readFixture(t, "worlds", "overview.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(
			w,
			`{"status":"ok","message":"","solution":{"response":%q,"status":200,"url":"https://www.rubinot.com.br/?subtopic=worlds"}}`,
			overviewHTML,
		)
	}))
	defer server.Close()

	result, sourceURL, err := FetchWorlds(context.Background(), "https://www.rubinot.com.br", FetchOptions{
		FlareSolverrURL: server.URL,
		MaxTimeoutMs:    120000,
	})
	if err != nil {
		t.Fatalf("expected FetchWorlds to succeed, got error: %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/?subtopic=worlds" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Worlds) == 0 {
		t.Fatal("expected non-empty worlds list")
	}
}

func readFixture(t *testing.T, dir, name string) string {
	t.Helper()

	filePath := filepath.Join("..", "..", "testdata", dir, name)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", filePath, err)
	}

	return string(data)
}

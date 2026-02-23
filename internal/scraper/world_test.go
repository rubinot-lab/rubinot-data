package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestParseWorldHTMLBelariaFixture(t *testing.T) {
	html := readFixture(t, "world", "belaria.html")
	result, sourceURL, err := parseWorldHTML("Belaria", "https://www.rubinot.com.br/?subtopic=worlds&world=Belaria", html)
	if err != nil {
		t.Fatalf("expected belaria fixture to parse, got error: %v", err)
	}
	if sourceURL == "" {
		t.Fatal("expected non-empty source URL")
	}
	if result.Name != "Belaria" {
		t.Fatalf("expected world name Belaria, got %q", result.Name)
	}
	if !strings.EqualFold(result.Info.Status, "Online") {
		t.Fatalf("expected Online status, got %q", result.Info.Status)
	}
	if !strings.HasSuffix(result.Info.CreationDate, "Z") {
		t.Fatalf("expected RFC3339 UTC creation date, got %q", result.Info.CreationDate)
	}
	if len(result.PlayersOnline) == 0 {
		t.Fatal("expected non-empty players list for belaria fixture")
	}
}

func TestParseWorldHTMLOfflineFixture(t *testing.T) {
	// FIXTURE: synthetic, must be replaced with real capture
	html := readFixture(t, "world", "offline_world.html")
	result, _, err := parseWorldHTML("Belaria", "https://www.rubinot.com.br/?subtopic=worlds&world=Belaria", html)
	if err != nil {
		t.Fatalf("expected offline fixture to parse, got error: %v", err)
	}
	if !strings.EqualFold(result.Info.Status, "Offline") {
		t.Fatalf("expected Offline status, got %q", result.Info.Status)
	}
	if result.Info.PlayersOnline != 0 {
		t.Fatalf("expected players_online=0 for offline world, got %d", result.Info.PlayersOnline)
	}
	if len(result.PlayersOnline) != 0 {
		t.Fatalf("expected empty players list for offline world, got %d", len(result.PlayersOnline))
	}
}

func TestParseWorldHTMLNotFoundFixture(t *testing.T) {
	html := readFixture(t, "world", "not_found.html")
	_, _, err := parseWorldHTML("WorldDoesNotExist123", "https://www.rubinot.com.br/?subtopic=worlds&world=WorldDoesNotExist123", html)
	assertValidationCode(t, err, validation.ErrorEntityNotFound)
}

func TestFetchWorldHappy(t *testing.T) {
	worldFixture := readFixture(t, "world", "belaria.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(
			w,
			`{"status":"ok","message":"","solution":{"response":%q,"status":200,"url":"https://www.rubinot.com.br/?subtopic=worlds&world=Belaria"}}`,
			worldFixture,
		)
	}))
	defer server.Close()

	result, sourceURL, err := FetchWorld(
		context.Background(),
		"https://www.rubinot.com.br",
		"Belaria",
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchWorld to succeed, got error: %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/?subtopic=worlds&world=Belaria" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.Name != "Belaria" {
		t.Fatalf("expected world Belaria, got %q", result.Name)
	}
	if len(result.PlayersOnline) == 0 {
		t.Fatal("expected non-empty players list")
	}
}

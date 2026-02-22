package scraper

import (
	"context"
	"strings"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestParseHighscoresHTMLExperiencePage1(t *testing.T) {
	html := readFixture(t, "highscores", "experience_page1.html")
	result, err := parseHighscoresHTML(html, "Belaria", "experience", "(all)", 1)
	if err != nil {
		t.Fatalf("expected experience fixture to parse, got error: %v", err)
	}

	if result.World != "Belaria" {
		t.Fatalf("expected world Belaria, got %q", result.World)
	}
	if result.Category != "experience" {
		t.Fatalf("expected category experience, got %q", result.Category)
	}
	if result.HighscorePage.CurrentPage != 1 {
		t.Fatalf("expected current page 1, got %d", result.HighscorePage.CurrentPage)
	}
	if result.HighscorePage.TotalPages < 20 {
		t.Fatalf("expected total pages >= 20, got %d", result.HighscorePage.TotalPages)
	}
	if result.HighscorePage.TotalRecords < 1000 {
		t.Fatalf("expected total records >= 1000, got %d", result.HighscorePage.TotalRecords)
	}
	if len(result.HighscoreList) == 0 {
		t.Fatal("expected highscores list to be populated")
	}

	first := result.HighscoreList[0]
	if first.Rank <= 0 || first.Name == "" || first.Level <= 0 || first.Value <= 0 {
		t.Fatalf("expected parsed first entry fields, got %+v", first)
	}
}

func TestParseHighscoresHTMLLastPageWithAuctionMarker(t *testing.T) {
	html := readFixture(t, "highscores", "last_page.html")
	result, err := parseHighscoresHTML(html, "Belaria", "experience", "(all)", 20)
	if err != nil {
		t.Fatalf("expected last_page fixture to parse, got error: %v", err)
	}
	if result.HighscorePage.TotalPages < 20 {
		t.Fatalf("expected total pages >= 20, got %d", result.HighscorePage.TotalPages)
	}
	if len(result.HighscoreList) == 0 {
		t.Fatal("expected highscores entries on last page")
	}

	foundAuctionMarker := false
	for _, entry := range result.HighscoreList {
		if entry.Traded && strings.Contains(entry.AuctionURL, "currentcharactertrades/") {
			foundAuctionMarker = true
			break
		}
	}
	if !foundAuctionMarker {
		t.Fatal("expected at least one traded entry with auction URL marker")
	}
}

func TestParseHighscoresHTMLEmpty(t *testing.T) {
	html := readFixture(t, "highscores", "empty.html")
	result, err := parseHighscoresHTML(html, "Belaria", "experience", "(all)", 999)
	if err != nil {
		t.Fatalf("expected empty fixture to parse, got error: %v", err)
	}
	if len(result.HighscoreList) != 0 {
		t.Fatalf("expected empty highscores list, got %d entries", len(result.HighscoreList))
	}
	if result.HighscorePage.TotalPages < 20 {
		t.Fatalf("expected parsed pagination even for empty page, got total pages=%d", result.HighscorePage.TotalPages)
	}
	if result.HighscorePage.TotalRecords != 1000 {
		t.Fatalf("expected total records 1000, got %d", result.HighscorePage.TotalRecords)
	}
}

func TestParseHighscoresHTMLDynamicColumns(t *testing.T) {
	html := readFixture(t, "highscores", "dynamic_columns.html")
	result, err := parseHighscoresHTML(html, "Belaria", "loyalty", "(all)", 1)
	if err != nil {
		t.Fatalf("expected dynamic_columns fixture to parse, got error: %v", err)
	}
	if len(result.HighscoreList) == 0 {
		t.Fatal("expected highscores list for dynamic columns fixture")
	}

	foundTitle := false
	for _, entry := range result.HighscoreList {
		if entry.Title != "" {
			foundTitle = true
			break
		}
	}
	if !foundTitle {
		t.Fatal("expected loyalty fixture to include title column values")
	}
}

func TestFetchHighscoresHappy(t *testing.T) {
	fixture := readFixture(t, "highscores", "experience_page1.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return fixture
	})
	defer server.Close()

	result, sourceURL, err := FetchHighscores(
		context.Background(),
		"https://www.rubinot.com.br",
		"Belaria",
		validation.HighscoreCategory{ID: 6, Name: "Experience Points", Slug: "experience"},
		validation.HighscoreVocation{Name: "(all)", ProfessionID: 0},
		1,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchHighscores to succeed, got error: %v", err)
	}
	if !strings.Contains(sourceURL, "subtopic=highscores") {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.HighscoreList) == 0 {
		t.Fatal("expected non-empty highscores list")
	}
}

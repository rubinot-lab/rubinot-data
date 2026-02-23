package scraper

import (
	"context"
	"strings"
	"testing"
)

func TestParseKillstatisticsHTMLNormalFixture(t *testing.T) {
	html := readFixture(t, "killstatistics", "normal.html")
	result, err := parseKillstatisticsHTML("Belaria", html)
	if err != nil {
		t.Fatalf("expected normal fixture to parse, got error: %v", err)
	}

	if result.World != "Belaria" {
		t.Fatalf("expected world Belaria, got %q", result.World)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected killstatistics entries")
	}
	if result.Total.LastDayKilled <= 0 || result.Total.LastWeekKilled <= 0 {
		t.Fatalf("expected non-zero totals, got %+v", result.Total)
	}

	first := result.Entries[0]
	if first.Race == "" {
		t.Fatalf("expected first entry race, got %+v", first)
	}
}

func TestParseKillstatisticsHTMLZeroEntriesFixture(t *testing.T) {
	html := readFixture(t, "killstatistics", "zero_entries.html")
	result, err := parseKillstatisticsHTML("Belaria", html)
	if err != nil {
		t.Fatalf("expected zero_entries fixture to parse, got error: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(result.Entries))
	}
	if result.Total.LastDayKilled != 0 || result.Total.LastWeekKilled != 0 {
		t.Fatalf("expected zero totals, got %+v", result.Total)
	}
}

func TestFetchKillstatisticsHappy(t *testing.T) {
	fixture := readFixture(t, "killstatistics", "normal.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return fixture
	})
	defer server.Close()

	result, sourceURL, err := FetchKillstatistics(
		context.Background(),
		"https://www.rubinot.com.br",
		"Belaria",
		15,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchKillstatistics to succeed, got error: %v", err)
	}
	if !strings.Contains(sourceURL, "subtopic=killstatistics") {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty entries")
	}
}

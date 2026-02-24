package scraper

import (
	"context"
	"strings"
	"testing"
)

func TestParseDeathsHTMLNormalFixture(t *testing.T) {
	html := readFixture(t, "deaths", "normal.html")
	result, err := parseDeathsHTML("Belaria", DeathsFilters{}, html)
	if err != nil {
		t.Fatalf("expected normal deaths fixture to parse, got error: %v", err)
	}
	if result.World != "Belaria" {
		t.Fatalf("expected world Belaria, got %q", result.World)
	}
	if result.TotalDeaths == 0 || len(result.Entries) == 0 {
		t.Fatal("expected non-empty deaths entries")
	}

	first := result.Entries[0]
	if first.Date == "" || !strings.HasSuffix(first.Date, "Z") {
		t.Fatalf("expected normalized RFC3339 UTC date, got %q", first.Date)
	}
	if first.Victim.Name == "" || first.Victim.Level <= 0 {
		t.Fatalf("expected victim name and level, got %+v", first.Victim)
	}
	if len(first.Killers) == 0 {
		t.Fatalf("expected killers list, got %+v", first)
	}
}

func TestParseDeathsHTMLPvPFixture(t *testing.T) {
	html := readFixture(t, "deaths", "pvp.html")
	result, err := parseDeathsHTML("Belaria", DeathsFilters{PvPOnly: boolPtr(true)}, html)
	if err != nil {
		t.Fatalf("expected pvp deaths fixture to parse, got error: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty pvp entries")
	}

	for _, entry := range result.Entries {
		if !entry.IsPvP {
			t.Fatalf("expected all entries to be pvp when filter is set, got non-pvp: %+v", entry)
		}
	}
}

func TestParseDeathsHTMLMinLevelFilter(t *testing.T) {
	html := readFixture(t, "deaths", "normal.html")

	unfiltered, err := parseDeathsHTML("Belaria", DeathsFilters{}, html)
	if err != nil {
		t.Fatalf("unfiltered parse failed: %v", err)
	}
	if len(unfiltered.Entries) == 0 {
		t.Fatal("expected non-empty unfiltered entries")
	}

	filtered, err := parseDeathsHTML("Belaria", DeathsFilters{MinLevel: 500}, html)
	if err != nil {
		t.Fatalf("filtered parse failed: %v", err)
	}

	if len(filtered.Entries) >= len(unfiltered.Entries) {
		t.Fatalf("expected filtered entries (%d) to be fewer than unfiltered (%d)", len(filtered.Entries), len(unfiltered.Entries))
	}
	for _, entry := range filtered.Entries {
		if entry.Victim.Level < 500 {
			t.Fatalf("expected all victims to have level >= 500, got %d for %s", entry.Victim.Level, entry.Victim.Name)
		}
	}
}

func TestParseDeathsHTMLEmptyFixture(t *testing.T) {
	html := readFixture(t, "deaths", "empty.html")
	result, err := parseDeathsHTML("Belaria", DeathsFilters{Guild: "___nonexistent___"}, html)
	if err != nil {
		t.Fatalf("expected empty deaths fixture to parse, got error: %v", err)
	}
	if result.TotalDeaths != 0 || len(result.Entries) != 0 {
		t.Fatalf("expected zero entries for empty fixture, got total=%d len=%d", result.TotalDeaths, len(result.Entries))
	}
}

func TestFetchDeathsHappy(t *testing.T) {
	normalFixture := readFixture(t, "deaths", "normal.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return normalFixture
	})
	defer server.Close()

	result, sourceURL, err := FetchDeaths(
		context.Background(),
		"https://www.rubinot.com.br",
		"Belaria",
		15,
		DeathsFilters{},
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchDeaths to succeed, got error: %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/?subtopic=latestdeaths&world=15" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.TotalDeaths == 0 {
		t.Fatal("expected non-empty deaths response")
	}
}

func TestBuildDeathsURLWithFilters(t *testing.T) {
	url := buildDeathsURL(
		"https://www.rubinot.com.br",
		15,
		DeathsFilters{
			Guild:    "My Guild",
			MinLevel: 150,
			PvPOnly:  boolPtr(true),
		},
	)

	if !strings.Contains(url, "subtopic=latestdeaths&world=15") {
		t.Fatalf("expected world parameter in url, got %s", url)
	}
	if !strings.Contains(url, "guild=My+Guild") {
		t.Fatalf("expected encoded guild in url, got %s", url)
	}
	if !strings.Contains(url, "level=150") {
		t.Fatalf("expected level parameter in url, got %s", url)
	}
	if !strings.Contains(url, "pvp=1") {
		t.Fatalf("expected pvp=1 in url, got %s", url)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

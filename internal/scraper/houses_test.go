package scraper

import (
	"context"
	"strings"
	"testing"
)

func TestParseHouseRowsVenoreFixture(t *testing.T) {
	html := readFixture(t, "houses", "venore_list.html")
	rows, err := parseHouseRows(html)
	if err != nil {
		t.Fatalf("expected venore fixture to parse, got error: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected non-empty house rows")
	}

	first := rows[0]
	if first.HouseID <= 0 || first.Name == "" || first.Size <= 0 || first.Rent <= 0 {
		t.Fatalf("expected populated first house row, got %+v", first)
	}
}

func TestParseHouseRowsGuildhallsFixture(t *testing.T) {
	html := readFixture(t, "houses", "guildhalls_list.html")
	rows, err := parseHouseRows(html)
	if err != nil {
		t.Fatalf("expected guildhalls fixture to parse, got error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty guildhalls rows from upstream, got %d", len(rows))
	}
}

func TestParseHouseRowsEmptyFixture(t *testing.T) {
	html := readFixture(t, "houses", "empty.html")
	rows, err := parseHouseRows(html)
	if err != nil {
		t.Fatalf("expected empty fixture to parse, got error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no house rows, got %d", len(rows))
	}
}

func TestParseHouseRowsWithAuctionsFixture(t *testing.T) {
	html := readFixture(t, "houses", "with_auctions.html")
	rows, err := parseHouseRows(html)
	if err != nil {
		t.Fatalf("expected with_auctions fixture to parse, got error: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected non-empty house rows")
	}

	auctioned := 0
	for _, row := range rows {
		if row.IsAuctioned {
			auctioned++
		}
	}
	if auctioned == 0 {
		t.Fatal("expected at least one auctioned house")
	}
}

func TestFetchHousesHappyWithGuildhallFallback(t *testing.T) {
	housesFixture := readFixture(t, "houses", "venore_list.html")
	guildhallsFixture := readFixture(t, "houses", "guildhalls_list.html")
	server := newFakeFlareSolverrServer(t, func(url string) string {
		if strings.Contains(url, "type=guildhalls") {
			return guildhallsFixture
		}
		return housesFixture
	})
	defer server.Close()

	result, sourceURL, err := FetchHouses(
		context.Background(),
		"https://www.rubinot.com.br",
		"Belaria",
		15,
		"Venore",
		1,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchHouses to succeed, got error: %v", err)
	}

	expectedSource := "https://www.rubinot.com.br/?subtopic=houses&world=15&town=1&state=&type=houses&order=name"
	if sourceURL != expectedSource {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.World != "Belaria" || result.Town != "Venore" {
		t.Fatalf("unexpected world/town: %+v", result)
	}
	if len(result.HouseList) == 0 {
		t.Fatal("expected non-empty house_list")
	}
	if len(result.GuildhallList) == 0 {
		t.Fatal("expected non-empty guildhall_list via fallback from houses list")
	}

	for _, row := range result.HouseList {
		if strings.Contains(strings.ToLower(row.Name), "guildhall") {
			t.Fatalf("expected house_list to exclude guildhalls, found %q", row.Name)
		}
	}
}

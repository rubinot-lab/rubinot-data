package scraper

import (
	"context"
	"strings"
	"testing"
)

func TestParseTransfersHTMLNormalFixture(t *testing.T) {
	html := readFixture(t, "transfers", "normal.html")
	result, err := parseTransfersHTML(TransfersFilters{Page: 1}, html)
	if err != nil {
		t.Fatalf("expected normal transfers fixture to parse, got error: %v", err)
	}
	if result.Page != 1 {
		t.Fatalf("expected page=1, got %d", result.Page)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty transfer entries")
	}
	if result.TotalTransfers <= 0 {
		t.Fatalf("expected total_transfers > 0, got %d", result.TotalTransfers)
	}

	first := result.Entries[0]
	if first.PlayerName == "" || first.Level <= 0 {
		t.Fatalf("expected player name and level, got %+v", first)
	}
	if first.FormerWorld == "" || first.DestinationWorld == "" {
		t.Fatalf("expected former/destination worlds, got %+v", first)
	}
	if first.TransferDate == "" || !strings.HasSuffix(first.TransferDate, "Z") {
		t.Fatalf("expected normalized transfer_date in RFC3339 UTC, got %q", first.TransferDate)
	}
}

func TestParseTransfersHTMLEmptyFixture(t *testing.T) {
	html := readFixture(t, "transfers", "empty.html")
	result, err := parseTransfersHTML(TransfersFilters{Page: 1}, html)
	if err != nil {
		t.Fatalf("expected empty transfers fixture to parse, got error: %v", err)
	}
	if result.TotalTransfers != 0 {
		t.Fatalf("expected total_transfers=0, got %d", result.TotalTransfers)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(result.Entries))
	}
}

func TestFetchTransfersHappy(t *testing.T) {
	normalFixture := readFixture(t, "transfers", "normal.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return normalFixture
	})
	defer server.Close()

	result, sourceURL, err := FetchTransfers(
		context.Background(),
		"https://www.rubinot.com.br",
		TransfersFilters{Page: 1},
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchTransfers to succeed, got error: %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/?subtopic=transferstatistics" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty transfers response")
	}
}

func TestBuildTransfersURLWithFilters(t *testing.T) {
	url := buildTransfersURL(
		"https://www.rubinot.com.br",
		TransfersFilters{
			WorldID:  15,
			MinLevel: 120,
			Page:     3,
		},
	)

	if !strings.Contains(url, "subtopic=transferstatistics") {
		t.Fatalf("expected transferstatistics path, got %s", url)
	}
	if !strings.Contains(url, "world=15") {
		t.Fatalf("expected world filter in URL, got %s", url)
	}
	if !strings.Contains(url, "level=120") {
		t.Fatalf("expected level filter in URL, got %s", url)
	}
	if !strings.Contains(url, "currentpage=3") {
		t.Fatalf("expected currentpage filter in URL, got %s", url)
	}
}

package scraper

import (
	"context"
	"strings"
	"testing"
)

func TestParseBanishmentsHTMLNormalFixture(t *testing.T) {
	html := readFixture(t, "banishments", "normal.html")
	result, err := parseBanishmentsHTML("Belaria", 1, html)
	if err != nil {
		t.Fatalf("expected normal fixture to parse, got error: %v", err)
	}

	if result.World != "Belaria" {
		t.Fatalf("expected world Belaria, got %q", result.World)
	}
	if result.Page != 1 {
		t.Fatalf("expected page 1, got %d", result.Page)
	}
	if result.TotalBans != 2 {
		t.Fatalf("expected total_bans=2, got %d", result.TotalBans)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 ban entries, got %d", len(result.Entries))
	}

	first := result.Entries[0]
	if first.Character != "Bad Actor" {
		t.Fatalf("expected first character Bad Actor, got %+v", first)
	}
	if first.Date == "" || !strings.HasSuffix(first.Date, "Z") {
		t.Fatalf("expected UTC RFC3339 date for first entry, got %q", first.Date)
	}
	if first.IsPermanent {
		t.Fatalf("expected first entry to be temporary, got %+v", first)
	}
	if first.Duration != "7 days" {
		t.Fatalf("expected first duration to be 7 days, got %q", first.Duration)
	}

	second := result.Entries[1]
	if second.ExpiresAt == "" || !strings.HasSuffix(second.ExpiresAt, "Z") {
		t.Fatalf("expected second entry expires_at in UTC RFC3339, got %q", second.ExpiresAt)
	}
}

func TestParseBanishmentsHTMLPermanentFixture(t *testing.T) {
	html := readFixture(t, "banishments", "permanent.html")
	result, err := parseBanishmentsHTML("Belaria", 1, html)
	if err != nil {
		t.Fatalf("expected permanent fixture to parse, got error: %v", err)
	}

	if result.TotalBans != 1 {
		t.Fatalf("expected total_bans=1, got %d", result.TotalBans)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 ban entry, got %d", len(result.Entries))
	}

	entry := result.Entries[0]
	if !entry.IsPermanent {
		t.Fatalf("expected entry to be permanent, got %+v", entry)
	}
	if entry.ExpiresAt != "" {
		t.Fatalf("expected no expires_at for permanent entry, got %q", entry.ExpiresAt)
	}
}

func TestParseBanishmentsHTMLEmptyFixture(t *testing.T) {
	html := readFixture(t, "banishments", "empty.html")
	result, err := parseBanishmentsHTML("Belaria", 1, html)
	if err != nil {
		t.Fatalf("expected empty fixture to parse, got error: %v", err)
	}
	if result.TotalBans != 0 {
		t.Fatalf("expected total_bans=0, got %d", result.TotalBans)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(result.Entries))
	}
}

func TestFetchBanishmentsHappy(t *testing.T) {
	fixture := readFixture(t, "banishments", "normal.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return fixture
	})
	defer server.Close()

	result, sourceURL, err := FetchBanishments(
		context.Background(),
		"https://www.rubinot.com.br",
		"Belaria",
		15,
		1,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchBanishments to succeed, got error: %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/?subtopic=bans&world=15" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty banishments response")
	}
}

func TestBuildBanishmentsURLWithPage(t *testing.T) {
	pageOneURL := buildBanishmentsURL("https://www.rubinot.com.br", 15, 1)
	if strings.Contains(pageOneURL, "currentpage=") {
		t.Fatalf("did not expect currentpage parameter on page 1 URL: %s", pageOneURL)
	}

	pageThreeURL := buildBanishmentsURL("https://www.rubinot.com.br", 15, 3)
	if !strings.Contains(pageThreeURL, "currentpage=3") {
		t.Fatalf("expected currentpage=3 in URL, got %s", pageThreeURL)
	}
}

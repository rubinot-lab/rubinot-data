package scraper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestParseAuctionsListHTMLCurrentFixture(t *testing.T) {
	html := readFixture(t, "auctions", "current.html")
	result, err := parseAuctionsListHTML(auctionTypeCurrent, 1, html)
	if err != nil {
		t.Fatalf("expected current auctions fixture to parse, got error: %v", err)
	}
	if result.Type != auctionTypeCurrent {
		t.Fatalf("expected type=current, got %q", result.Type)
	}
	if result.Page != 1 {
		t.Fatalf("expected page=1, got %d", result.Page)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty current auctions entries")
	}
	if result.TotalPages <= 1 {
		t.Fatalf("expected total_pages > 1, got %d", result.TotalPages)
	}
	if result.TotalResults <= 0 {
		t.Fatalf("expected total_results > 0, got %d", result.TotalResults)
	}

	first := result.Entries[0]
	if first.AuctionID == "" || first.CharacterName == "" {
		t.Fatalf("expected first entry auction_id and character_name, got %+v", first)
	}
	if first.Level <= 0 || first.Vocation == "" || first.World == "" {
		t.Fatalf("expected first entry level/vocation/world, got %+v", first)
	}
	if first.Status != "active" && first.Status != "ended" {
		t.Fatalf("unexpected first entry status %q", first.Status)
	}
}

func TestParseAuctionsListHTMLHistoryFixture(t *testing.T) {
	html := readFixture(t, "auctions", "history.html")
	result, err := parseAuctionsListHTML(auctionTypeHistory, 1, html)
	if err != nil {
		t.Fatalf("expected history auctions fixture to parse, got error: %v", err)
	}
	if result.Type != auctionTypeHistory {
		t.Fatalf("expected type=history, got %q", result.Type)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty history auctions entries")
	}
	if result.Entries[0].Status != "ended" {
		t.Fatalf("expected first history entry status ended, got %+v", result.Entries[0])
	}
}

func TestParseAuctionsListHTMLEmptyFixtures(t *testing.T) {
	currentEmpty := readFixture(t, "auctions", "current_empty.html")
	result, err := parseAuctionsListHTML(auctionTypeCurrent, 999, currentEmpty)
	if err != nil {
		t.Fatalf("expected current_empty fixture to parse, got error: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected no entries in current_empty, got %d", len(result.Entries))
	}
	if result.TotalResults != 0 {
		t.Fatalf("expected total_results=0 for current_empty, got %d", result.TotalResults)
	}

	historyEmpty := readFixture(t, "auctions", "history_empty.html")
	historyResult, historyErr := parseAuctionsListHTML(auctionTypeHistory, 99999, historyEmpty)
	if historyErr != nil {
		t.Fatalf("expected history_empty fixture to parse, got error: %v", historyErr)
	}
	if len(historyResult.Entries) != 0 {
		t.Fatalf("expected no entries in history_empty, got %d", len(historyResult.Entries))
	}
	if historyResult.TotalResults != 0 {
		t.Fatalf("expected total_results=0 for history_empty, got %d", historyResult.TotalResults)
	}
}

func TestParseAuctionDetailHTMLActiveFixture(t *testing.T) {
	html := readFixture(t, "auctions", "detail_active.html")
	detail, err := parseAuctionDetailHTML("164830", html)
	if err != nil {
		t.Fatalf("expected active detail fixture to parse, got error: %v", err)
	}
	if detail.AuctionID != "164830" {
		t.Fatalf("expected auction_id=164830, got %q", detail.AuctionID)
	}
	if detail.CharacterName == "" || detail.Level <= 0 {
		t.Fatalf("expected character_name and level in active detail, got %+v", detail)
	}
	if detail.Status != "active" {
		t.Fatalf("expected active detail status active, got %q", detail.Status)
	}
	if detail.BidType != "current" || detail.BidValue <= 0 {
		t.Fatalf("expected current bid in active detail, got %+v", detail)
	}
	if detail.AuctionStart == "" || !strings.HasSuffix(detail.AuctionStart, "Z") {
		t.Fatalf("expected UTC auction_start in active detail, got %q", detail.AuctionStart)
	}
	if detail.AuctionEnd == "" || !strings.HasSuffix(detail.AuctionEnd, "Z") {
		t.Fatalf("expected UTC auction_end in active detail, got %q", detail.AuctionEnd)
	}
}

func TestParseAuctionDetailHTMLEndedFixture(t *testing.T) {
	html := readFixture(t, "auctions", "detail_ended.html")
	detail, err := parseAuctionDetailHTML("145062", html)
	if err != nil {
		t.Fatalf("expected ended detail fixture to parse, got error: %v", err)
	}
	if detail.Status != "ended" {
		t.Fatalf("expected ended detail status ended, got %+v", detail)
	}
	if detail.BidType != "winning" || detail.BidValue <= 0 {
		t.Fatalf("expected winning bid in ended detail, got %+v", detail)
	}
}

func TestParseAuctionDetailHTMLNotFoundFixture(t *testing.T) {
	html := readFixture(t, "auctions", "detail_not_found.html")
	_, err := parseAuctionDetailHTML("999999999", html)
	if err == nil {
		t.Fatal("expected not-found error from detail_not_found fixture")
	}

	var validationErr validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation.Error, got %T: %v", err, err)
	}
	if validationErr.Code() != validation.ErrorEntityNotFound {
		t.Fatalf("expected error code %d, got %d", validation.ErrorEntityNotFound, validationErr.Code())
	}
}

func TestFetchCurrentAuctionsHappy(t *testing.T) {
	fixture := readFixture(t, "auctions", "current.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return fixture
	})
	defer server.Close()

	result, sourceURL, err := FetchCurrentAuctions(
		context.Background(),
		"https://www.rubinot.com.br",
		1,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchCurrentAuctions to succeed, got error: %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/currentcharactertrades" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty current auctions response")
	}
}

func TestFetchAuctionDetailFallbackToPastURL(t *testing.T) {
	notFoundFixture := readFixture(t, "auctions", "detail_not_found.html")
	endedFixture := readFixture(t, "auctions", "detail_ended.html")
	server := newFakeFlareSolverrServer(t, func(url string) string {
		switch {
		case strings.Contains(url, "?currentcharactertrades/145062"):
			return notFoundFixture
		case strings.Contains(url, "?pastcharactertrades/145062"):
			return endedFixture
		default:
			return notFoundFixture
		}
	})
	defer server.Close()

	detail, sources, err := FetchAuctionDetail(
		context.Background(),
		"https://www.rubinot.com.br",
		"145062",
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected fallback to past URL to succeed, got error: %v", err)
	}
	if detail.Status != "ended" {
		t.Fatalf("expected ended detail after fallback, got %+v", detail)
	}
	if len(sources) != 1 || !strings.Contains(sources[0], "?pastcharactertrades/145062") {
		t.Fatalf("expected source to be past URL, got %+v", sources)
	}
}

func TestBuildAuctionsListURL(t *testing.T) {
	currentPageOne := buildAuctionsListURL("https://www.rubinot.com.br", auctionTypeCurrent, 1)
	if currentPageOne != "https://www.rubinot.com.br/currentcharactertrades" {
		t.Fatalf("unexpected current page 1 URL: %s", currentPageOne)
	}

	currentPageTwo := buildAuctionsListURL("https://www.rubinot.com.br", auctionTypeCurrent, 2)
	if !strings.Contains(currentPageTwo, "subtopic=currentcharactertrades") || !strings.Contains(currentPageTwo, "currentpage=2") {
		t.Fatalf("unexpected current page 2 URL: %s", currentPageTwo)
	}

	historyPageOne := buildAuctionsListURL("https://www.rubinot.com.br", auctionTypeHistory, 1)
	if historyPageOne != "https://www.rubinot.com.br/pastcharactertrades" {
		t.Fatalf("unexpected history page 1 URL: %s", historyPageOne)
	}
}

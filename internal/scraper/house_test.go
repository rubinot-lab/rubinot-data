package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestParseHouseHTMLRented(t *testing.T) {
	html := readFixture(t, "house", "rented.html")
	house, notFound, err := parseHouseHTML(html, 50, "Belaria", "Venore")
	if err != nil {
		t.Fatalf("expected rented fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected rented fixture to exist")
	}
	if house.Status != "rented" {
		t.Fatalf("expected rented status, got %q", house.Status)
	}
	if house.Owner == nil || house.Owner.Name == "" {
		t.Fatalf("expected owner data for rented house, got %+v", house.Owner)
	}
}

func TestParseHouseHTMLAuctioned(t *testing.T) {
	html := readFixture(t, "house", "auctioned.html")
	house, notFound, err := parseHouseHTML(html, 31, "Belaria", "Venore")
	if err != nil {
		t.Fatalf("expected auctioned fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected auctioned fixture to exist")
	}
	if house.Status != "auctioned" {
		t.Fatalf("expected auctioned status, got %q", house.Status)
	}
	if house.Auction == nil || house.Auction.CurrentBid <= 0 || house.Auction.NoBidYet {
		t.Fatalf("expected active bid auction data, got %+v", house.Auction)
	}
}

func TestParseHouseHTMLAuctionedNoBid(t *testing.T) {
	html := readFixture(t, "house", "auctioned_no_bid.html")
	house, notFound, err := parseHouseHTML(html, 3437, "Belaria", "Moonfall")
	if err != nil {
		t.Fatalf("expected no-bid auction fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected no-bid auction fixture to exist")
	}
	if house.Status != "auctioned" {
		t.Fatalf("expected auctioned status, got %q", house.Status)
	}
	if house.Auction == nil || !house.Auction.NoBidYet {
		t.Fatalf("expected no-bid auction flag, got %+v", house.Auction)
	}
}

func TestParseHouseHTMLVacantSynthetic(t *testing.T) {
	html := readFixture(t, "house", "vacant.html")
	house, notFound, err := parseHouseHTML(html, 50, "Belaria", "Venore")
	if err != nil {
		t.Fatalf("expected vacant fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected vacant fixture to exist")
	}
	if house.Status != "vacant" {
		t.Fatalf("expected vacant status, got %q", house.Status)
	}
}

func TestParseHouseHTMLMovingSynthetic(t *testing.T) {
	html := readFixture(t, "house", "moving.html")
	house, notFound, err := parseHouseHTML(html, 50, "Belaria", "Venore")
	if err != nil {
		t.Fatalf("expected moving fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected moving fixture to exist")
	}
	if house.Status != "moving" {
		t.Fatalf("expected moving status, got %q", house.Status)
	}
	if house.Owner == nil || house.Owner.MovingDate == "" {
		t.Fatalf("expected moving date for moving status, got %+v", house.Owner)
	}
}

func TestParseHouseHTMLTransferSynthetic(t *testing.T) {
	html := readFixture(t, "house", "transfer.html")
	house, notFound, err := parseHouseHTML(html, 50, "Belaria", "Venore")
	if err != nil {
		t.Fatalf("expected transfer fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected transfer fixture to exist")
	}
	if house.Status != "transfer" {
		t.Fatalf("expected transfer status, got %q", house.Status)
	}
}

func TestParseHouseHTMLNotFound(t *testing.T) {
	html := readFixture(t, "house", "not_found.html")
	_, notFound, err := parseHouseHTML(html, 999999, "Belaria", "Venore")
	if err != nil {
		t.Fatalf("expected not-found fixture parse without hard error, got: %v", err)
	}
	if !notFound {
		t.Fatal("expected not-found fixture to set notFound=true")
	}
}

func TestFetchHouseNotFound(t *testing.T) {
	notFoundFixture := readFixture(t, "house", "not_found.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return notFoundFixture
	})
	defer server.Close()

	_, _, err := FetchHouse(
		context.Background(),
		"https://www.rubinot.com.br",
		"Belaria",
		15,
		999999,
		[]validation.Town{{ID: 1, Name: "Venore"}, {ID: 66, Name: "Moonfall"}},
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err == nil {
		t.Fatal("expected not-found error from FetchHouse")
	}

	var validationErr validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation.Error, got %T: %v", err, err)
	}
	if validationErr.Code() != validation.ErrorEntityNotFound {
		t.Fatalf("expected not-found code %d, got %d", validation.ErrorEntityNotFound, validationErr.Code())
	}
}

func newFakeFlareSolverrServer(t *testing.T, getHTML func(url string) string) *httptest.Server {
	t.Helper()

	type requestPayload struct {
		URL string `json:"url"`
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			t.Fatalf("unexpected HTTP method: %s", r.Method)
		}

		var payload requestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode flaresolverr payload: %v", err)
		}

		responseHTML := getHTML(payload.URL)
		fmt.Fprintf(
			w,
			`{"status":"ok","message":"","solution":{"response":%q,"status":200,"url":%q}}`,
			responseHTML,
			payload.URL,
		)
	}))
}

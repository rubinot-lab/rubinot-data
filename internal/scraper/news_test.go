package scraper

import (
	"context"
	"strings"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestParseNewsArticleHTMLFixture(t *testing.T) {
	html := readFixture(t, "news", "article.html")
	result, notFound, err := parseNewsArticleHTML(140, html)
	if err != nil {
		t.Fatalf("expected article fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected article fixture to be found")
	}
	if result.Type != "article" {
		t.Fatalf("expected article type, got %q", result.Type)
	}
	if result.ID != 140 {
		t.Fatalf("expected id 140, got %d", result.ID)
	}
	if result.Title == "" || result.Content == "" {
		t.Fatalf("expected title and content, got %+v", result)
	}
	if result.Date == "" || !strings.HasSuffix(result.Date, "Z") {
		t.Fatalf("expected normalized RFC3339 date, got %q", result.Date)
	}
}

func TestParseNewsTickerEntryFixture(t *testing.T) {
	html := readFixture(t, "news", "ticker.html")
	result, notFound, err := parseNewsTickerEntryByIndex(1, html)
	if err != nil {
		t.Fatalf("expected ticker fixture to parse, got error: %v", err)
	}
	if notFound {
		t.Fatal("expected ticker entry to exist")
	}
	if result.Type != "ticker" {
		t.Fatalf("expected ticker type, got %q", result.Type)
	}
	if result.Content == "" {
		t.Fatalf("expected ticker content, got %+v", result)
	}
	if result.Date == "" || !strings.HasSuffix(result.Date, "Z") {
		t.Fatalf("expected normalized RFC3339 date, got %q", result.Date)
	}
}

func TestParseNewsArticleHTMLNotFoundFixture(t *testing.T) {
	html := readFixture(t, "news", "not_found.html")
	_, notFound, err := parseNewsArticleHTML(999999, html)
	if err != nil {
		t.Fatalf("expected not_found fixture parse without hard error, got: %v", err)
	}
	if !notFound {
		t.Fatal("expected not_found fixture to be detected")
	}
}

func TestParseNewsArchiveListHTMLFixture(t *testing.T) {
	html := readFixture(t, "news", "list.html")
	result, err := parseNewsArchiveListHTML(html, 90)
	if err != nil {
		t.Fatalf("expected archive list fixture to parse, got error: %v", err)
	}
	if result.Mode != "archive" {
		t.Fatalf("expected mode archive, got %q", result.Mode)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected archive entries")
	}

	first := result.Entries[0]
	if first.ID <= 0 || first.Title == "" || first.Type != "article" {
		t.Fatalf("expected parsed archive entry fields, got %+v", first)
	}
}

func TestParseNewsArchiveListHTMLEmptyFixture(t *testing.T) {
	html := readFixture(t, "news", "empty.html")
	result, err := parseNewsArchiveListHTML(html, 90)
	if err != nil {
		t.Fatalf("expected empty fixture to parse, got error: %v", err)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected no archive entries, got %d", len(result.Entries))
	}
}

func TestParseNewsLatestListHTMLFixture(t *testing.T) {
	html := readFixture(t, "news", "ticker.html")
	result, err := parseNewsLatestListHTML(html)
	if err != nil {
		t.Fatalf("expected latest list fixture to parse, got error: %v", err)
	}
	if result.Mode != "latest" {
		t.Fatalf("expected mode latest, got %q", result.Mode)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected latest entries")
	}
}

func TestParseNewsTickerListHTMLFixture(t *testing.T) {
	html := readFixture(t, "news", "ticker.html")
	result, err := parseNewsTickerListHTML(html)
	if err != nil {
		t.Fatalf("expected ticker list fixture to parse, got error: %v", err)
	}
	if result.Mode != "newsticker" {
		t.Fatalf("expected mode newsticker, got %q", result.Mode)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected ticker entries")
	}
}

func TestFetchNewsByIDHappy(t *testing.T) {
	articleFixture := readFixture(t, "news", "article.html")
	tickerFixture := readFixture(t, "news", "ticker.html")
	server := newFakeFlareSolverrServer(t, func(url string) string {
		if strings.Contains(url, "?news/archive/140") {
			return articleFixture
		}
		if strings.Contains(url, "?news") {
			return tickerFixture
		}
		return articleFixture
	})
	defer server.Close()

	result, sources, err := FetchNewsByID(
		context.Background(),
		"https://www.rubinot.com.br",
		140,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchNewsByID to succeed, got error: %v", err)
	}
	if result.Type != "article" {
		t.Fatalf("expected article result type, got %q", result.Type)
	}
	if len(sources) != 1 || !strings.Contains(sources[0], "news/archive/140") {
		t.Fatalf("unexpected sources: %#v", sources)
	}
}

func TestFetchNewsByIDNotFound(t *testing.T) {
	notFoundFixture := readFixture(t, "news", "not_found.html")
	tickerFixture := readFixture(t, "news", "ticker.html")
	server := newFakeFlareSolverrServer(t, func(url string) string {
		if strings.Contains(url, "?news/archive/999999") {
			return notFoundFixture
		}
		if strings.Contains(url, "?news") {
			return tickerFixture
		}
		return notFoundFixture
	})
	defer server.Close()

	_, _, err := FetchNewsByID(
		context.Background(),
		"https://www.rubinot.com.br",
		999999,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	assertValidationCode(t, err, validation.ErrorEntityNotFound)
}

func TestFetchNewsArchiveHappy(t *testing.T) {
	archiveFixture := readFixture(t, "news", "list.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return archiveFixture
	})
	defer server.Close()

	result, sourceURL, err := FetchNewsArchive(
		context.Background(),
		"https://www.rubinot.com.br",
		90,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchNewsArchive to succeed, got error: %v", err)
	}
	if !strings.Contains(sourceURL, "?news/archive") {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected non-empty archive entries")
	}
}

func TestFetchNewsLatestAndTickerHappy(t *testing.T) {
	tickerFixture := readFixture(t, "news", "ticker.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return tickerFixture
	})
	defer server.Close()

	latest, latestSource, latestErr := FetchNewsLatest(
		context.Background(),
		"https://www.rubinot.com.br",
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if latestErr != nil {
		t.Fatalf("expected FetchNewsLatest to succeed, got error: %v", latestErr)
	}
	if !strings.Contains(latestSource, "?news") {
		t.Fatalf("unexpected latest source URL: %s", latestSource)
	}
	if len(latest.Entries) == 0 {
		t.Fatal("expected latest entries")
	}

	ticker, tickerSource, tickerErr := FetchNewsTicker(
		context.Background(),
		"https://www.rubinot.com.br",
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if tickerErr != nil {
		t.Fatalf("expected FetchNewsTicker to succeed, got error: %v", tickerErr)
	}
	if !strings.Contains(tickerSource, "?news") {
		t.Fatalf("unexpected ticker source URL: %s", tickerSource)
	}
	if len(ticker.Entries) == 0 {
		t.Fatal("expected ticker entries")
	}
}

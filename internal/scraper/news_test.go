package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchNewsByIDFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/news")
		writeJSON(w, map[string]any{
			"tickers": []map[string]any{{
				"id":          146,
				"message":     "<p>[World Transfer]</p>",
				"category_id": 3,
				"category":    map[string]any{"id": 3, "name": "Events", "slug": "events", "color": "#9b59b6", "icon": "calendar", "icon_url": "https://static.rubinot.com/news/categories/events.gif"},
				"author":      "vtn",
				"created_at":  "2026-02-20T00:00:00.000Z",
			}},
			"articles": []map[string]any{{
				"id":           3,
				"title":        "Announcement",
				"slug":         "announcement",
				"summary":      "summary",
				"content":      "<p>hello</p>",
				"cover_image":  "images/news/announcement.jpg",
				"author":       "@guido",
				"category":     map[string]any{"id": 1, "name": "Community", "slug": "community", "color": "#fff", "icon": "users", "icon_url": "https://example/icon.gif"},
				"published_at": "2024-07-01T00:00:00.000Z",
			}},
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	entry, _, err := FetchNewsByID(context.Background(), baseURLOf(api), 3, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if entry.Title != "Announcement" || entry.Type != "article" {
		t.Fatalf("unexpected news entry: %+v", entry)
	}

	ticker, _, err := FetchNewsByID(context.Background(), baseURLOf(api), 146, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ticker.Type != "ticker" {
		t.Fatalf("expected ticker type, got %+v", ticker)
	}
}

func TestFetchNewsListsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/news")
		writeJSON(w, map[string]any{
			"tickers": []map[string]any{{
				"id":          146,
				"message":     "<p>[World Transfer]</p>",
				"category_id": 3,
				"category":    map[string]any{"id": 3, "name": "Events", "slug": "events", "color": "#9b59b6", "icon": "calendar", "icon_url": "https://static.rubinot.com/news/categories/events.gif"},
				"author":      "vtn",
				"created_at":  "2026-02-20T00:00:00.000Z",
			}},
			"articles": []map[string]any{{
				"id":           3,
				"title":        "Announcement",
				"slug":         "announcement",
				"summary":      "summary",
				"content":      "<p>hello</p>",
				"cover_image":  "images/news/announcement.jpg",
				"author":       "@guido",
				"category":     map[string]any{"id": 1, "name": "Community", "slug": "community", "color": "#fff", "icon": "users", "icon_url": "https://example/icon.gif"},
				"published_at": "2026-02-24T00:00:00.000Z",
			}},
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	archive, _, err := FetchNewsArchive(context.Background(), baseURLOf(api), 365, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("archive error: %v", err)
	}
	if len(archive.Entries) == 0 {
		t.Fatal("expected archive entries")
	}

	latest, _, err := FetchNewsLatest(context.Background(), baseURLOf(api), testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("latest error: %v", err)
	}
	if len(latest.Entries) != 1 || latest.Entries[0].Type != "article" {
		t.Fatalf("unexpected latest entries: %+v", latest.Entries)
	}

	ticker, _, err := FetchNewsTicker(context.Background(), baseURLOf(api), testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("ticker error: %v", err)
	}
	if len(ticker.Entries) != 1 || ticker.Entries[0].Type != "ticker" {
		t.Fatalf("unexpected ticker entries: %+v", ticker.Entries)
	}
}

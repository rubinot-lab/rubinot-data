package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestFetchGuildsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/guilds")
		assertQuery(t, r, "world", "15")
		assertQuery(t, r, "page", "2")
		writeJSON(w, map[string]any{
			"guilds": []map[string]any{{
				"id":          2089,
				"name":        "A Banda",
				"description": "desc",
				"world_id":    15,
				"logo_name":   "guild_2089.gif",
			}},
			"totalCount":  30,
			"totalPages":  2,
			"currentPage": 2,
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, _, err := FetchGuilds(context.Background(), baseURLOf(api), "Belaria", 15, 2, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Guilds) != 1 {
		t.Fatalf("expected one guild entry, got %d", len(result.Guilds))
	}
	if result.Guilds[0].ID != 2089 {
		t.Fatalf("unexpected guild payload: %+v", result.Guilds[0])
	}
	if result.Pagination == nil || result.Pagination.CurrentPage != 2 {
		t.Fatalf("unexpected pagination payload: %+v", result.Pagination)
	}
}

func TestFetchAllGuildsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/guilds")
		assertQuery(t, r, "world", "15")
		page := r.URL.Query().Get("page")
		writeJSON(w, map[string]any{
			"guilds": []map[string]any{{
				"id":          2000 + mustAtoi(t, page),
				"name":        "Guild " + page,
				"description": "desc " + page,
				"world_id":    15,
				"logo_name":   "guild_" + page + ".gif",
			}},
			"totalCount":  3,
			"totalPages":  3,
			"currentPage": mustAtoi(t, page),
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sources, err := FetchAllGuilds(context.Background(), baseURLOf(api), "Belaria", 15, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Guilds) != 3 {
		t.Fatalf("expected 3 guild entries, got %d", len(result.Guilds))
	}
	if len(sources) != 3 {
		t.Fatalf("expected 3 source URLs, got %d", len(sources))
	}
	if result.Pagination == nil || result.Pagination.TotalCount != 3 || result.Pagination.TotalPages != 1 {
		t.Fatalf("unexpected pagination payload: %+v", result.Pagination)
	}
}

func mustAtoi(t *testing.T, raw string) int {
	t.Helper()
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("parse int %q: %v", raw, err)
	}
	return parsed
}

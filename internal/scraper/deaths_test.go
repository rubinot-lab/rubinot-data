package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchDeathsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/deaths")
		assertQuery(t, r, "world", "15")
		assertQuery(t, r, "page", "2")
		assertQuery(t, r, "level", "100")
		assertQuery(t, r, "pvp", "true")
		writeJSON(w, map[string]any{
			"deaths": []map[string]any{
				{
					"player_id":            467572,
					"time":                 "1772043027",
					"level":                341,
					"killed_by":            "sphinx",
					"is_player":            0,
					"mostdamage_by":        "sphinx",
					"mostdamage_is_player": 0,
					"victim":               "Dona Creusa",
					"world_id":             15,
				},
			},
			"pagination": map[string]any{"currentPage": 2, "totalPages": 6, "totalCount": 300, "itemsPerPage": 50},
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	pvpOnly := true
	result, _, err := FetchDeaths(
		context.Background(),
		baseURLOf(api),
		"Belaria",
		15,
		DeathsFilters{MinLevel: 100, PvPOnly: &pvpOnly, Page: 2},
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Pagination.CurrentPage != 2 {
		t.Fatalf("expected page 2, got %d", result.Pagination.CurrentPage)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected one death entry, got %d", len(result.Entries))
	}
	if result.Entries[0].Victim.Name != "Dona Creusa" {
		t.Fatalf("unexpected victim %+v", result.Entries[0].Victim)
	}
}

func TestFetchAllDeathsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/deaths")
		assertQuery(t, r, "world", "15")
		page := mustAtoi(t, r.URL.Query().Get("page"))
		writeJSON(w, map[string]any{
			"deaths": []map[string]any{
				{
					"player_id":            467572 + page,
					"time":                 "1772043027",
					"level":                341,
					"killed_by":            "sphinx",
					"is_player":            0,
					"mostdamage_by":        "sphinx",
					"mostdamage_is_player": 0,
					"victim":               "Victim",
					"world_id":             15,
				},
			},
			"pagination": map[string]any{"currentPage": page, "totalPages": 3, "totalCount": 3, "itemsPerPage": 50},
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sources, err := FetchAllDeaths(
		context.Background(),
		baseURLOf(api),
		"Belaria",
		15,
		DeathsFilters{},
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Entries) != 3 {
		t.Fatalf("expected 3 death entries, got %d", len(result.Entries))
	}
	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}
	if result.Pagination.TotalPages != 1 {
		t.Fatalf("expected total pages 1, got %d", result.Pagination.TotalPages)
	}
}

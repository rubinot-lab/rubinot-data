package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTransfersFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/transfers")
		assertQuery(t, r, "world", "15")
		assertQuery(t, r, "level", "500")
		assertQuery(t, r, "page", "3")
		writeJSON(w, map[string]any{
			"transfers": []map[string]any{{
				"id":             123,
				"player_id":      456,
				"player_name":    "Hero",
				"player_level":   500,
				"from_world_id":  11,
				"to_world_id":    15,
				"transferred_at": int64(1772043027000),
			}},
			"totalResults": 1000,
			"totalPages":   20,
			"currentPage":  3,
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, _, err := FetchTransfers(
		context.Background(),
		baseURLOf(api),
		TransfersFilters{WorldID: 15, WorldName: "Belaria", MinLevel: 500, Page: 3},
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Page != 3 || result.TotalTransfers != 1000 {
		t.Fatalf("unexpected pagination payload: %+v", result)
	}
	if len(result.Entries) != 1 || result.Entries[0].FormerWorld != "Auroria" {
		t.Fatalf("unexpected transfer entries: %+v", result.Entries)
	}
}

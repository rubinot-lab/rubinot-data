package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchKillstatisticsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/killstats")
		assertQuery(t, r, "world", "15")
		writeJSON(w, map[string]any{
			"entries": []map[string]any{{
				"race_name":            "dragon",
				"players_killed_24h":   2,
				"creatures_killed_24h": 500,
				"players_killed_7d":    10,
				"creatures_killed_7d":  3500,
			}},
			"totals": map[string]any{
				"players_killed_24h":   660,
				"creatures_killed_24h": 1948846,
				"players_killed_7d":    6074,
				"creatures_killed_7d":  18566020,
			},
		})
	}))
	defer api.Close()

	fs := newFlareSolverrJSONServer(t, nil)
	defer fs.Close()

	result, _, err := FetchKillstatistics(context.Background(), baseURLOf(api), "Belaria", 15, testFetchOptions(fs.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(result.Entries))
	}
	if result.Total.LastWeekKilled != 18566020 {
		t.Fatalf("unexpected totals: %+v", result.Total)
	}
}

package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchWorldFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/worlds/Belaria")
		writeJSON(w, map[string]any{
			"world": map[string]any{
				"id":           15,
				"name":         "Belaria",
				"pvpType":      "pvp",
				"pvpTypeLabel": "Open PvP",
				"worldType":    "green",
				"locked":       true,
				"creationDate": int64(1731297600),
			},
			"playersOnline": 1024,
			"record":        2048,
			"recordTime":    int64(1770597564),
			"players": []map[string]any{{
				"name":       "Tester",
				"level":      100,
				"vocation":   "Elder Druid",
				"vocationId": 6,
			}},
		})
	}))
	defer api.Close()

	fs := newFlareSolverrProxyServer(t, api)
	defer fs.Close()

	result, sourceURL, err := FetchWorld(context.Background(), baseURLOf(api), "Belaria", testFetchOptions(fs.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != baseURLOf(api)+"/api/worlds/Belaria" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.Name != "Belaria" {
		t.Fatalf("unexpected world name: %s", result.Name)
	}
	if result.Info.Record != 2048 {
		t.Fatalf("expected record 2048, got %d", result.Info.Record)
	}
	if len(result.PlayersOnline) != 1 || result.PlayersOnline[0].VocationID != 6 {
		t.Fatalf("unexpected players payload: %+v", result.PlayersOnline)
	}
}

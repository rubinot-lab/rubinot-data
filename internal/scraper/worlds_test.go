package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchWorldsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/worlds")
		assertHasCookieHeader(t, r)
		writeJSON(w, map[string]any{
			"worlds": []map[string]any{
				{"id": 11, "name": "Auroria", "pvpType": "pvp", "pvpTypeLabel": "Open PvP", "worldType": "yellow", "locked": false, "playersOnline": 690},
				{"id": 15, "name": "Belaria", "pvpType": "pvp", "pvpTypeLabel": "Open PvP", "worldType": "green", "locked": true, "playersOnline": 1024},
			},
			"totalOnline":       1714,
			"overallRecord":     27884,
			"overallRecordTime": int64(1770583857),
		})
	}))
	defer api.Close()

	fs := newFlareSolverrJSONServer(t, nil)
	defer fs.Close()

	result, sourceURL, err := FetchWorlds(context.Background(), baseURLOf(api), testFetchOptions(fs.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != baseURLOf(api)+"/api/worlds" {
		t.Fatalf("unexpected source url: %s", sourceURL)
	}
	if result.TotalPlayersOnline != 1714 {
		t.Fatalf("expected total online 1714, got %d", result.TotalPlayersOnline)
	}
	if len(result.Worlds) != 2 {
		t.Fatalf("expected 2 worlds, got %d", len(result.Worlds))
	}
	if result.Worlds[0].ID != 11 || result.Worlds[0].Name != "Auroria" {
		t.Fatalf("unexpected first world: %+v", result.Worlds[0])
	}
}

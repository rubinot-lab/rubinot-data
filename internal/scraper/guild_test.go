package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGuildFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/guilds/Panq Alliance")
		writeJSON(w, map[string]any{
			"guild": map[string]any{
				"id":           7343,
				"name":         "Panq Alliance",
				"motd":         "",
				"description":  "desc",
				"homepage":     "",
				"world_id":     15,
				"logo_name":    "default.gif",
				"balance":      "0",
				"creationdata": int64(1748825316),
				"owner":        map[string]any{"id": 699107, "name": "Luann", "level": 917, "vocation": 6},
				"members": []map[string]any{{
					"id":        699107,
					"name":      "Luann",
					"level":     917,
					"vocation":  6,
					"rank":      "Leader",
					"rankLevel": 3,
					"nick":      "",
					"joinDate":  int64(1748825316),
					"isOnline":  false,
				}},
				"ranks":     []map[string]any{{"id": 123, "name": "Leader", "level": 3}},
				"residence": map[string]any{"id": 1, "name": "Thais", "town": "Thais"},
			},
		})
	}))
	defer api.Close()

	fs := newFlareSolverrProxyServer(t, api)
	defer fs.Close()

	result, _, err := FetchGuild(context.Background(), baseURLOf(api), "Panq Alliance", testFetchOptions(fs.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Panq Alliance" {
		t.Fatalf("unexpected guild name: %s", result.Name)
	}
	if result.World != "Belaria" || len(result.Members) != 1 {
		t.Fatalf("unexpected guild payload: %+v", result)
	}
}

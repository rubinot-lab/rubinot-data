package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGuildsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/guilds")
		assertQuery(t, r, "world", "15")
		writeJSON(w, map[string]any{
			"guilds": []map[string]any{{
				"id":          2089,
				"name":        "A Banda",
				"description": "desc",
				"world_id":    15,
				"logo_name":   "guild_2089.gif",
			}},
		})
	}))
	defer api.Close()

	fs := newFlareSolverrJSONServer(t, nil)
	defer fs.Close()

	result, _, err := FetchGuilds(context.Background(), baseURLOf(api), "Belaria", 15, testFetchOptions(fs.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Guilds) != 1 {
		t.Fatalf("expected one guild entry, got %d", len(result.Guilds))
	}
	if result.Guilds[0].ID != 2089 {
		t.Fatalf("unexpected guild payload: %+v", result.Guilds[0])
	}
}

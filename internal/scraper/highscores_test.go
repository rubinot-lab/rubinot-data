package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestFetchHighscoresFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/highscores")
		assertQuery(t, r, "world", "15")
		assertQuery(t, r, "category", "experience")
		assertQuery(t, r, "vocation", "0")
		writeJSON(w, map[string]any{
			"players": []map[string]any{
				{"rank": 1, "id": 100, "name": "A", "level": 1000, "vocation": 6, "world_id": 15, "value": "1000000"},
				{"rank": 2, "id": 101, "name": "B", "level": 999, "vocation": 8, "world_id": 15, "value": "999999"},
			},
			"totalCount": 2,
			"cachedAt":   int64(1772042575929),
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, _, err := FetchHighscores(
		context.Background(),
		baseURLOf(api),
		"Belaria",
		15,
		validation.HighscoreCategory{ID: 1, Name: "Experience", Slug: "experience"},
		validation.HighscoreVocation{Name: "(all)", ProfessionID: 0},
		1,
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.HighscorePage.TotalRecords != 2 {
		t.Fatalf("expected total records 2, got %d", result.HighscorePage.TotalRecords)
	}
	if len(result.HighscoreList) != 2 {
		t.Fatalf("expected 2 highscores, got %d", len(result.HighscoreList))
	}
	if result.HighscoreList[0].Value != "1000000" {
		t.Fatalf("expected string value 1000000, got %q", result.HighscoreList[0].Value)
	}
}

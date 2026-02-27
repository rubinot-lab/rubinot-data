package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestFetchAllHighscoresFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/highscores")
		assertQuery(t, r, "world", "15")
		assertQuery(t, r, "category", "experience")
		assertQuery(t, r, "vocation", "0")
		writeJSON(w, map[string]any{
			"players": []map[string]any{
				{"rank": 1, "id": 100, "name": "A", "level": 1000, "vocation": 6, "world_id": 15, "value": "1000000"},
				{"rank": 2, "id": 101, "name": "B", "level": 999, "vocation": 8, "world_id": 15, "value": "999999"},
				{"rank": 3, "id": 102, "name": "C", "level": 998, "vocation": 7, "world_id": 15, "value": "999998"},
			},
			"totalCount": 3,
			"cachedAt":   int64(1772042575929),
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, _, err := FetchAllHighscores(
		context.Background(),
		baseURLOf(api),
		"Belaria",
		15,
		validation.HighscoreCategory{ID: 1, Name: "Experience", Slug: "experience"},
		validation.HighscoreVocation{Name: "(all)", ProfessionID: 0},
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.HighscoreList) != 3 {
		t.Fatalf("expected 3 highscores, got %d", len(result.HighscoreList))
	}
	if result.HighscorePage.TotalPages != 1 {
		t.Fatalf("expected total pages 1, got %d", result.HighscorePage.TotalPages)
	}
}

func TestFetchHighscoresAllWorldsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/highscores")
		assertQuery(t, r, "world", "all")
		assertQuery(t, r, "category", "experience")
		assertQuery(t, r, "vocation", "0")
		writeJSON(w, map[string]any{
			"players": []map[string]any{
				{"rank": 1, "id": 200, "name": "X", "level": 1100, "vocation": 6, "world_id": 15, "value": "1000000"},
				{"rank": 2, "id": 201, "name": "Y", "level": 1090, "vocation": 8, "world_id": 11, "value": "999000"},
			},
			"totalCount": 2,
			"cachedAt":   int64(1772042575929),
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, _, err := FetchHighscoresAllWorlds(
		context.Background(),
		baseURLOf(api),
		validation.HighscoreCategory{ID: 1, Name: "Experience", Slug: "experience"},
		validation.HighscoreVocation{Name: "(all)", ProfessionID: 0},
		1,
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.World != "all" {
		t.Fatalf("expected world=all, got %q", result.World)
	}
	if len(result.HighscoreList) != 2 {
		t.Fatalf("expected 2 highscores, got %d", len(result.HighscoreList))
	}
}

func TestFetchAllHighscoresAllWorldsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/highscores")
		assertQuery(t, r, "world", "all")
		assertQuery(t, r, "category", "experience")
		assertQuery(t, r, "vocation", "0")
		writeJSON(w, map[string]any{
			"players": []map[string]any{
				{"rank": 1, "id": 200, "name": "X", "level": 1100, "vocation": 6, "world_id": 15, "value": "1000000"},
				{"rank": 2, "id": 201, "name": "Y", "level": 1090, "vocation": 8, "world_id": 11, "value": "999000"},
				{"rank": 3, "id": 202, "name": "Z", "level": 1080, "vocation": 7, "world_id": 17, "value": "998000"},
			},
			"totalCount": 3,
			"cachedAt":   int64(1772042575929),
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, _, err := FetchAllHighscoresAllWorlds(
		context.Background(),
		baseURLOf(api),
		validation.HighscoreCategory{ID: 1, Name: "Experience", Slug: "experience"},
		validation.HighscoreVocation{Name: "(all)", ProfessionID: 0},
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.World != "all" {
		t.Fatalf("expected world=all, got %q", result.World)
	}
	if len(result.HighscoreList) != 3 {
		t.Fatalf("expected 3 highscores, got %d", len(result.HighscoreList))
	}
	if result.HighscorePage.TotalPages != 1 {
		t.Fatalf("expected total pages 1, got %d", result.HighscorePage.TotalPages)
	}
}

func TestFetchAllHighscoresPerWorldFromAPI(t *testing.T) {
	var (
		mu   sync.Mutex
		hits = map[string]int{}
	)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/highscores")
		world := r.URL.Query().Get("world")
		assertQuery(t, r, "category", "exp_today")
		assertQuery(t, r, "vocation", "0")

		mu.Lock()
		hits[world]++
		mu.Unlock()

		switch world {
		case "15":
			writeJSON(w, map[string]any{
				"players": []map[string]any{
					{"rank": 1, "id": 1501, "name": "Belaria One", "level": 1000, "vocation": 6, "world_id": 15, "value": "12345"},
				},
				"totalCount": 1,
				"cachedAt":   int64(1772042575929),
			})
		case "22":
			writeJSON(w, map[string]any{
				"players": []map[string]any{
					{"rank": 1, "id": 2201, "name": "Serenian One", "level": 900, "vocation": 8, "world_id": 22, "value": "23456"},
				},
				"totalCount": 1,
				"cachedAt":   int64(1772042575930),
			})
		default:
			failUnexpectedRequest(t, r)
		}
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sources, err := FetchAllHighscoresPerWorld(
		context.Background(),
		baseURLOf(api),
		[]validation.World{
			{ID: 15, Name: "Belaria"},
			{ID: 22, Name: "Serenian I"},
		},
		validation.HighscoreCategory{ID: 12, Name: "Exp Today", Slug: "exp_today"},
		validation.HighscoreVocation{Name: "(all)", ProfessionID: 0},
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.World != "all" {
		t.Fatalf("expected grouped world=all, got %q", result.World)
	}
	if result.TotalWorlds != 2 {
		t.Fatalf("expected 2 worlds, got %d", result.TotalWorlds)
	}
	if result.TotalRecords != 2 || result.TotalEntries != 2 {
		t.Fatalf("expected totals 2/2, got %d/%d", result.TotalRecords, result.TotalEntries)
	}
	if len(result.Worlds) != 2 {
		t.Fatalf("expected 2 world payloads, got %d", len(result.Worlds))
	}
	if result.Worlds[0].World != "Belaria" || result.Worlds[1].World != "Serenian I" {
		t.Fatalf("unexpected world ordering/payloads: %+v", result.Worlds)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 source URLs, got %d", len(sources))
	}

	mu.Lock()
	defer mu.Unlock()
	if hits["15"] != 1 || hits["22"] != 1 {
		t.Fatalf("expected one request per world, got hits=%v", hits)
	}
}

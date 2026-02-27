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

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sourceURL, err := FetchWorld(context.Background(), baseURLOf(api), "Belaria", testFetchOptionsWithCDP("", cdpSrv.URL))
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

func TestFetchWorldDetailsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/worlds/Belaria":
			writeJSON(w, map[string]any{
				"world": map[string]any{
					"id":           15,
					"name":         "Belaria",
					"pvpTypeLabel": "Open PvP",
					"worldType":    "green",
					"locked":       true,
					"creationDate": int64(1731297600),
				},
				"playersOnline": 2,
				"record":        2048,
				"recordTime":    int64(1770597564),
				"players": []map[string]any{
					{"name": "Alpha", "level": 100, "vocation": "Elder Druid", "vocationId": 6},
					{"name": "Bravo", "level": 200, "vocation": "Elite Knight", "vocationId": 8},
				},
			})
		case "/api/characters/search":
			name := r.URL.Query().Get("name")
			writeJSON(w, map[string]any{
				"player": map[string]any{
					"id":            1,
					"account_id":    2,
					"name":          name,
					"level":         100,
					"vocation":      "Elder Druid",
					"vocationId":    6,
					"world_id":      15,
					"sex":           "Female",
					"residence":     "Thais",
					"lastlogin":     "1771978547",
					"created":       1749685309,
					"account_created": 1729809234,
					"loyalty_points":  70,
				},
				"deaths":               []any{},
				"otherCharacters":      []any{},
				"accountBadges":        []any{},
				"displayedAchievements": []any{},
			})
		default:
			failUnexpectedRequest(t, r)
		}
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sources, err := FetchWorldDetails(context.Background(), baseURLOf(api), "Belaria", 15, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(result.Characters))
	}
	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}
	if result.Characters[0].CharacterInfo.Name == "" {
		t.Fatalf("expected character name, got %+v", result.Characters[0].CharacterInfo)
	}
}

func TestFetchWorldDashboardFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/worlds/Belaria":
			writeJSON(w, map[string]any{
				"world": map[string]any{
					"id":           15,
					"name":         "Belaria",
					"pvpTypeLabel": "Open PvP",
					"worldType":    "green",
					"locked":       true,
					"creationDate": int64(1731297600),
				},
				"playersOnline": 1,
				"record":        2048,
				"recordTime":    int64(1770597564),
				"players": []map[string]any{
					{"name": "Alpha", "level": 100, "vocation": "Elder Druid", "vocationId": 6},
				},
			})
		case "/api/deaths":
			assertQuery(t, r, "world", "15")
			assertQuery(t, r, "page", "1")
			writeJSON(w, map[string]any{
				"deaths": []map[string]any{
					{
						"player_id":            1,
						"time":                 "1772043027",
						"level":                341,
						"killed_by":            "sphinx",
						"is_player":            0,
						"mostdamage_by":        "sphinx",
						"mostdamage_is_player": 0,
						"victim":               "Alpha",
						"world_id":             15,
					},
				},
				"pagination": map[string]any{"currentPage": 1, "totalPages": 1, "totalCount": 1, "itemsPerPage": 50},
			})
		case "/api/killstats":
			assertQuery(t, r, "world", "15")
			writeJSON(w, map[string]any{
				"entries": []map[string]any{
					{"race_name": "dragon", "players_killed_24h": 1, "creatures_killed_24h": 100, "players_killed_7d": 5, "creatures_killed_7d": 700},
				},
				"totals": map[string]any{"players_killed_24h": 1, "creatures_killed_24h": 100, "players_killed_7d": 5, "creatures_killed_7d": 700},
			})
		default:
			failUnexpectedRequest(t, r)
		}
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sources, err := FetchWorldDashboard(context.Background(), baseURLOf(api), "Belaria", 15, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}
	if result.World.Name != "Belaria" {
		t.Fatalf("expected world Belaria, got %+v", result.World)
	}
	if len(result.RecentDeaths.Entries) != 1 {
		t.Fatalf("expected 1 death entry, got %d", len(result.RecentDeaths.Entries))
	}
	if len(result.KillStatistics.Entries) != 1 {
		t.Fatalf("expected 1 killstatistics entry, got %d", len(result.KillStatistics.Entries))
	}
}

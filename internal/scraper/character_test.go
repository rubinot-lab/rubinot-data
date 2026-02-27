package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCharacterFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/characters/search")
		assertQuery(t, r, "name", "Terah")
		writeJSON(w, map[string]any{
			"player": map[string]any{
				"id":                721294,
				"account_id":        170893,
				"name":              "Terah",
				"level":             1561,
				"vocation":          "Elder Druid",
				"vocationId":        6,
				"world_id":          15,
				"sex":               "Female",
				"residence":         "Svargrond",
				"lastlogin":         "1771978547",
				"created":           int64(1749685309),
				"account_created":   int64(1729809234),
				"loyalty_points":    70,
				"isHidden":          true,
				"guild":             map[string]any{"id": 10407, "name": "Ascended Belaria", "rank": "Member", "nick": ""},
				"house":             map[string]any{"id": 2384, "name": "Darashia 8, Flat 03", "town_id": 10, "rent": 300000, "size": 171},
				"formerNames":       []any{},
				"looktype":          1832,
				"lookhead":          34,
				"lookbody":          44,
				"looklegs":          44,
				"lookfeet":          43,
				"lookaddons":        3,
				"vip_time":          int64(1919328364),
				"achievementPoints": 221,
				"comment":           "",
			},
			"deaths": []map[string]any{{
				"time":                 "1770479389",
				"level":                1542,
				"killed_by":            "darklight striker",
				"is_player":            0,
				"mostdamage_by":        "darklight striker",
				"mostdamage_is_player": 0,
			}},
			"otherCharacters":       []any{},
			"accountBadges":         []any{},
			"displayedAchievements": []any{},
			"banInfo":               nil,
			"isAdmin":               false,
			"canSeeDeathDetails":    false,
			"foundByOldName":        false,
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, _, err := FetchCharacter(context.Background(), baseURLOf(api), "Terah", testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.CharacterInfo.Name != "Terah" {
		t.Fatalf("unexpected character name: %s", result.CharacterInfo.Name)
	}
	if result.CharacterInfo.AccountID != 170893 {
		t.Fatalf("unexpected account id: %d", result.CharacterInfo.AccountID)
	}
	if len(result.Deaths) != 1 {
		t.Fatalf("expected one death entry, got %d", len(result.Deaths))
	}
}

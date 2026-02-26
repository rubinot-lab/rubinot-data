package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCurrentAuctionsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/bazaar":
			assertQuery(t, r, "page", "1")
			assertQuery(t, r, "limit", "25")
			writeJSON(w, map[string]any{
				"auctions": []map[string]any{{
					"id":                166676,
					"state":             1,
					"stateName":         "active",
					"playerId":          695106,
					"owner":             239185,
					"startingValue":     400,
					"currentValue":      407,
					"auctionStart":      int64(1771874176),
					"auctionEnd":        int64(1772053200),
					"name":              "Kulikitaka ti",
					"level":             639,
					"vocation":          6,
					"vocationName":      "Elder Druid",
					"sex":               1,
					"worldId":           18,
					"worldName":         "Mystian",
					"lookType":          158,
					"lookHead":          0,
					"lookBody":          0,
					"lookLegs":          0,
					"lookFeet":          0,
					"lookAddons":        0,
					"charmPoints":       1234,
					"achievementPoints": 56,
					"magLevel":          42,
					"skills":            map[string]any{"axe": 10, "club": 10, "distance": 10, "shielding": 41, "sword": 10},
					"highlightItems":    []map[string]any{},
					"highlightAugments": []map[string]any{},
				}},
				"pagination": map[string]any{"page": 1, "limit": 25, "total": 1661, "totalPages": 67},
			})
		case "/api/bazaar/history":
			writeJSON(w, map[string]any{
				"auctions":   []map[string]any{},
				"pagination": map[string]any{"page": 1, "limit": 25, "total": 15535, "totalPages": 622},
			})
		case "/api/bazaar/193226":
			writeJSON(w, map[string]any{
				"auction":           map[string]any{"id": 193226, "state": 1, "stateName": "active", "playerId": 123, "owner": 456, "startingValue": 100, "currentValue": 200, "winningBid": 0, "highestBidderId": 0, "auctionStart": int64(1772000000), "auctionEnd": int64(1772100000)},
				"player":            map[string]any{"name": "Terah", "level": 1000, "vocation": 6, "vocationName": "Elder Druid", "sex": 0, "worldId": 15, "worldName": "Belaria", "lookType": 1832, "lookHead": 34, "lookBody": 44, "lookLegs": 44, "lookFeet": 43, "lookAddons": 3, "lookMount": 10},
				"general":           map[string]any{"achievementPoints": 221, "charmPoints": 500, "magLevel": 120, "skills": map[string]any{"axe": 10, "club": 10, "distance": 10, "shielding": 41, "sword": 10}},
				"highlightItems":    []map[string]any{{"itemId": 1001, "name": "Soulcutter"}},
				"highlightAugments": []map[string]any{{"argType": 1, "text": "Life Leech"}},
			})
		default:
			failUnexpectedRequest(t, r)
		}
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	current, _, err := FetchCurrentAuctions(context.Background(), baseURLOf(api), 1, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("current auctions error: %v", err)
	}
	if len(current.Entries) != 1 || current.Entries[0].AuctionID != 166676 {
		t.Fatalf("unexpected current entries: %+v", current.Entries)
	}

	history, _, err := FetchAuctionHistory(context.Background(), baseURLOf(api), 1, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("history auctions error: %v", err)
	}
	if history.TotalPages != 622 {
		t.Fatalf("unexpected history payload: %+v", history)
	}

	detail, _, err := FetchAuctionDetail(context.Background(), baseURLOf(api), 193226, testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("auction detail error: %v", err)
	}
	if detail.AuctionID != 193226 || detail.CharacterName != "Terah" {
		t.Fatalf("unexpected detail payload: %+v", detail)
	}
}

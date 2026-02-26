package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

const eventsFixtureHTML = `
<html>
  <body>
    <span class="flex items-center gap-2">Fevereiro 2026</span>
    <table class="w-full table-fixed border-collapse text-xs">
      <tr><th>Seg</th><th>Ter</th></tr>
      <tr><td><div>24</div><div>Castle</div></td><td><div>25</div><div>Skill Event</div></td></tr>
    </table>
  </body>
</html>
`

func TestRouterIntegrationHappyPaths(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL)

	testCases := []struct {
		name       string
		path       string
		httpCode   int
		errorCode  int
		payloadKey string
	}{
		{name: "worlds", path: "/v1/worlds", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "world", path: "/v1/world/Belaria", httpCode: http.StatusOK, payloadKey: "world"},
		{name: "character", path: "/v1/character/Terah", httpCode: http.StatusOK, payloadKey: "character"},
		{name: "guild", path: "/v1/guild/Panq%20Alliance", httpCode: http.StatusOK, payloadKey: "guild"},
		{name: "guilds", path: "/v1/guilds/Belaria", httpCode: http.StatusOK, payloadKey: "guilds"},
		{name: "highscores", path: "/v1/highscores/Belaria/experience/all/1", httpCode: http.StatusOK, payloadKey: "highscores"},
		{name: "killstatistics", path: "/v1/killstatistics/Belaria", httpCode: http.StatusOK, payloadKey: "killstatistics"},
		{name: "news by id", path: "/v1/news/id/140", httpCode: http.StatusOK, payloadKey: "news"},
		{name: "news archive", path: "/v1/news/archive?days=90", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "news latest", path: "/v1/news/latest", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "news ticker", path: "/v1/news/newsticker", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "deaths", path: "/v1/deaths/Belaria?page=1", httpCode: http.StatusOK, payloadKey: "deaths"},
		{name: "transfers", path: "/v1/transfers?page=1", httpCode: http.StatusOK, payloadKey: "transfers"},
		{name: "banishments", path: "/v1/banishments/Belaria?page=1", httpCode: http.StatusOK, payloadKey: "banishments"},
		{name: "events", path: "/v1/events/schedule", httpCode: http.StatusOK, payloadKey: "events"},
		{name: "auctions current", path: "/v1/auctions/current/1", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions history", path: "/v1/auctions/history/1", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auction detail", path: "/v1/auctions/193226", httpCode: http.StatusOK, payloadKey: "auction"},
		{name: "houses deprecated", path: "/v1/houses/Belaria/Venore", httpCode: http.StatusGone, errorCode: validation.ErrorEndpointDeprecated},
		{name: "house deprecated", path: "/v1/house/Belaria/50", httpCode: http.StatusGone, errorCode: validation.ErrorEndpointDeprecated},
		{name: "towns deprecated", path: "/v1/houses/towns", httpCode: http.StatusGone, errorCode: validation.ErrorEndpointDeprecated},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(router, http.MethodGet, tc.path)
			if rec.Code != tc.httpCode {
				t.Fatalf("expected status %d, got %d: %s", tc.httpCode, rec.Code, rec.Body.String())
			}

			body := decodeJSONBody(t, rec)
			assertEnvelope(t, body, tc.httpCode, tc.errorCode)
			if tc.payloadKey != "" {
				if _, exists := body[tc.payloadKey]; !exists {
					t.Fatalf("expected payload key %q in response body", tc.payloadKey)
				}
			}
		})
	}
}

func TestRouterHighscoresRedirects(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL)

	testCases := []struct {
		path     string
		location string
	}{
		{path: "/v1/highscores/Belaria", location: "/v1/highscores/Belaria/experience/all/1"},
		{path: "/v1/highscores/Belaria/magic", location: "/v1/highscores/Belaria/magic/all/1"},
	}

	for _, tc := range testCases {
		rec := performRequest(router, http.MethodGet, tc.path)
		if rec.Code != http.StatusFound {
			t.Fatalf("expected redirect status %d, got %d", http.StatusFound, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != tc.location {
			t.Fatalf("expected redirect location %q, got %q", tc.location, got)
		}
	}
}

func TestRouterValidationErrors(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL)

	testCases := []struct {
		path  string
		code  int
		error int
	}{
		{path: "/v1/world/Nope", code: http.StatusBadRequest, error: validation.ErrorWorldDoesNotExist},
		{path: "/v1/character/a", code: http.StatusBadRequest, error: validation.ErrorCharacterNameTooShort},
		{path: "/v1/highscores/Belaria/unknown/all/1", code: http.StatusBadRequest, error: validation.ErrorHighscoreCategoryDoesNotExist},
		{path: "/v1/transfers?page=0", code: http.StatusBadRequest, error: validation.ErrorPageOutOfBounds},
	}

	for _, tc := range testCases {
		rec := performRequest(router, http.MethodGet, tc.path)
		if rec.Code != tc.code {
			t.Fatalf("expected status %d, got %d", tc.code, rec.Code)
		}
		body := decodeJSONBody(t, rec)
		assertEnvelope(t, body, tc.code, tc.error)
	}
}

func TestRouterMaintenancePropagation(t *testing.T) {
	api := newAPIUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/worlds" {
			writeJSON(w, map[string]any{"worlds": []map[string]any{{"id": 15, "name": "Belaria"}}})
			return
		}
		if r.URL.Path == "/api/highscores" {
			writeJSON(w, map[string]any{"error": validation.UpstreamMaintenanceMessage})
			return
		}
		writeJSON(w, map[string]any{"error": "unsupported"})
	})
	defer api.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL)
	rec := performRequest(router, http.MethodGet, "/v1/highscores/Belaria/experience/all/1")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	body := decodeJSONBody(t, rec)
	assertEnvelope(t, body, http.StatusServiceUnavailable, validation.ErrorUpstreamMaintenanceMode)
}

func TestRouterNotFoundPropagation(t *testing.T) {
	api := newAPIUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/worlds" {
			writeJSON(w, map[string]any{"worlds": []map[string]any{{"id": 15, "name": "Belaria"}}})
			return
		}
		if r.URL.Path == "/api/characters/search" {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]any{"error": "Character not found"})
			return
		}
		writeJSON(w, map[string]any{"error": "unsupported"})
	})
	defer api.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL)
	rec := performRequest(router, http.MethodGet, "/v1/character/Unknown%20Player")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	body := decodeJSONBody(t, rec)
	assertEnvelope(t, body, http.StatusNotFound, validation.ErrorEntityNotFound)
}

func newIntegrationTestRouter(t *testing.T, flaresolverrURL, baseURL string) http.Handler {
	t.Helper()
	t.Setenv("FLARESOLVERR_URL", flaresolverrURL)
	t.Setenv("RUBINOT_BASE_URL", baseURL)
	t.Setenv("SCRAPE_MAX_TIMEOUT_MS", "120000")
	t.Setenv("SCRAPE_MAX_CONCURRENCY", "8")

	router, err := NewRouter()
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}
	return http.Handler(router)
}

func newFakeFlareSolverrForRouter(t *testing.T, eventsHTML string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected flaresolverr method: %s", r.Method)
		}

		var payload struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode flaresolverr request: %v", err)
		}

		body := "<html><body>ok</body></html>"
		solutionStatus := http.StatusOK
		if strings.Contains(payload.URL, "/events") {
			body = eventsHTML
		} else {
			proxyResp, err := http.Get(payload.URL)
			if err != nil {
				t.Fatalf("failed to proxy to target: %v", err)
			}
			defer proxyResp.Body.Close()
			raw, err := io.ReadAll(proxyResp.Body)
			if err != nil {
				t.Fatalf("failed to read proxy response: %v", err)
			}
			body = string(raw)
			solutionStatus = proxyResp.StatusCode
		}

		resp := map[string]any{
			"status":  "ok",
			"message": "",
			"solution": map[string]any{
				"response": body,
				"status":   solutionStatus,
				"url":      payload.URL,
			},
		}
		writeJSON(w, resp)
	}))
}

func newHappyAPIUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	return newAPIUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/worlds":
			writeJSON(w, map[string]any{"worlds": []map[string]any{{"id": 15, "name": "Belaria", "pvpType": "pvp", "pvpTypeLabel": "Open PvP", "worldType": "green", "locked": true, "playersOnline": 1000}}, "totalOnline": 1000, "overallRecord": 27884, "overallRecordTime": 1770583857})
		case r.URL.Path == "/api/worlds/Belaria":
			writeJSON(w, map[string]any{"world": map[string]any{"id": 15, "name": "Belaria", "pvpTypeLabel": "Open PvP", "worldType": "green", "locked": true, "creationDate": 1731297600}, "playersOnline": 1000, "record": 2000, "recordTime": 1770597564, "players": []map[string]any{{"name": "Test", "level": 100, "vocation": "Elder Druid", "vocationId": 6}}})
		case r.URL.Path == "/api/characters/search":
			writeJSON(w, map[string]any{"player": map[string]any{"id": 1, "account_id": 2, "name": "Terah", "level": 100, "vocation": "Elder Druid", "vocationId": 6, "world_id": 15, "sex": "Female", "residence": "Thais", "lastlogin": "1771978547", "created": 1749685309, "account_created": 1729809234, "loyalty_points": 70, "isHidden": false, "formerNames": []any{}, "looktype": 1, "lookhead": 2, "lookbody": 3, "looklegs": 4, "lookfeet": 5, "lookaddons": 0, "vip_time": 1919328364, "achievementPoints": 221, "comment": ""}, "deaths": []any{}, "otherCharacters": []any{}, "accountBadges": []any{}, "displayedAchievements": []any{}, "banInfo": nil, "foundByOldName": false})
		case r.URL.Path == "/api/guilds":
			if name := r.URL.Query().Get("world"); name == "15" {
				writeJSON(w, map[string]any{"guilds": []map[string]any{{"id": 10, "name": "Panq Alliance", "description": "desc", "world_id": 15, "logo_name": "default.gif"}}})
				return
			}
			writeJSON(w, map[string]any{"error": "invalid world"})
		case strings.HasPrefix(r.URL.Path, "/api/guilds/"):
			writeJSON(w, map[string]any{"guild": map[string]any{"id": 10, "name": "Panq Alliance", "motd": "", "description": "desc", "homepage": "", "world_id": 15, "logo_name": "default.gif", "balance": "0", "creationdata": 1748825316, "owner": map[string]any{"id": 1, "name": "Leader", "level": 1000, "vocation": 6}, "members": []map[string]any{{"id": 1, "name": "Leader", "level": 1000, "vocation": 6, "rank": "Leader", "rankLevel": 3, "nick": "", "joinDate": 1748825316, "isOnline": false}}, "ranks": []map[string]any{{"id": 1, "name": "Leader", "level": 3}}, "residence": map[string]any{"id": 1, "name": "Thais", "town": "Thais"}}})
		case r.URL.Path == "/api/highscores":
			writeJSON(w, map[string]any{"players": []map[string]any{{"rank": 1, "id": 1, "name": "Top", "level": 1000, "vocation": 6, "world_id": 15, "value": "999"}}, "totalCount": 1, "cachedAt": 1772042575929})
		case r.URL.Path == "/api/killstats":
			writeJSON(w, map[string]any{"entries": []map[string]any{{"race_name": "dragon", "players_killed_24h": 1, "creatures_killed_24h": 100, "players_killed_7d": 5, "creatures_killed_7d": 700}}, "totals": map[string]any{"players_killed_24h": 1, "creatures_killed_24h": 100, "players_killed_7d": 5, "creatures_killed_7d": 700}})
		case r.URL.Path == "/api/news":
			writeJSON(w, map[string]any{"tickers": []map[string]any{{"id": 200, "message": "<p>ticker</p>", "category_id": 1, "category": map[string]any{"id": 1, "name": "Events", "slug": "events", "color": "#fff", "icon": "calendar", "icon_url": "https://example/icon.gif"}, "author": "staff", "created_at": "2026-02-20T00:00:00.000Z"}}, "articles": []map[string]any{{"id": 140, "title": "News", "slug": "news", "summary": "sum", "content": "<p>body</p>", "cover_image": "cover.jpg", "author": "staff", "category": map[string]any{"id": 2, "name": "Community", "slug": "community", "color": "#000", "icon": "users", "icon_url": "https://example/icon2.gif"}, "published_at": "2026-02-24T00:00:00.000Z"}}})
		case r.URL.Path == "/api/deaths":
			writeJSON(w, map[string]any{"deaths": []map[string]any{{"player_id": 1, "time": "1772043027", "level": 300, "killed_by": "dragon", "is_player": 0, "mostdamage_by": "dragon", "mostdamage_is_player": 0, "victim": "Victim", "world_id": 15}}, "pagination": map[string]any{"currentPage": 1, "totalPages": 1, "totalCount": 1, "itemsPerPage": 50}})
		case r.URL.Path == "/api/transfers":
			writeJSON(w, map[string]any{"transfers": []map[string]any{{"id": 1, "player_id": 1, "player_name": "Player", "player_level": 500, "from_world_id": 11, "to_world_id": 15, "transferred_at": 1772043027000}}, "totalResults": 1, "totalPages": 1, "currentPage": 1})
		case r.URL.Path == "/api/bans":
			writeJSON(w, map[string]any{"bans": []map[string]any{{"account_id": 1, "account_name": "acc", "main_character": "Player", "reason": "Rule", "banned_at": "1772043027", "expires_at": "-1", "banned_by": "GM", "is_permanent": true}}, "totalCount": 1, "totalPages": 1, "currentPage": 1})
		case r.URL.Path == "/api/bazaar":
			writeJSON(w, map[string]any{"auctions": []map[string]any{{"id": 193226, "state": 1, "stateName": "active", "playerId": 1, "owner": 2, "startingValue": 100, "currentValue": 200, "auctionStart": 1772000000, "auctionEnd": 1772100000, "name": "Char", "level": 1000, "vocation": 6, "vocationName": "Elder Druid", "sex": 0, "worldId": 15, "worldName": "Belaria", "lookType": 1, "lookHead": 2, "lookBody": 3, "lookLegs": 4, "lookFeet": 5, "lookAddons": 0, "charmPoints": 10, "achievementPoints": 20, "magLevel": 30, "skills": map[string]any{"axe": 1, "club": 2, "distance": 3, "shielding": 4, "sword": 5}, "highlightItems": []any{}, "highlightAugments": []any{}}}, "pagination": map[string]any{"page": 1, "limit": 25, "total": 1, "totalPages": 1}})
		case r.URL.Path == "/api/bazaar/history":
			writeJSON(w, map[string]any{"auctions": []any{}, "pagination": map[string]any{"page": 1, "limit": 25, "total": 1, "totalPages": 1}})
		case strings.HasPrefix(r.URL.Path, "/api/bazaar/"):
			writeJSON(w, map[string]any{"auction": map[string]any{"id": 193226, "state": 1, "stateName": "active", "playerId": 1, "owner": 2, "startingValue": 100, "currentValue": 200, "winningBid": 0, "highestBidderId": 0, "auctionStart": 1772000000, "auctionEnd": 1772100000}, "player": map[string]any{"name": "Char", "level": 1000, "vocation": 6, "vocationName": "Elder Druid", "sex": 0, "worldId": 15, "worldName": "Belaria", "lookType": 1, "lookHead": 2, "lookBody": 3, "lookLegs": 4, "lookFeet": 5, "lookAddons": 0, "lookMount": 0}, "general": map[string]any{"achievementPoints": 20, "charmPoints": 10, "magLevel": 30, "skills": map[string]any{"axe": 1, "club": 2, "distance": 3, "shielding": 4, "sword": 5}}, "highlightItems": []any{}, "highlightAugments": []any{}})
		default:
			t.Fatalf("unexpected upstream request: %s", r.URL.String())
		}
	})
}

func newAPIUpstream(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(handler))
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func performRequest(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeJSONBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode JSON response %q: %v", rec.Body.String(), err)
	}
	return out
}

func assertEnvelope(t *testing.T, body map[string]any, expectedHTTPCode int, expectedErrorCode int) {
	t.Helper()

	information, ok := body["information"].(map[string]any)
	if !ok {
		t.Fatalf("missing information envelope in body: %+v", body)
	}

	status, ok := information["status"].(map[string]any)
	if !ok {
		t.Fatalf("missing information.status in body: %+v", body)
	}

	if got := toInt(t, status["http_code"]); got != expectedHTTPCode {
		t.Fatalf("expected information.status.http_code=%d, got %d", expectedHTTPCode, got)
	}

	if expectedErrorCode == 0 {
		if _, hasError := status["error"]; hasError {
			t.Fatalf("did not expect error code in success response: %+v", status)
		}
		return
	}

	if got := toInt(t, status["error"]); got != expectedErrorCode {
		t.Fatalf("expected information.status.error=%d, got %d", expectedErrorCode, got)
	}
}

func toInt(t *testing.T, value any) int {
	t.Helper()
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			t.Fatalf("failed to parse json.Number %q: %v", v, err)
		}
		return int(i)
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("failed to parse int string %q: %v", v, err)
		}
		return i
	default:
		t.Fatalf("unsupported numeric type %T (%v)", value, value)
		return 0
	}
}

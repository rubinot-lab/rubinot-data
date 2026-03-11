package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestV2RouterRegistration(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)

	routes := []struct {
		name   string
		method string
		path   string
	}{
		{name: "v2 worlds", method: http.MethodGet, path: "/v2/worlds"},
		{name: "v2 world", method: http.MethodGet, path: "/v2/world/Belaria"},
		{name: "v2 world details", method: http.MethodGet, path: "/v2/world/Belaria/details"},
		{name: "v2 world dashboard", method: http.MethodGet, path: "/v2/world/Belaria/dashboard"},
		{name: "v2 highscores", method: http.MethodGet, path: "/v2/highscores/Belaria/experience/all"},
		{name: "v2 killstatistics", method: http.MethodGet, path: "/v2/killstatistics/Belaria"},
		{name: "v2 deaths", method: http.MethodGet, path: "/v2/deaths/Belaria"},
		{name: "v2 deaths all", method: http.MethodGet, path: "/v2/deaths/Belaria/all"},
		{name: "v2 banishments", method: http.MethodGet, path: "/v2/banishments/Belaria"},
		{name: "v2 banishments all", method: http.MethodGet, path: "/v2/banishments/Belaria/all"},
		{name: "v2 transfers", method: http.MethodGet, path: "/v2/transfers"},
		{name: "v2 transfers all", method: http.MethodGet, path: "/v2/transfers/all"},
		{name: "v2 character", method: http.MethodGet, path: "/v2/character/Terah"},
		{name: "v2 guild", method: http.MethodGet, path: "/v2/guild/Panq%20Alliance"},
		{name: "v2 guilds", method: http.MethodGet, path: "/v2/guilds/Belaria"},
		{name: "v2 guilds all", method: http.MethodGet, path: "/v2/guilds/Belaria/all"},
		{name: "v2 boosted", method: http.MethodGet, path: "/v2/boosted"},
		{name: "v2 maintenance", method: http.MethodGet, path: "/v2/maintenance"},
		{name: "v2 auctions current all", method: http.MethodGet, path: "/v2/auctions/current/all"},
		{name: "v2 auctions current page", method: http.MethodGet, path: "/v2/auctions/current/1"},
		{name: "v2 auctions history all", method: http.MethodGet, path: "/v2/auctions/history/all"},
		{name: "v2 auctions history page", method: http.MethodGet, path: "/v2/auctions/history/1"},
		{name: "v2 auction detail", method: http.MethodGet, path: "/v2/auctions/193226"},
		{name: "v2 news by id", method: http.MethodGet, path: "/v2/news/id/140"},
		{name: "v2 news archive", method: http.MethodGet, path: "/v2/news/archive"},
		{name: "v2 news latest", method: http.MethodGet, path: "/v2/news/latest"},
		{name: "v2 news ticker", method: http.MethodGet, path: "/v2/news/newsticker"},
		{name: "v2 outfit", method: http.MethodGet, path: "/v2/outfit?looktype=131"},
		{name: "v2 outfit by name", method: http.MethodGet, path: "/v2/outfit/Terah"},
	}

	for _, tc := range routes {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(router, tc.method, tc.path)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("route %s %s returned 404; route not registered", tc.method, tc.path)
			}
		})
	}
}

func TestV2HappyPaths(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)

	testCases := []struct {
		name       string
		path       string
		httpCode   int
		payloadKey string
	}{
		{name: "v2 worlds", path: "/v2/worlds", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "v2 world", path: "/v2/world/Belaria", httpCode: http.StatusOK, payloadKey: "world"},
		{name: "v2 world all", path: "/v2/world/all", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "v2 world details", path: "/v2/world/Belaria/details", httpCode: http.StatusOK, payloadKey: "world"},
		{name: "v2 world all details", path: "/v2/world/all/details", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "v2 world dashboard", path: "/v2/world/Belaria/dashboard", httpCode: http.StatusOK, payloadKey: "dashboard"},
		{name: "v2 world all dashboard", path: "/v2/world/all/dashboard", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "v2 highscores", path: "/v2/highscores/Belaria/experience/all", httpCode: http.StatusOK, payloadKey: "highscores"},
		{name: "v2 killstatistics", path: "/v2/killstatistics/Belaria", httpCode: http.StatusOK, payloadKey: "killstatistics"},
		{name: "v2 deaths", path: "/v2/deaths/Belaria?page=1", httpCode: http.StatusOK, payloadKey: "deaths"},
		{name: "v2 deaths all", path: "/v2/deaths/Belaria/all", httpCode: http.StatusOK, payloadKey: "deaths"},
		{name: "v2 banishments", path: "/v2/banishments/Belaria?page=1", httpCode: http.StatusOK, payloadKey: "banishments"},
		{name: "v2 banishments all", path: "/v2/banishments/Belaria/all", httpCode: http.StatusOK, payloadKey: "banishments"},
		{name: "v2 transfers", path: "/v2/transfers?page=1", httpCode: http.StatusOK, payloadKey: "transfers"},
		{name: "v2 transfers all", path: "/v2/transfers/all", httpCode: http.StatusOK, payloadKey: "transfers"},
		{name: "v2 character", path: "/v2/character/Terah", httpCode: http.StatusOK, payloadKey: "character"},
		{name: "v2 guild", path: "/v2/guild/Panq%20Alliance", httpCode: http.StatusOK, payloadKey: "guild"},
		{name: "v2 guilds", path: "/v2/guilds/Belaria", httpCode: http.StatusOK, payloadKey: "guilds"},
		{name: "v2 guilds all", path: "/v2/guilds/Belaria/all", httpCode: http.StatusOK, payloadKey: "guilds"},
		{name: "v2 boosted", path: "/v2/boosted", httpCode: http.StatusOK, payloadKey: "boosted"},
		{name: "v2 maintenance", path: "/v2/maintenance", httpCode: http.StatusOK, payloadKey: "maintenance"},
		{name: "v2 auctions current all", path: "/v2/auctions/current/all", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "v2 auctions current page", path: "/v2/auctions/current/1", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "v2 auctions history all", path: "/v2/auctions/history/all", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "v2 auctions history page", path: "/v2/auctions/history/1", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "v2 auction detail", path: "/v2/auctions/193226", httpCode: http.StatusOK, payloadKey: "auction"},
		{name: "v2 news by id", path: "/v2/news/id/140", httpCode: http.StatusOK, payloadKey: "news"},
		{name: "v2 news archive", path: "/v2/news/archive?days=90", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "v2 news latest", path: "/v2/news/latest", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "v2 news ticker", path: "/v2/news/newsticker", httpCode: http.StatusOK, payloadKey: "newslist"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(router, http.MethodGet, tc.path)
			if rec.Code != tc.httpCode {
				t.Fatalf("expected status %d, got %d: %s", tc.httpCode, rec.Code, rec.Body.String())
			}

			body := decodeJSONBody(t, rec)
			assertEnvelope(t, body, tc.httpCode, 0)
			if tc.payloadKey != "" {
				if _, exists := body[tc.payloadKey]; !exists {
					t.Fatalf("expected payload key %q in response body, got keys: %v", tc.payloadKey, mapKeys(body))
				}
			}
		})
	}
}

func TestV2ValidationErrors(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)

	testCases := []struct {
		name  string
		path  string
		code  int
		error int
	}{
		{name: "v2 world not found", path: "/v2/world/Nope", code: http.StatusBadRequest, error: validation.ErrorWorldDoesNotExist},
		{name: "v2 character name too short", path: "/v2/character/a", code: http.StatusBadRequest, error: validation.ErrorCharacterNameTooShort},
		{name: "v2 highscores bad category", path: "/v2/highscores/Belaria/unknown/all", code: http.StatusBadRequest, error: validation.ErrorHighscoreCategoryDoesNotExist},
		{name: "v2 highscores bad vocation", path: "/v2/highscores/Belaria/experience/fakevoc", code: http.StatusBadRequest, error: validation.ErrorVocationDoesNotExist},
		{name: "v2 transfers bad page", path: "/v2/transfers?page=0", code: http.StatusBadRequest, error: validation.ErrorPageOutOfBounds},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(router, http.MethodGet, tc.path)
			if rec.Code != tc.code {
				t.Fatalf("expected status %d, got %d: %s", tc.code, rec.Code, rec.Body.String())
			}
			body := decodeJSONBody(t, rec)
			assertEnvelope(t, body, tc.code, tc.error)
		})
	}
}

func TestV2BoostedHasImageURLs(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)
	rec := performRequest(router, http.MethodGet, "/v2/boosted")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Boosted struct {
			Boss struct {
				ImageURL string `json:"image_url"`
			} `json:"boss"`
			Monster struct {
				ImageURL string `json:"image_url"`
			} `json:"monster"`
		} `json:"boosted"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Boosted.Boss.ImageURL == "" {
		t.Fatal("expected boss image_url to be set")
	}
	if resp.Boosted.Monster.ImageURL == "" {
		t.Fatal("expected monster image_url to be set")
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}


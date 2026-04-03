package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
	"github.com/gorilla/websocket"
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

type openAPIParam struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

type openAPITagMeta struct {
	Name string `json:"name"`
}

type openAPIRequestBodyMeta struct {
	Required bool `json:"required"`
	Content  map[string]struct {
		Schema map[string]any `json:"schema"`
	} `json:"content"`
}

type openAPIResponseMeta struct {
	Description string `json:"description"`
}

func TestRouterIntegrationHappyPaths(t *testing.T) {
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
		errorCode  int
		payloadKey string
	}{
		{name: "worlds", path: "/v1/worlds", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "world all", path: "/v1/world/all", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "world all details", path: "/v1/world/all/details", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "world all dashboard", path: "/v1/world/all/dashboard", httpCode: http.StatusOK, payloadKey: "worlds"},
		{name: "world details", path: "/v1/world/Belaria/details", httpCode: http.StatusOK, payloadKey: "world"},
		{name: "world dashboard", path: "/v1/world/Belaria/dashboard", httpCode: http.StatusOK, payloadKey: "dashboard"},
		{name: "world", path: "/v1/world/Belaria", httpCode: http.StatusOK, payloadKey: "world"},
		{name: "character", path: "/v1/character/Terah", httpCode: http.StatusOK, payloadKey: "character"},
		{name: "guild", path: "/v1/guild/Panq%20Alliance", httpCode: http.StatusOK, payloadKey: "guild"},
		{name: "guilds", path: "/v1/guilds/Belaria", httpCode: http.StatusOK, payloadKey: "guilds"},
		{name: "guilds page", path: "/v1/guilds/Belaria/1", httpCode: http.StatusOK, payloadKey: "guilds"},
		{name: "guilds all", path: "/v1/guilds/Belaria/all", httpCode: http.StatusOK, payloadKey: "guilds"},
		{name: "guilds all details", path: "/v1/guilds/Belaria/all/details", httpCode: http.StatusOK, payloadKey: "guilds"},
		{name: "highscores", path: "/v1/highscores/Belaria/experience/all/1", httpCode: http.StatusOK, payloadKey: "highscores"},
		{name: "highscores all", path: "/v1/highscores/Belaria/experience/all/all", httpCode: http.StatusOK, payloadKey: "highscores"},
		{name: "highscores all worlds", path: "/v1/highscores/all/experience/all/1", httpCode: http.StatusOK, payloadKey: "highscores"},
		{name: "highscores all worlds all pages", path: "/v1/highscores/all/experience/all/all", httpCode: http.StatusOK, payloadKey: "highscores"},
		{name: "killstatistics", path: "/v1/killstatistics/Belaria", httpCode: http.StatusOK, payloadKey: "killstatistics"},
		{name: "news by id", path: "/v1/news/id/140", httpCode: http.StatusOK, payloadKey: "news"},
		{name: "news archive", path: "/v1/news/archive?days=90", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "news latest", path: "/v1/news/latest", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "news ticker", path: "/v1/news/newsticker", httpCode: http.StatusOK, payloadKey: "newslist"},
		{name: "boosted", path: "/v1/boosted", httpCode: http.StatusOK, payloadKey: "boosted"},
		{name: "maintenance", path: "/v1/maintenance", httpCode: http.StatusOK, payloadKey: "maintenance"},
		{name: "geo-language", path: "/v1/geo-language", httpCode: http.StatusOK, payloadKey: "geo_language"},
		{name: "deaths", path: "/v1/deaths/Belaria?page=1", httpCode: http.StatusOK, payloadKey: "deaths"},
		{name: "deaths all", path: "/v1/deaths/Belaria/all", httpCode: http.StatusOK, payloadKey: "deaths"},
		{name: "transfers", path: "/v1/transfers?page=1", httpCode: http.StatusOK, payloadKey: "transfers"},
		{name: "transfers all", path: "/v1/transfers/all", httpCode: http.StatusOK, payloadKey: "transfers"},
		{name: "banishments", path: "/v1/banishments/Belaria?page=1", httpCode: http.StatusOK, payloadKey: "banishments"},
		{name: "banishments all", path: "/v1/banishments/Belaria/all", httpCode: http.StatusOK, payloadKey: "banishments"},
		{name: "events", path: "/v1/events/schedule", httpCode: http.StatusOK, payloadKey: "events"},
		{name: "events calendar", path: "/v1/events/calendar", httpCode: http.StatusOK, payloadKey: "events"},
		{name: "auctions current", path: "/v1/auctions/current/1", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions current details", path: "/v1/auctions/current/1/details", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions current all", path: "/v1/auctions/current/all", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions current all details", path: "/v1/auctions/current/all/details", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions history", path: "/v1/auctions/history/1", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions history details", path: "/v1/auctions/history/1/details", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions history all", path: "/v1/auctions/history/all", httpCode: http.StatusOK, payloadKey: "auctions"},
		{name: "auctions history all details", path: "/v1/auctions/history/all/details", httpCode: http.StatusOK, payloadKey: "auctions"},
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

func TestRouterOutfitBinary(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)
	for _, path := range []string{
		"/v1/outfit?looktype=131",
		"/v1/outfit/Terah",
		"/v1/outfit/Terah?direction=0&animated=0&walk=0&size=1&format=gif",
	} {
		rec := performRequest(router, http.MethodGet, path)
		if rec.Code != http.StatusOK {
			t.Fatalf("path %q: expected status %d, got %d: %s", path, http.StatusOK, rec.Code, rec.Body.String())
		}
		ct := rec.Header().Get("Content-Type")
		if strings.Contains(path, "format=gif") {
			if !strings.Contains(ct, "image/gif") {
				t.Fatalf("path %q: expected content-type image/gif, got %q", path, ct)
			}
		} else if !strings.Contains(ct, "image/png") {
			t.Fatalf("path %q: expected content-type image/png, got %q", path, ct)
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("path %q: expected non-empty binary body", path)
		}
		sourceURL := rec.Header().Get("X-Source-Url")
		if !strings.Contains(sourceURL, "/api/outfit?") {
			t.Fatalf("path %q: expected X-Source-Url pointing to /api/outfit, got %q", path, sourceURL)
		}
		if !strings.Contains(sourceURL, "type=") {
			t.Fatalf("path %q: expected normalized type query in X-Source-Url, got %q", path, sourceURL)
		}
	}
}

func TestRouterAssetRoutesRegistered(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"v1 creature asset", "/v1/assets/creatures/nonexistent", http.StatusNotFound, "creature not found"},
		{"v1 item asset invalid", "/v1/assets/items/abc", http.StatusBadRequest, "invalid item ID"},
		{"v1 charm asset", "/v1/assets/charms/nonexistent", http.StatusNotFound, "not found"},
		{"v1 creature-type asset", "/v1/assets/creature-types/nonexistent", http.StatusNotFound, "not found"},
		{"v2 creature asset", "/v2/assets/creatures/nonexistent", http.StatusNotFound, "creature not found"},
		{"v2 item asset invalid", "/v2/assets/items/abc", http.StatusBadRequest, "invalid item ID"},
		{"v2 charm asset", "/v2/assets/charms/nonexistent", http.StatusNotFound, "not found"},
		{"v2 creature-type asset", "/v2/assets/creature-types/nonexistent", http.StatusNotFound, "not found"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(router, http.MethodGet, tc.path)
			if rec.Code != tc.wantStatus {
				t.Fatalf("path %q: expected status %d, got %d: %s", tc.path, tc.wantStatus, rec.Code, rec.Body.String())
			}
			body := rec.Body.String()
			if body != tc.wantBody {
				t.Fatalf("path %q: expected body %q, got %q", tc.path, tc.wantBody, body)
			}
		})
	}
}

func TestRouterOpenAPISpecIncludesRegisteredRoutes(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()

	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()

	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)
	rec := performRequest(router, http.MethodGet, "/openapi.json")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected application/json content-type, got %q", got)
	}

	var spec struct {
		OpenAPI string           `json:"openapi"`
		Tags    []openAPITagMeta `json:"tags"`
		Paths   map[string]map[string]struct {
			Tags        []string                       `json:"tags"`
			Parameters  []openAPIParam                 `json:"parameters"`
			RequestBody openAPIRequestBodyMeta         `json:"requestBody"`
			Responses   map[string]openAPIResponseMeta `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode openapi response: %v", err)
	}
	if spec.OpenAPI == "" {
		t.Fatal("expected openapi version in /openapi.json")
	}
	if !hasOpenAPITag(spec.Tags, "characters") || !hasOpenAPITag(spec.Tags, "highscores") || !hasOpenAPITag(spec.Tags, "auctions") {
		t.Fatalf("expected docs tags to include category groups, got %#v", spec.Tags)
	}

	if _, ok := spec.Paths["/v1/news/all"]["get"]; !ok {
		t.Fatal("expected /v1/news/all GET in generated openapi paths")
	}
	charBatchPost, ok := spec.Paths["/v1/characters/batch"]["post"]
	if !ok {
		t.Fatal("expected /v1/characters/batch POST in generated openapi paths")
	}
	if len(charBatchPost.Tags) != 1 || charBatchPost.Tags[0] != "characters" {
		t.Fatalf("expected /v1/characters/batch POST to be grouped under characters tag, got %#v", charBatchPost.Tags)
	}
	if !charBatchPost.RequestBody.Required {
		t.Fatal("expected /v1/characters/batch POST to require a request body")
	}
	if _, ok := charBatchPost.RequestBody.Content["application/json"]; !ok {
		t.Fatal("expected /v1/characters/batch POST to document application/json request body")
	}
	if _, ok := charBatchPost.Responses["200"]; !ok {
		t.Fatal("expected /v1/characters/batch POST to document 200 response")
	}
	if _, ok := charBatchPost.Responses["400"]; !ok {
		t.Fatal("expected /v1/characters/batch POST to document 400 response")
	}
	if _, ok := charBatchPost.Responses["502"]; !ok {
		t.Fatal("expected /v1/characters/batch POST to document 502 response")
	}

	charComparePost, ok := spec.Paths["/v1/characters/compare"]["post"]
	if !ok {
		t.Fatal("expected /v1/characters/compare POST in generated openapi paths")
	}
	if _, ok := charComparePost.Responses["404"]; !ok {
		t.Fatal("expected /v1/characters/compare POST to document 404 response")
	}

	worldGet, ok := spec.Paths["/v1/world/{name}"]["get"]
	if !ok {
		t.Fatal("expected /v1/world/{name} GET in generated openapi paths")
	}
	if !hasOpenAPIParameter(worldGet.Parameters, "name", "path", true) {
		t.Fatal("expected /v1/world/{name} GET to include required path parameter name")
	}

	outfitGet, ok := spec.Paths["/v1/outfit"]["get"]
	if !ok {
		t.Fatal("expected /v1/outfit GET in generated openapi paths")
	}
	if len(outfitGet.Tags) != 1 || outfitGet.Tags[0] != "outfit" {
		t.Fatalf("expected /v1/outfit GET to be grouped under outfit tag, got %#v", outfitGet.Tags)
	}
	if !hasOpenAPIParameter(outfitGet.Parameters, "format", "query", false) {
		t.Fatal("expected /v1/outfit GET to include format query parameter")
	}
}

func TestRouterHighscoresRedirects(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()
	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)

	testCases := []struct {
		path     string
		location string
	}{
		{path: "/v1/highscores/Belaria", location: "/v1/highscores/Belaria/experience/all/1"},
		{path: "/v1/highscores/Belaria/magic", location: "/v1/highscores/Belaria/magic/all/1"},
		{path: "/v1/highscores/Belaria/experience/all", location: "/v1/highscores/Belaria/experience/all/1"},
		{path: "/v1/highscores/all/experience/all", location: "/v1/highscores/all/experience/all/1"},
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

func TestRouterHighscoresAllWorldsAllReturnsGroupedPayload(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()
	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)
	rec := performRequest(router, http.MethodGet, "/v1/highscores/all/experience/all/all")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	body := decodeJSONBody(t, rec)
	assertEnvelope(t, body, http.StatusOK, 0)

	highscores, ok := body["highscores"].(map[string]any)
	if !ok {
		t.Fatalf("expected highscores object, got %#v", body["highscores"])
	}
	if highscores["world"] != "all" {
		t.Fatalf("expected grouped highscores.world=all, got %#v", highscores["world"])
	}

	worlds, ok := highscores["worlds"].([]any)
	if !ok || len(worlds) == 0 {
		t.Fatalf("expected highscores.worlds array, got %#v", highscores["worlds"])
	}
	first, ok := worlds[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first grouped world payload object, got %#v", worlds[0])
	}
	if _, ok := first["highscore_list"]; !ok {
		t.Fatalf("expected grouped world payload with highscore_list, got %#v", first)
	}
}

func TestRouterValidationErrors(t *testing.T) {
	api := newHappyAPIUpstream(t)
	defer api.Close()
	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)

	testCases := []struct {
		path  string
		code  int
		error int
	}{
		{path: "/v1/world/Nope", code: http.StatusBadRequest, error: validation.ErrorWorldDoesNotExist},
		{path: "/v1/character/a", code: http.StatusBadRequest, error: validation.ErrorCharacterNameTooShort},
		{path: "/v1/outfit/a", code: http.StatusBadRequest, error: validation.ErrorCharacterNameTooShort},
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
	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)
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
			writeJSON(w, map[string]any{"error": "Character not found"})
			return
		}
		writeJSON(w, map[string]any{"error": "unsupported"})
	})
	defer api.Close()
	cdpSrv := newMockCDPForRouter(t, api)
	defer cdpSrv.Close()
	fs := newFakeFlareSolverrForRouter(t, eventsFixtureHTML)
	defer fs.Close()

	router := newIntegrationTestRouter(t, fs.URL, api.URL, cdpSrv.URL)
	rec := performRequest(router, http.MethodGet, "/v1/character/Unknown%20Player")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	body := decodeJSONBody(t, rec)
	assertEnvelope(t, body, http.StatusNotFound, validation.ErrorEntityNotFound)
}

func newIntegrationTestRouter(t *testing.T, flaresolverrURL, baseURL, cdpURL string) http.Handler {
	t.Helper()
	t.Setenv("FLARESOLVERR_URL", flaresolverrURL)
	t.Setenv("RUBINOT_BASE_URL", baseURL)
	t.Setenv("SCRAPE_MAX_TIMEOUT_MS", "120000")
	t.Setenv("SCRAPE_MAX_CONCURRENCY", "8")
	t.Setenv("CDP_URL", cdpURL)

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
			Cmd     string `json:"cmd"`
			URL     string `json:"url"`
			Session string `json:"session"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode flaresolverr request: %v", err)
		}

		if payload.Cmd == "sessions.create" {
			writeJSON(w, map[string]any{"status": "ok", "message": "Session created successfully.", "session": payload.Session})
			return
		}

		if payload.Session != "" && !strings.Contains(payload.URL, "/events") {
			writeJSON(w, map[string]any{
				"status":  "ok",
				"message": "Challenge not detected!",
				"solution": map[string]any{
					"response": "<html><body>ok</body></html>",
					"status":   http.StatusOK,
					"url":      payload.URL,
				},
			})
			return
		}

		body := "<html><body>ok</body></html>"
		solutionStatus := http.StatusOK
		if strings.Contains(payload.URL, "/events") {
			body = eventsHTML
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
		case r.URL.Path == "/api/boosted":
			writeJSON(w, map[string]any{
				"boss":    map[string]any{"id": 1226, "name": "Eradicator", "looktype": 875},
				"monster": map[string]any{"id": 1145, "name": "Vicious Squire", "looktype": 131},
			})
		case r.URL.Path == "/api/maintenance":
			writeJSON(w, map[string]any{"isClosed": false, "closeMessage": "Server is under maintenance, please visit later."})
		case r.URL.Path == "/api/geo-language":
			writeJSON(w, map[string]any{"language": "pt", "countryCode": "BR"})
		case r.URL.Path == "/api/events/calendar":
			writeJSON(w, map[string]any{
				"events": []map[string]any{
					{
						"id":                 9,
						"name":               "Gaz'Haragoth",
						"description":        "Boss spawn",
						"colorDark":          "#735D10",
						"colorLight":         "#8B6D05",
						"displayPriority":    5,
						"specialEffect":      nil,
						"startDate":          nil,
						"endDate":            nil,
						"isRecurring":        true,
						"recurringWeekdays":  nil,
						"recurringMonthDays": []int{1, 15},
						"recurringStart":     "2026-02-01T16:00:00.000Z",
						"recurringEnd":       "2026-04-30T16:00:00.000Z",
						"tags":               []string{"boss"},
					},
				},
				"eventsByDay": map[string]any{
					"1": []map[string]any{
						{
							"id":                 9,
							"name":               "Gaz'Haragoth",
							"description":        "Boss spawn",
							"colorDark":          "#735D10",
							"colorLight":         "#8B6D05",
							"displayPriority":    5,
							"specialEffect":      nil,
							"startDate":          nil,
							"endDate":            nil,
							"isRecurring":        true,
							"recurringWeekdays":  nil,
							"recurringMonthDays": []int{1, 15},
							"recurringStart":     "2026-02-01T16:00:00.000Z",
							"recurringEnd":       "2026-04-30T16:00:00.000Z",
							"tags":               []string{"boss"},
						},
					},
				},
				"month": 2,
				"year":  2026,
			})
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
		case r.URL.Path == "/api/outfit":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(testPNGBytes(t))
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

func hasOpenAPIParameter(params []openAPIParam, name, in string, required bool) bool {
	for _, param := range params {
		if param.Name == name && param.In == in && param.Required == required {
			return true
		}
	}
	return false
}

func hasOpenAPITag(tags []openAPITagMeta, name string) bool {
	for _, tag := range tags {
		if tag.Name == name {
			return true
		}
	}
	return false
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

var routerCDPPathRe = regexp.MustCompile(`fetch\('([^']+)'\)`)

var routerWSUpgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func newMockCDPForRouter(t *testing.T, apiUpstream *httptest.Server) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/json/list", func(w http.ResponseWriter, r *http.Request) {
		targets := []map[string]string{{
			"id":                   "ROUTER_PAGE_1",
			"type":                 "page",
			"url":                  "https://rubinot.com.br/news",
			"webSocketDebuggerUrl": fmt.Sprintf("ws://%s/devtools/page/ROUTER_PAGE_1", r.Host),
		}}
		writeJSON(w, targets)
	})

	mux.HandleFunc("/devtools/page/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := routerWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("websocket upgrade: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req struct {
				ID     int64  `json:"id"`
				Method string `json:"method"`
				Params struct {
					Expression string `json:"expression"`
				} `json:"params"`
			}
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			var value string
			if req.Method == "Runtime.evaluate" {
				matches := routerCDPPathRe.FindAllStringSubmatch(req.Params.Expression, -1)
				if len(matches) > 0 {
					isBatch := strings.Contains(req.Params.Expression, "Promise.allSettled")
					if len(matches) == 1 && !isBatch {
						resp, err := http.Get(apiUpstream.URL + matches[0][1])
						if err != nil {
							value = fmt.Sprintf(`{"error":"%s"}`, err.Error())
						} else {
							defer resp.Body.Close()
							raw, _ := io.ReadAll(resp.Body)
							if strings.Contains(req.Params.Expression, "bodyBase64") {
								payload := map[string]any{
									"status":      resp.StatusCode,
									"contentType": resp.Header.Get("Content-Type"),
									"bodyBase64":  base64.StdEncoding.EncodeToString(raw),
								}
								encoded, _ := json.Marshal(payload)
								value = string(encoded)
							} else {
								wrapper := map[string]any{"ok": true, "status": resp.StatusCode, "body": string(raw)}
								encoded, _ := json.Marshal(wrapper)
								value = string(encoded)
							}
						}
					} else {
						results := make([]map[string]string, 0, len(matches))
						for _, match := range matches {
							resp, err := http.Get(apiUpstream.URL + match[1])
							if err != nil {
								results = append(results, map[string]string{
									"status": "rejected",
									"value":  err.Error(),
								})
								continue
							}

							raw, readErr := io.ReadAll(resp.Body)
							resp.Body.Close()
							if readErr != nil {
								results = append(results, map[string]string{
									"status": "rejected",
									"value":  readErr.Error(),
								})
								continue
							}

							results = append(results, map[string]string{
								"status": "fulfilled",
								"value":  string(raw),
							})
						}

						raw, _ := json.Marshal(results)
						value = string(raw)
					}
				}
			}

			conn.WriteJSON(map[string]any{
				"id": req.ID,
				"result": map[string]any{
					"result": map[string]any{
						"type":  "string",
						"value": value,
					},
				},
			})
		}
	})

	return httptest.NewServer(mux)
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

func testPNGBytes(t *testing.T) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	opaque := color.NRGBA{R: 33, G: 122, B: 244, A: 255}
	img.Set(0, 0, opaque)
	img.Set(1, 0, opaque)
	img.Set(0, 1, opaque)
	img.Set(1, 1, opaque)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png fixture: %v", err)
	}
	return buf.Bytes()
}

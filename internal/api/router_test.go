package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

type fakeFlareSolverrReply struct {
	HTTPStatus   int
	Status       string
	Message      string
	TargetStatus int
	HTML         string
}

func TestRouterIntegrationHappyPaths(t *testing.T) {
	happyResponder := newHappyFlareSolverrResponder(t)
	flaresolverrServer, _ := newFakeFlareSolverrServer(t, happyResponder)
	defer flaresolverrServer.Close()

	router := newIntegrationTestRouter(t, flaresolverrServer.URL)

	testCases := []struct {
		name       string
		path       string
		payloadKey string
	}{
		{name: "E1 worlds", path: "/v1/worlds", payloadKey: "worlds"},
		{name: "E2 world", path: "/v1/world/Belaria", payloadKey: "world"},
		{name: "E3 character", path: "/v1/character/Test%20Character", payloadKey: "character"},
		{name: "E4 guild", path: "/v1/guild/Test%20Guild", payloadKey: "guild"},
		{name: "E5 guilds", path: "/v1/guilds/Belaria", payloadKey: "guilds"},
		{name: "E6 houses", path: "/v1/houses/Belaria/Venore", payloadKey: "houses"},
		{name: "E7 house", path: "/v1/house/Belaria/50", payloadKey: "house"},
		{name: "E8 highscores", path: "/v1/highscores/Belaria/experience/all/1", payloadKey: "highscores"},
		{name: "E9 killstatistics", path: "/v1/killstatistics/Belaria", payloadKey: "killstatistics"},
		{name: "E10 news by id", path: "/v1/news/id/140", payloadKey: "news"},
		{name: "E11 news archive", path: "/v1/news/archive?days=90", payloadKey: "newslist"},
		{name: "E11 news latest", path: "/v1/news/latest", payloadKey: "newslist"},
		{name: "E11 news newsticker", path: "/v1/news/newsticker", payloadKey: "newslist"},
		{name: "E12 deaths", path: "/v1/deaths/Belaria", payloadKey: "deaths"},
		{name: "E13 transfers", path: "/v1/transfers", payloadKey: "transfers"},
		{name: "E14 banishments", path: "/v1/banishments/Belaria", payloadKey: "banishments"},
		{name: "E15 events", path: "/v1/events/schedule", payloadKey: "events"},
		{name: "E16 auctions current", path: "/v1/auctions/current/1", payloadKey: "auctions"},
		{name: "E17 auctions history", path: "/v1/auctions/history/1", payloadKey: "auctions"},
		{name: "E18 auction detail", path: "/v1/auctions/193226", payloadKey: "auction"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(t, router, http.MethodGet, tc.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
			}

			body := decodeJSONBody(t, rec)
			assertEnvelope(t, body, http.StatusOK, 0)
			if _, exists := body[tc.payloadKey]; !exists {
				t.Fatalf("expected payload key %q in response body: %+v", tc.payloadKey, body)
			}
		})
	}
}

func TestRouterHighscoresRedirects(t *testing.T) {
	happyResponder := newHappyFlareSolverrResponder(t)
	flaresolverrServer, _ := newFakeFlareSolverrServer(t, happyResponder)
	defer flaresolverrServer.Close()

	router := newIntegrationTestRouter(t, flaresolverrServer.URL)

	testCases := []struct {
		path     string
		location string
	}{
		{path: "/v1/highscores/Belaria", location: "/v1/highscores/Belaria/experience/all/1"},
		{path: "/v1/highscores/Belaria/magic", location: "/v1/highscores/Belaria/magic/all/1"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			rec := performRequest(t, router, http.MethodGet, tc.path)
			if rec.Code != http.StatusFound {
				t.Fatalf("expected redirect status %d, got %d", http.StatusFound, rec.Code)
			}
			if location := rec.Header().Get("Location"); location != tc.location {
				t.Fatalf("expected redirect location %q, got %q", tc.location, location)
			}
		})
	}
}

func TestRouterIntegrationValidationErrors(t *testing.T) {
	testCases := []struct {
		name          string
		path          string
		expectedError int
	}{
		{name: "E2 invalid world", path: "/v1/world/Nope", expectedError: 11001},
		{name: "E3 invalid character", path: "/v1/character/a", expectedError: 10002},
		{name: "E4 invalid guild", path: "/v1/guild/ab", expectedError: 14002},
		{name: "E5 invalid world", path: "/v1/guilds/Nope", expectedError: 11001},
		{name: "E6 invalid town", path: "/v1/houses/Belaria/UnknownTown", expectedError: 11002},
		{name: "E7 invalid house id", path: "/v1/house/Belaria/abc", expectedError: 11006},
		{name: "E8 invalid category", path: "/v1/highscores/Belaria/unknown/all/1", expectedError: 11004},
		{name: "E9 invalid world", path: "/v1/killstatistics/Nope", expectedError: 11001},
		{name: "E10 invalid news id", path: "/v1/news/id/abc", expectedError: 30002},
		{name: "E11 invalid archive days", path: "/v1/news/archive?days=0", expectedError: 30006},
		{name: "E12 invalid world", path: "/v1/deaths/Nope", expectedError: 11001},
		{name: "E13 invalid page", path: "/v1/transfers?page=0", expectedError: 30001},
		{name: "E14 invalid world", path: "/v1/banishments/Nope", expectedError: 11001},
		{name: "E15 invalid month", path: "/v1/events/schedule?month=13&year=2026", expectedError: 30004},
		{name: "E16 invalid page", path: "/v1/auctions/current/0", expectedError: 30001},
		{name: "E17 invalid page", path: "/v1/auctions/history/0", expectedError: 30001},
		{name: "E18 invalid id", path: "/v1/auctions/%20", expectedError: 30008},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flaresolverrServer, scrapeCalls := newFakeFlareSolverrServer(t, func(targetURL string) fakeFlareSolverrReply {
				t.Fatalf("validation path should not scrape upstream, got URL: %s", targetURL)
				return fakeFlareSolverrReply{HTML: "<html></html>"}
			})
			defer flaresolverrServer.Close()

			router := newIntegrationTestRouter(t, flaresolverrServer.URL)
			rec := performRequest(t, router, http.MethodGet, tc.path)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
			}

			body := decodeJSONBody(t, rec)
			assertEnvelope(t, body, http.StatusBadRequest, tc.expectedError)
			if scrapeCalls.Load() != 0 {
				t.Fatalf("expected zero non-bootstrap scrapes on validation failure, got %d", scrapeCalls.Load())
			}
		})
	}
}

func TestRouterIntegrationUpstreamErrors(t *testing.T) {
	flaresolverrServer, _ := newFakeFlareSolverrServer(t, func(_ string) fakeFlareSolverrReply {
		return fakeFlareSolverrReply{HTTPStatus: http.StatusInternalServerError}
	})
	defer flaresolverrServer.Close()

	router := newIntegrationTestRouter(t, flaresolverrServer.URL)

	testCases := []struct {
		name string
		path string
	}{
		{name: "E1 worlds", path: "/v1/worlds"},
		{name: "E2 world", path: "/v1/world/Belaria"},
		{name: "E3 character", path: "/v1/character/Test%20Character"},
		{name: "E4 guild", path: "/v1/guild/Test%20Guild"},
		{name: "E5 guilds", path: "/v1/guilds/Belaria"},
		{name: "E6 houses", path: "/v1/houses/Belaria/Venore"},
		{name: "E7 house", path: "/v1/house/Belaria/50"},
		{name: "E8 highscores", path: "/v1/highscores/Belaria/experience/all/1"},
		{name: "E9 killstatistics", path: "/v1/killstatistics/Belaria"},
		{name: "E10 news by id", path: "/v1/news/id/140"},
		{name: "E11 news archive", path: "/v1/news/archive?days=90"},
		{name: "E11 news latest", path: "/v1/news/latest"},
		{name: "E11 news newsticker", path: "/v1/news/newsticker"},
		{name: "E12 deaths", path: "/v1/deaths/Belaria"},
		{name: "E13 transfers", path: "/v1/transfers"},
		{name: "E14 banishments", path: "/v1/banishments/Belaria"},
		{name: "E15 events", path: "/v1/events/schedule"},
		{name: "E16 auctions current", path: "/v1/auctions/current/1"},
		{name: "E17 auctions history", path: "/v1/auctions/history/1"},
		{name: "E18 auction detail", path: "/v1/auctions/193226"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(t, router, http.MethodGet, tc.path)
			if rec.Code != http.StatusBadGateway {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadGateway, rec.Code, rec.Body.String())
			}

			body := decodeJSONBody(t, rec)
			assertEnvelope(t, body, http.StatusBadGateway, 20002)
		})
	}
}

func TestRouterIntegrationNotFound(t *testing.T) {
	happyResponder := newHappyFlareSolverrResponder(t)
	worldNotFound := mustReadFixture(t, "world/not_found.html")
	characterNotFound := mustReadFixture(t, "character/not_found.html")
	guildNotFound := mustReadFixture(t, "guild/not_found.html")
	houseNotFound := mustReadFixture(t, "house/not_found.html")
	newsNotFound := mustReadFixture(t, "news/not_found.html")
	auctionNotFound := mustReadFixture(t, "auctions/detail_not_found.html")

	flaresolverrServer, _ := newFakeFlareSolverrServer(t, func(targetURL string) fakeFlareSolverrReply {
		switch {
		case strings.Contains(targetURL, "subtopic=worlds&world="):
			return fakeFlareSolverrReply{HTML: worldNotFound}
		case strings.Contains(targetURL, "subtopic=characters"):
			return fakeFlareSolverrReply{HTML: characterNotFound}
		case strings.Contains(targetURL, "subtopic=guilds&page=view"):
			return fakeFlareSolverrReply{HTML: guildNotFound}
		case strings.Contains(targetURL, "subtopic=houses&page=view") && strings.Contains(targetURL, "houseid=999999"):
			return fakeFlareSolverrReply{HTML: houseNotFound}
		case strings.Contains(targetURL, "?news/archive/999999"):
			return fakeFlareSolverrReply{HTML: newsNotFound}
		case strings.HasSuffix(targetURL, "/?news"):
			return fakeFlareSolverrReply{HTML: newsNotFound}
		case strings.Contains(targetURL, "/?currentcharactertrades/999999"):
			return fakeFlareSolverrReply{HTML: auctionNotFound}
		case strings.Contains(targetURL, "/?pastcharactertrades/999999"):
			return fakeFlareSolverrReply{HTML: auctionNotFound}
		default:
			return happyResponder(targetURL)
		}
	})
	defer flaresolverrServer.Close()

	router := newIntegrationTestRouter(t, flaresolverrServer.URL)

	testCases := []struct {
		name string
		path string
	}{
		{name: "E2 world not found", path: "/v1/world/Belaria"},
		{name: "E3 character not found", path: "/v1/character/Ghost%20Player"},
		{name: "E4 guild not found", path: "/v1/guild/Unknown%20Guild"},
		{name: "E7 house not found", path: "/v1/house/Belaria/999999"},
		{name: "E10 news not found", path: "/v1/news/id/999999"},
		{name: "E18 auction not found", path: "/v1/auctions/999999"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequest(t, router, http.MethodGet, tc.path)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, rec.Code, rec.Body.String())
			}

			body := decodeJSONBody(t, rec)
			assertEnvelope(t, body, http.StatusNotFound, 20004)
		})
	}
}

func newIntegrationTestRouter(t *testing.T, flaresolverrURL string) http.Handler {
	t.Helper()
	t.Setenv("FLARESOLVERR_URL", flaresolverrURL)
	t.Setenv("RUBINOT_BASE_URL", "https://www.rubinot.com.br")
	t.Setenv("SCRAPE_MAX_TIMEOUT_MS", "120000")
	t.Setenv("SCRAPE_MAX_CONCURRENCY", "8")

	router, err := NewRouter()
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	return http.Handler(router)
}

func newHappyFlareSolverrResponder(t *testing.T) func(string) fakeFlareSolverrReply {
	t.Helper()
	fixtures := map[string]string{
		"worlds":            mustReadFixture(t, "worlds/overview.html"),
		"world":             mustReadFixture(t, "world/belaria.html"),
		"character":         mustReadFixture(t, "character/normal.html"),
		"guild":             mustReadFixture(t, "guild/active.html"),
		"guilds":            mustReadFixture(t, "guilds/list.html"),
		"houses":            mustReadFixture(t, "houses/venore_list.html"),
		"guildhalls":        mustReadFixture(t, "houses/guildhalls_list.html"),
		"house":             mustReadFixture(t, "house/rented.html"),
		"highscores":        mustReadFixture(t, "highscores/experience_page1.html"),
		"killstatistics":    mustReadFixture(t, "killstatistics/normal.html"),
		"newsArticle":       mustReadFixture(t, "news/article.html"),
		"newsTicker":        mustReadFixture(t, "news/ticker.html"),
		"newsArchive":       mustReadFixture(t, "news/list.html"),
		"deaths":            mustReadFixture(t, "deaths/normal.html"),
		"transfers":         mustReadFixture(t, "transfers/normal.html"),
		"banishments":       mustReadFixture(t, "banishments/normal.html"),
		"events":            mustReadFixture(t, "events/schedule.html"),
		"auctionsCurrent":   mustReadFixture(t, "auctions/current.html"),
		"auctionsHistory":   mustReadFixture(t, "auctions/history.html"),
		"auctionDetail":     mustReadFixture(t, "auctions/detail_active.html"),
		"auctionDetailPast": mustReadFixture(t, "auctions/detail_ended.html"),
	}

	return func(targetURL string) fakeFlareSolverrReply {
		switch {
		case strings.Contains(targetURL, "subtopic=worlds&world="):
			return fakeFlareSolverrReply{HTML: fixtures["world"]}
		case strings.Contains(targetURL, "subtopic=worlds"):
			return fakeFlareSolverrReply{HTML: fixtures["worlds"]}
		case strings.Contains(targetURL, "subtopic=characters"):
			return fakeFlareSolverrReply{HTML: fixtures["character"]}
		case strings.Contains(targetURL, "subtopic=guilds&page=view"):
			return fakeFlareSolverrReply{HTML: fixtures["guild"]}
		case strings.Contains(targetURL, "subtopic=guilds"):
			return fakeFlareSolverrReply{HTML: fixtures["guilds"]}
		case strings.Contains(targetURL, "subtopic=houses&page=view"):
			if strings.Contains(targetURL, "houseid=50") {
				return fakeFlareSolverrReply{HTML: fixtures["house"]}
			}
			return fakeFlareSolverrReply{HTML: fixtures["house"]}
		case strings.Contains(targetURL, "subtopic=houses") && strings.Contains(targetURL, "type=guildhalls"):
			return fakeFlareSolverrReply{HTML: fixtures["guildhalls"]}
		case strings.Contains(targetURL, "subtopic=houses"):
			return fakeFlareSolverrReply{HTML: fixtures["houses"]}
		case strings.Contains(targetURL, "subtopic=highscores"):
			return fakeFlareSolverrReply{HTML: fixtures["highscores"]}
		case strings.Contains(targetURL, "subtopic=killstatistics"):
			return fakeFlareSolverrReply{HTML: fixtures["killstatistics"]}
		case strings.Contains(targetURL, "?news/archive/"):
			return fakeFlareSolverrReply{HTML: fixtures["newsArticle"]}
		case strings.HasSuffix(targetURL, "/?news/archive"):
			return fakeFlareSolverrReply{HTML: fixtures["newsArchive"]}
		case strings.HasSuffix(targetURL, "/?news"):
			return fakeFlareSolverrReply{HTML: fixtures["newsTicker"]}
		case strings.Contains(targetURL, "subtopic=latestdeaths&world="):
			return fakeFlareSolverrReply{HTML: fixtures["deaths"]}
		case strings.Contains(targetURL, "subtopic=transferstatistics"):
			return fakeFlareSolverrReply{HTML: fixtures["transfers"]}
		case strings.Contains(targetURL, "subtopic=bans"):
			return fakeFlareSolverrReply{HTML: fixtures["banishments"]}
		case strings.Contains(targetURL, "subtopic=eventcalendar"):
			return fakeFlareSolverrReply{HTML: fixtures["events"]}
		case strings.Contains(targetURL, "/?currentcharactertrades/"):
			return fakeFlareSolverrReply{HTML: fixtures["auctionDetail"]}
		case strings.Contains(targetURL, "/?pastcharactertrades/"):
			return fakeFlareSolverrReply{HTML: fixtures["auctionDetailPast"]}
		case strings.Contains(targetURL, "/currentcharactertrades") || strings.Contains(targetURL, "subtopic=currentcharactertrades"):
			return fakeFlareSolverrReply{HTML: fixtures["auctionsCurrent"]}
		case strings.Contains(targetURL, "/pastcharactertrades") || strings.Contains(targetURL, "subtopic=pastcharactertrades"):
			return fakeFlareSolverrReply{HTML: fixtures["auctionsHistory"]}
		default:
			t.Fatalf("unexpected upstream URL requested in integration test: %s", targetURL)
			return fakeFlareSolverrReply{HTML: "<html></html>"}
		}
	}
}

func newFakeFlareSolverrServer(t *testing.T, responder func(string) fakeFlareSolverrReply) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	type requestPayload struct {
		URL string `json:"url"`
	}

	nonBootstrapCalls := &atomic.Int64{}
	bootstrapHTML := `<html><body><select name="world"><option value="0">Select</option><option value="15">Belaria</option></select></body></html>`
	bootstrapHousesHTML := `<html><body><label><input type="radio" name="town" value="1">Venore</label><label><input type="radio" name="town" value="2">Thais</label></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected flaresolverr method: %s", r.Method)
		}

		var payload requestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode flaresolverr request: %v", err)
		}

		targetURL := payload.URL
		if strings.Contains(targetURL, "subtopic=latestdeaths") && !strings.Contains(targetURL, "&world=") {
			writeFakeFlareSolverrResponse(t, w, targetURL, fakeFlareSolverrReply{HTML: bootstrapHTML})
			return
		}
		if strings.Contains(targetURL, "subtopic=houses") && !strings.Contains(targetURL, "world=") {
			writeFakeFlareSolverrResponse(t, w, targetURL, fakeFlareSolverrReply{HTML: bootstrapHousesHTML})
			return
		}

		nonBootstrapCalls.Add(1)
		reply := responder(targetURL)
		writeFakeFlareSolverrResponse(t, w, targetURL, reply)
	}))

	return server, nonBootstrapCalls
}

func writeFakeFlareSolverrResponse(t *testing.T, w http.ResponseWriter, targetURL string, reply fakeFlareSolverrReply) {
	t.Helper()

	httpStatus := reply.HTTPStatus
	if httpStatus == 0 {
		httpStatus = http.StatusOK
	}

	if httpStatus != http.StatusOK {
		w.WriteHeader(httpStatus)
		if _, err := w.Write([]byte(`{"status":"error"}`)); err != nil {
			t.Fatalf("failed to write fake flaresolverr error body: %v", err)
		}
		return
	}

	status := strings.TrimSpace(reply.Status)
	if status == "" {
		status = "ok"
	}
	targetStatus := reply.TargetStatus
	if targetStatus == 0 {
		targetStatus = http.StatusOK
	}

	response := map[string]any{
		"status":  status,
		"message": reply.Message,
		"solution": map[string]any{
			"response": reply.HTML,
			"status":   targetStatus,
			"url":      targetURL,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("failed to encode fake flaresolverr response: %v", err)
	}
}

func performRequest(t *testing.T, handler http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
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

func mustReadFixture(t *testing.T, relativePath string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file for fixture lookup")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	fixturePath := filepath.Join(repoRoot, "testdata", relativePath)
	bytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixturePath, err)
	}
	return string(bytes)
}

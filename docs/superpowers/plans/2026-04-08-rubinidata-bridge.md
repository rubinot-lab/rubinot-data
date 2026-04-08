# rubinidata.com Bridge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Use api.rubinidata.com as upstream instead of rubinot.com.br to bypass Cloudflare blocking, while keeping the same response format for all consumers.

**Architecture:** A RubinidataClient sits behind CachedFetcher. When `UPSTREAM_PROVIDER=rubinidata`, CachedFetcher routes fetches to RubinidataClient (plain HTTP) instead of CDP. The client translates upstream paths (`/api/worlds` → `/v1/worlds`), fetches from rubinidata.com, and adapts the response JSON to match rubinot.com.br's format. All existing V2Fetch functions, parsers, and domain models stay unchanged.

**Tech Stack:** Go 1.23, net/http, encoding/json

**Branch:** Throwaway branch `rubinidata-bridge` — deploy from it, don't merge to main.

**Design doc:** `docs/plans/2026-04-08-rubinidata-bridge-design.md`

---

## File Structure

```
internal/scraper/
├── rubinidata_client.go       # NEW: HTTP client, path translation, orchestration
├── rubinidata_adapters.go     # NEW: Response format adapters (rubinidata → rubinot.com.br shape)
├── rubinidata_client_test.go  # NEW: Tests for path translation + adapters
├── cached_fetcher.go          # MODIFY: Add rubinidata provider routing
├── optimized_client.go        # MODIFY: Support rubinidata for batch operations
├── cdp_pool.go                # MODIFY: Make Init() optional when using rubinidata

internal/api/
├── router.go                  # MODIFY: Skip CDP init when provider=rubinidata, readiness always ready
```

Key design decision: The adapter layer produces JSON strings that match the exact format of rubinot.com.br's `/api/*` responses. This means the existing `worldsAPIResponse`, `characterAPIResponse`, etc. structs and all `map*Response` functions stay completely untouched. The RubinidataClient is invisible to everything above CachedFetcher.

---

### Task 1: Create throwaway branch

**Files:**
- None (git operation)

- [ ] **Step 1: Create and push the branch**

```bash
git checkout -b rubinidata-bridge
git push -u origin rubinidata-bridge
```

- [ ] **Step 2: Verify branch**

Run: `git branch --show-current`
Expected: `rubinidata-bridge`

---

### Task 2: RubinidataClient — HTTP client and path translation

This is the core client that makes plain HTTP requests to api.rubinidata.com and translates rubinot.com.br API paths to rubinidata.com paths.

**Files:**
- Create: `internal/scraper/rubinidata_client.go`
- Create: `internal/scraper/rubinidata_client_test.go`

- [ ] **Step 1: Write failing test for path translation**

The client receives paths like `/api/worlds` (rubinot.com.br format) and must translate to `/v1/worlds` (rubinidata.com format). This is the most critical piece — get the routing right.

```go
// internal/scraper/rubinidata_client_test.go
package scraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslatePath(t *testing.T) {
	c := &RubinidataClient{worldIDToName: map[int]string{1: "Elysian", 2: "Bellum"}}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"worlds list", "/api/worlds", "/v1/worlds"},
		{"world detail", "/api/worlds/Elysian", "/v1/world/Elysian"},
		{"world detail encoded", "/api/worlds/Ascended%20Belaria", "/v1/world/Ascended%20Belaria"},
		{"character search", "/api/characters/search?name=Bubble", "/v1/characters/Bubble"},
		{"character search encoded", "/api/characters/search?name=Blindao+Full+Rage", "/v1/characters/Blindao%20Full%20Rage"},
		{"guilds list", "/api/guilds?world=1&page=2", "/v1/guilds/Elysian?page=2"},
		{"guild detail", "/api/guilds/Falange", "/v1/guild/Falange"},
		{"guild detail encoded", "/api/guilds/Ascended%20Belaria", "/v1/guild/Ascended%20Belaria"},
		{"highscores", "/api/highscores?world=1&category=experience&vocation=5", "/v1/highscores?world=Elysian&category=experience&vocation=1"},
		{"highscores all worlds", "/api/highscores?category=experience&vocation=0", "/v1/highscores?world=all&category=experience&vocation=0"},
		{"killstats", "/api/killstats?world=1", "/v1/killstatistics/Elysian"},
		{"deaths", "/api/deaths?world=1&page=2", "/v1/deaths/Elysian?page=2"},
		{"deaths no page", "/api/deaths?world=1", "/v1/deaths/Elysian"},
		{"bans", "/api/bans?world=1&page=1", "/v1/banishments/Elysian?page=1"},
		{"transfers", "/api/transfers?page=2", "/v1/transfers?page=2"},
		{"boosted", "/api/boosted", "/v1/boosted"},
		{"highscores categories", "/api/highscores/categories", "/v1/highscores/categories"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.translatePath(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTranslateVocationID(t *testing.T) {
	// rubinot.com.br vocation IDs → rubinidata.com vocation IDs
	// rubinot: 0=all, 1=none, 2=sorcerer, 3=druid, 4=paladin, 5=knight, 9=monk
	// rubinidata: 0=all, 1=knight, 2=druid, 3=paladin, 4=sorcerer, 5=monk
	tests := []struct {
		upstream   string
		rubinidata string
	}{
		{"0", "0"},
		{"5", "1"},  // knight
		{"3", "2"},  // druid
		{"4", "3"},  // paladin
		{"2", "4"},  // sorcerer
		{"9", "5"},  // monk
	}

	for _, tt := range tests {
		t.Run(tt.upstream+"->"+tt.rubinidata, func(t *testing.T) {
			assert.Equal(t, tt.rubinidata, translateVocationToRubinidata(tt.upstream))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scraper/ -run TestTranslatePath -v`
Expected: FAIL — `RubinidataClient` not defined

- [ ] **Step 3: Write the RubinidataClient with path translation**

```go
// internal/scraper/rubinidata_client.go
package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type RubinidataClient struct {
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	worldIDToName map[int]string
}

func NewRubinidataClient() *RubinidataClient {
	baseURL := os.Getenv("RUBINIDATA_URL")
	if baseURL == "" {
		baseURL = "https://api.rubinidata.com"
	}
	return &RubinidataClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  os.Getenv("RUBINIDATA_API_KEY"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		worldIDToName: make(map[int]string),
	}
}

func (c *RubinidataClient) SetWorldMapping(m map[int]string) {
	c.worldIDToName = m
}

func IsRubinidataProvider() bool {
	return strings.EqualFold(os.Getenv("UPSTREAM_PROVIDER"), "rubinidata")
}

func (c *RubinidataClient) Fetch(ctx context.Context, upstreamPath string) (string, error) {
	translatedPath, err := c.translatePath(upstreamPath)
	if err != nil {
		return "", fmt.Errorf("translate path %s: %w", upstreamPath, err)
	}

	fullURL := c.baseURL + translatedPath

	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			return "", fmt.Errorf("build request: %w", err)
		}
		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("rubinidata fetch %s: %w", translatedPath, err)
			log.Printf("[rubinidata] attempt %d/%d failed for %s: %v", attempt+1, maxRetries, translatedPath, err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("rubinidata read body: %w", readErr)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rubinidata rate limited for %s", translatedPath)
			log.Printf("[rubinidata] rate limited on %s, retrying...", translatedPath)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("rubinidata 404 for %s", translatedPath)
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("rubinidata HTTP %d for %s: %s", resp.StatusCode, translatedPath, string(body[:min(len(body), 200)]))
			continue
		}

		adapted, adaptErr := adaptResponse(upstreamPath, string(body))
		if adaptErr != nil {
			return "", fmt.Errorf("adapt response for %s: %w", upstreamPath, adaptErr)
		}

		return adapted, nil
	}

	return "", lastErr
}

// translatePath converts a rubinot.com.br /api/* path to rubinidata.com /v1/* path.
//
// Examples:
//   /api/worlds              → /v1/worlds
//   /api/worlds/Elysian      → /v1/world/Elysian
//   /api/characters/search?name=Bubble → /v1/characters/Bubble
//   /api/guilds?world=1&page=2 → /v1/guilds/Elysian?page=2
//   /api/guilds/GuildName    → /v1/guild/GuildName
//   /api/highscores?world=1&category=experience&vocation=5 → /v1/highscores?world=Elysian&category=experience&vocation=1
//   /api/killstats?world=1   → /v1/killstatistics/Elysian
//   /api/deaths?world=1&page=2 → /v1/deaths/Elysian?page=2
//   /api/bans?world=1&page=1 → /v1/banishments/Elysian?page=1
//   /api/transfers?page=2    → /v1/transfers?page=2
//   /api/boosted             → /v1/boosted
func (c *RubinidataClient) translatePath(upstreamPath string) (string, error) {
	parsed, err := url.Parse(upstreamPath)
	if err != nil {
		return "", fmt.Errorf("parse upstream path: %w", err)
	}
	path := parsed.Path
	query := parsed.Query()

	switch {
	case path == "/api/worlds":
		return "/v1/worlds", nil

	case strings.HasPrefix(path, "/api/worlds/"):
		worldName := strings.TrimPrefix(path, "/api/worlds/")
		return "/v1/world/" + worldName, nil

	case path == "/api/characters/search":
		name := query.Get("name")
		if name == "" {
			return "", fmt.Errorf("character search missing name param")
		}
		return "/v1/characters/" + url.PathEscape(name), nil

	case path == "/api/guilds" && query.Get("world") != "":
		worldName := c.resolveWorldID(query.Get("world"))
		q := url.Values{}
		if p := query.Get("page"); p != "" {
			q.Set("page", p)
		}
		result := "/v1/guilds/" + url.PathEscape(worldName)
		if len(q) > 0 {
			result += "?" + q.Encode()
		}
		return result, nil

	case strings.HasPrefix(path, "/api/guilds/"):
		guildName := strings.TrimPrefix(path, "/api/guilds/")
		return "/v1/guild/" + guildName, nil

	case path == "/api/highscores/categories":
		return "/v1/highscores/categories", nil

	case path == "/api/highscores":
		q := url.Values{}
		if w := query.Get("world"); w != "" {
			q.Set("world", c.resolveWorldID(w))
		} else {
			q.Set("world", "all")
		}
		if cat := query.Get("category"); cat != "" {
			q.Set("category", cat)
		}
		if voc := query.Get("vocation"); voc != "" {
			q.Set("vocation", translateVocationToRubinidata(voc))
		}
		if p := query.Get("page"); p != "" {
			q.Set("page", p)
		}
		return "/v1/highscores?" + q.Encode(), nil

	case path == "/api/killstats":
		worldName := c.resolveWorldID(query.Get("world"))
		return "/v1/killstatistics/" + url.PathEscape(worldName), nil

	case path == "/api/deaths":
		worldName := c.resolveWorldID(query.Get("world"))
		q := url.Values{}
		if p := query.Get("page"); p != "" {
			q.Set("page", p)
		}
		result := "/v1/deaths/" + url.PathEscape(worldName)
		if len(q) > 0 {
			result += "?" + q.Encode()
		}
		return result, nil

	case path == "/api/bans":
		worldName := c.resolveWorldID(query.Get("world"))
		q := url.Values{}
		if p := query.Get("page"); p != "" {
			q.Set("page", p)
		}
		result := "/v1/banishments/" + url.PathEscape(worldName)
		if len(q) > 0 {
			result += "?" + q.Encode()
		}
		return result, nil

	case path == "/api/transfers":
		q := url.Values{}
		if p := query.Get("page"); p != "" {
			q.Set("page", p)
		}
		if w := query.Get("world"); w != "" {
			q.Set("from", c.resolveWorldID(w))
		}
		result := "/v1/transfers"
		if len(q) > 0 {
			result += "?" + q.Encode()
		}
		return result, nil

	case path == "/api/boosted":
		return "/v1/boosted", nil

	default:
		return "", fmt.Errorf("unsupported upstream path: %s", upstreamPath)
	}
}

func (c *RubinidataClient) resolveWorldID(idStr string) string {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return idStr
	}
	if name, ok := c.worldIDToName[id]; ok {
		return name
	}
	return idStr
}

// translateVocationToRubinidata maps rubinot.com.br vocation IDs to rubinidata.com vocation IDs.
// rubinot.com.br: 0=all, 1=none, 2=sorcerer, 3=druid, 4=paladin, 5=knight, 9=monk
// rubinidata.com: 0=all, 1=knight, 2=druid, 3=paladin, 4=sorcerer, 5=monk
func translateVocationToRubinidata(upstreamVoc string) string {
	switch upstreamVoc {
	case "0":
		return "0"
	case "5":
		return "1" // knight
	case "3":
		return "2" // druid
	case "4":
		return "3" // paladin
	case "2":
		return "4" // sorcerer
	case "9":
		return "5" // monk
	default:
		return "0"
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/scraper/ -run "TestTranslatePath|TestTranslateVocation" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scraper/rubinidata_client.go internal/scraper/rubinidata_client_test.go
git commit -m "feat(rubinidata): add client with path translation and retry logic"
```

---

### Task 3: Response adapters — worlds, boosted, killstatistics

These are the simplest adapters. Each takes rubinidata.com's JSON response and produces a JSON string matching rubinot.com.br's format (the format that existing `worldsAPIResponse`, `boostedAPIResponse`, `killstatisticsAPIResponse` structs expect).

**Files:**
- Create: `internal/scraper/rubinidata_adapters.go`
- Modify: `internal/scraper/rubinidata_client_test.go`

- [ ] **Step 1: Write failing test for worlds adapter**

```go
// Add to internal/scraper/rubinidata_client_test.go
func TestAdaptWorldsResponse(t *testing.T) {
	rubinidataJSON := `{
		"worlds": {
			"overview": {
				"total_players_online": 12902,
				"overall_maximum": 27884,
				"maximum_date": "Feb 08 2026, 17:50:57 BRT"
			},
			"regular_worlds": [
				{"name": "Elysian", "players_online": 1313, "pvp_type": "no-pvp",
				 "pvp_type_label": "Optional PvP", "world_type": "yellow", "locked": false, "id": 0}
			]
		}
	}`

	adapted, err := adaptWorldsResponse(rubinidataJSON)
	require.NoError(t, err)

	var result worldsAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	assert.Equal(t, 12902, result.TotalOnline)
	assert.Equal(t, 27884, result.OverallRecord)
	assert.True(t, result.OverallRecordTime > 0)
	require.Len(t, result.Worlds, 1)
	assert.Equal(t, "Elysian", result.Worlds[0].Name)
	assert.Equal(t, 1313, result.Worlds[0].PlayersOnline)
	assert.Equal(t, "no-pvp", result.Worlds[0].PVPType)
	assert.Equal(t, "Optional PvP", result.Worlds[0].PVPTypeLabel)
	assert.Equal(t, "yellow", result.Worlds[0].WorldType)
	assert.False(t, result.Worlds[0].Locked)
}

func TestAdaptBoostedResponse(t *testing.T) {
	rubinidataJSON := `{
		"boosted": {
			"creature": {"name": "Twisted Shaper", "id": 1322, "looktype": 932,
				"image_url": "/v1/outfit?type=932"},
			"boss": {"name": "Tropical Desolator", "id": 2683, "looktype": 1589,
				"image_url": "/v1/outfit?type=1589"}
		}
	}`

	adapted, err := adaptBoostedResponse(rubinidataJSON)
	require.NoError(t, err)

	var result boostedAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	assert.Equal(t, "Twisted Shaper", result.Monster.Name)
	assert.Equal(t, 932, result.Monster.LookType)
	assert.Equal(t, 1322, result.Monster.ID)
	assert.Equal(t, "Tropical Desolator", result.Boss.Name)
	assert.Equal(t, 1589, result.Boss.LookType)
}

func TestAdaptKillstatisticsResponse(t *testing.T) {
	rubinidataJSON := `{
		"killstatistics": {
			"entries": [
				{"race": "Demon", "killed_players_last_day": 10, "killed_by_players_last_day": 200,
				 "killed_players_last_week": 50, "killed_by_players_last_week": 1400}
			]
		}
	}`

	adapted, err := adaptKillstatisticsResponse(rubinidataJSON)
	require.NoError(t, err)

	var result killstatisticsAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	require.Len(t, result.Entries, 1)
	assert.Equal(t, "Demon", result.Entries[0].RaceName)
	assert.Equal(t, 10, result.Entries[0].PlayersKilled24h)
	assert.Equal(t, 200, result.Entries[0].CreaturesKilled24h)
	assert.Equal(t, 50, result.Entries[0].PlayersKilled7d)
	assert.Equal(t, 1400, result.Entries[0].CreaturesKilled7d)
	assert.Equal(t, 10, result.Totals.PlayersKilled24h)
	assert.Equal(t, 200, result.Totals.CreaturesKilled24h)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/scraper/ -run "TestAdaptWorlds|TestAdaptBoosted|TestAdaptKillstatistics" -v`
Expected: FAIL — adapter functions not defined

- [ ] **Step 3: Implement the adapters**

The pattern: parse rubinidata JSON into a local intermediate struct, transform fields, marshal back to JSON matching the upstream API response shape.

```go
// internal/scraper/rubinidata_adapters.go
package scraper

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// adaptResponse is the top-level router that picks the right adapter based on the upstream path.
func adaptResponse(upstreamPath string, rubinidataBody string) (string, error) {
	path := upstreamPath
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	switch {
	case path == "/api/worlds":
		return adaptWorldsResponse(rubinidataBody)
	case strings.HasPrefix(path, "/api/worlds/"):
		return adaptWorldDetailResponse(rubinidataBody)
	case path == "/api/characters/search":
		return adaptCharacterResponse(rubinidataBody)
	case path == "/api/guilds" && strings.Contains(upstreamPath, "world="):
		return adaptGuildsListResponse(rubinidataBody)
	case strings.HasPrefix(path, "/api/guilds/"):
		return adaptGuildDetailResponse(rubinidataBody)
	case path == "/api/highscores":
		return adaptHighscoresResponse(rubinidataBody)
	case path == "/api/killstats":
		return adaptKillstatisticsResponse(rubinidataBody)
	case path == "/api/deaths":
		return adaptDeathsResponse(rubinidataBody)
	case path == "/api/bans":
		return adaptBanishmentsResponse(rubinidataBody)
	case path == "/api/transfers":
		return adaptTransfersResponse(rubinidataBody)
	case path == "/api/boosted":
		return adaptBoostedResponse(rubinidataBody)
	default:
		return "", fmt.Errorf("no adapter for path: %s", upstreamPath)
	}
}

// --- Worlds adapter ---

func adaptWorldsResponse(body string) (string, error) {
	var src struct {
		Worlds struct {
			Overview struct {
				TotalPlayersOnline int    `json:"total_players_online"`
				OverallMaximum     int    `json:"overall_maximum"`
				MaximumDate        string `json:"maximum_date"`
			} `json:"overview"`
			RegularWorlds []struct {
				Name          string `json:"name"`
				PlayersOnline int    `json:"players_online"`
				PVPType       string `json:"pvp_type"`
				PVPTypeLabel  string `json:"pvp_type_label"`
				WorldType     string `json:"world_type"`
				Locked        bool   `json:"locked"`
				ID            int    `json:"id"`
			} `json:"regular_worlds"`
		} `json:"worlds"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata worlds: %w", err)
	}

	type worldEntry struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		PVPType       string `json:"pvpType"`
		PVPTypeLabel  string `json:"pvpTypeLabel"`
		WorldType     string `json:"worldType"`
		Locked        bool   `json:"locked"`
		PlayersOnline int    `json:"playersOnline"`
	}

	worlds := make([]worldEntry, 0, len(src.Worlds.RegularWorlds))
	for _, w := range src.Worlds.RegularWorlds {
		worlds = append(worlds, worldEntry{
			ID: w.ID, Name: w.Name, PVPType: w.PVPType, PVPTypeLabel: w.PVPTypeLabel,
			WorldType: w.WorldType, Locked: w.Locked, PlayersOnline: w.PlayersOnline,
		})
	}

	out := struct {
		Worlds            []worldEntry `json:"worlds"`
		TotalOnline       int          `json:"totalOnline"`
		OverallRecord     int          `json:"overallRecord"`
		OverallRecordTime int64        `json:"overallRecordTime"`
	}{
		Worlds:            worlds,
		TotalOnline:       src.Worlds.Overview.TotalPlayersOnline,
		OverallRecord:     src.Worlds.Overview.OverallMaximum,
		OverallRecordTime: parseBRTDate(src.Worlds.Overview.MaximumDate),
	}

	return marshalJSON(out)
}

// --- Boosted adapter ---

func adaptBoostedResponse(body string) (string, error) {
	var src struct {
		Boosted struct {
			Creature struct {
				Name     string `json:"name"`
				ID       int    `json:"id"`
				LookType int    `json:"looktype"`
			} `json:"creature"`
			Boss struct {
				Name     string `json:"name"`
				ID       int    `json:"id"`
				LookType int    `json:"looktype"`
			} `json:"boss"`
		} `json:"boosted"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata boosted: %w", err)
	}

	out := struct {
		Boss struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			LookType int    `json:"looktype"`
		} `json:"boss"`
		Monster struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			LookType int    `json:"looktype"`
		} `json:"monster"`
	}{
		Boss:    struct{ ID int `json:"id"`; Name string `json:"name"`; LookType int `json:"looktype"` }{src.Boosted.Boss.ID, src.Boosted.Boss.Name, src.Boosted.Boss.LookType},
		Monster: struct{ ID int `json:"id"`; Name string `json:"name"`; LookType int `json:"looktype"` }{src.Boosted.Creature.ID, src.Boosted.Creature.Name, src.Boosted.Creature.LookType},
	}

	return marshalJSON(out)
}

// --- Killstatistics adapter ---

func adaptKillstatisticsResponse(body string) (string, error) {
	var src struct {
		Killstatistics struct {
			Entries []struct {
				Race                   string `json:"race"`
				KilledPlayersLastDay   int    `json:"killed_players_last_day"`
				KilledByPlayersLastDay int    `json:"killed_by_players_last_day"`
				KilledPlayersLastWeek  int    `json:"killed_players_last_week"`
				KilledByPlayersLastWeek int   `json:"killed_by_players_last_week"`
			} `json:"entries"`
		} `json:"killstatistics"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata killstatistics: %w", err)
	}

	type entry struct {
		RaceName           string `json:"race_name"`
		PlayersKilled24h   int    `json:"players_killed_24h"`
		CreaturesKilled24h int    `json:"creatures_killed_24h"`
		PlayersKilled7d    int    `json:"players_killed_7d"`
		CreaturesKilled7d  int    `json:"creatures_killed_7d"`
	}

	entries := make([]entry, 0, len(src.Killstatistics.Entries))
	var totPK24, totCK24, totPK7, totCK7 int
	for _, e := range src.Killstatistics.Entries {
		entries = append(entries, entry{
			RaceName: e.Race, PlayersKilled24h: e.KilledPlayersLastDay,
			CreaturesKilled24h: e.KilledByPlayersLastDay, PlayersKilled7d: e.KilledPlayersLastWeek,
			CreaturesKilled7d: e.KilledByPlayersLastWeek,
		})
		totPK24 += e.KilledPlayersLastDay
		totCK24 += e.KilledByPlayersLastDay
		totPK7 += e.KilledPlayersLastWeek
		totCK7 += e.KilledByPlayersLastWeek
	}

	out := struct {
		Entries []entry `json:"entries"`
		Totals  struct {
			PlayersKilled24h   int `json:"players_killed_24h"`
			CreaturesKilled24h int `json:"creatures_killed_24h"`
			PlayersKilled7d    int `json:"players_killed_7d"`
			CreaturesKilled7d  int `json:"creatures_killed_7d"`
		} `json:"totals"`
	}{Entries: entries}
	out.Totals.PlayersKilled24h = totPK24
	out.Totals.CreaturesKilled24h = totCK24
	out.Totals.PlayersKilled7d = totPK7
	out.Totals.CreaturesKilled7d = totCK7

	return marshalJSON(out)
}

// --- Helpers ---

func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal adapted response: %w", err)
	}
	return string(b), nil
}

func parseBRTDate(s string) int64 {
	// Format: "Feb 08 2026, 17:50:57 BRT"
	s = strings.TrimSuffix(strings.TrimSpace(s), " BRT")
	layouts := []string{
		"Jan 02 2006, 15:04:05",
		"Jan 2 2006, 15:04:05",
	}
	brt := time.FixedZone("BRT", -3*3600)
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, brt); err == nil {
			return t.Unix()
		}
	}
	return 0
}

func parseSimpleDate(s string) int64 {
	// Format: "Aug 17 2023" or "Dec 29 2025"
	layouts := []string{
		"Jan 02 2006",
		"Jan 2 2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.Unix()
		}
	}
	return 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/scraper/ -run "TestAdaptWorlds|TestAdaptBoosted|TestAdaptKillstatistics" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scraper/rubinidata_adapters.go internal/scraper/rubinidata_client_test.go
git commit -m "feat(rubinidata): add worlds, boosted, killstatistics adapters"
```

---

### Task 4: Response adapters — world detail, character, deaths

These are medium-complexity adapters with more field transformations.

**Files:**
- Modify: `internal/scraper/rubinidata_adapters.go`
- Modify: `internal/scraper/rubinidata_client_test.go`

- [ ] **Step 1: Write failing tests for world detail, character, deaths adapters**

```go
// Add to internal/scraper/rubinidata_client_test.go

func TestAdaptWorldDetailResponse(t *testing.T) {
	rubinidataJSON := `{
		"world": {
			"name": "Elysian", "status": "online", "players_online": 1339,
			"online_record": {"players": 3816, "date": "May 30 2024, 19:54:36 BRT"},
			"creation_date": "Aug 17 2023", "pvp_type": "no-pvp",
			"pvp_type_label": "Optional PvP", "world_type": "yellow", "locked": false, "id": 0
		},
		"players": [
			{"name": "Bubble", "level": 8, "vocation": "Knight", "vocation_id": 0}
		]
	}`

	adapted, err := adaptWorldDetailResponse(rubinidataJSON)
	require.NoError(t, err)

	var result worldDetailAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	assert.Equal(t, "Elysian", result.World.Name)
	assert.Equal(t, 1339, result.PlayersOnline)
	assert.Equal(t, 3816, result.Record)
	assert.True(t, result.RecordTime > 0)
	assert.True(t, result.World.CreationDate > 0)
	assert.Equal(t, "no-pvp", result.World.PVPType)
	require.Len(t, result.Players, 1)
	assert.Equal(t, "Bubble", result.Players[0].Name)
}

func TestAdaptCharacterResponse(t *testing.T) {
	rubinidataJSON := `{
		"characters": {
			"character": {
				"id": 0, "name": "Icraozinho", "traded": false, "level": 1160,
				"vocation": "Exalted Monk", "world_name": "Belaria", "world_id": 2,
				"sex": "male", "achievement_points": 255, "residence": "Thais",
				"last_login": "2026-04-08T19:13:45-03:00", "account_status": "Premium Account",
				"house": "Main Street 9b", "former_names": ["Blindao Full Rage", "Teexzinho"],
				"found_by_old_name": true,
				"outfit_url": "/v1/outfit?type=128&head=79&body=49&legs=49&feet=79&addons=1&direction=3&animated=0&walk=0&size=0",
				"loyalty_points": 6, "created": "2025-06-14T18:48:57-03:00"
			},
			"other_characters": [
				{"id": 0, "name": "Alt", "level": 23, "vocation": "Knight", "world_id": 1, "world_name": "Elysian"}
			]
		}
	}`

	adapted, err := adaptCharacterResponse(rubinidataJSON)
	require.NoError(t, err)

	var result characterAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	require.NotNil(t, result.Player)
	assert.Equal(t, "Icraozinho", result.Player.Name)
	assert.Equal(t, 1160, result.Player.Level)
	assert.Equal(t, "Exalted Monk", result.Player.Vocation)
	assert.Equal(t, 255, result.Player.AchievementPoints)
	assert.Equal(t, 128, result.Player.LookType)
	assert.Equal(t, 79, result.Player.LookHead)
	assert.Equal(t, 1, result.Player.LookAddons)
	assert.True(t, result.FoundByOldName)
	assert.Equal(t, []string{"Blindao Full Rage", "Teexzinho"}, result.Player.FormerNames)
	require.Len(t, result.OtherCharacters, 1)
	assert.Equal(t, "Elysian", result.OtherCharacters[0].World)
}

func TestAdaptDeathsResponse(t *testing.T) {
	rubinidataJSON := `{
		"deaths": {
			"entries": [
				{
					"name": "PlayerName", "level": 62,
					"killers": [{"name": "monster", "player": false}, {"name": "monster2", "player": false}],
					"time": "08.04.2026, 18:31:48", "datetime": "2026-04-08T18:31:48-03:00",
					"world_id": 0, "player_id": 0
				}
			]
		}
	}`

	adapted, err := adaptDeathsResponse(rubinidataJSON)
	require.NoError(t, err)

	var result deathsAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	require.Len(t, result.Deaths, 1)
	assert.Equal(t, "PlayerName", result.Deaths[0].Victim)
	assert.Equal(t, 62, result.Deaths[0].Level)
	assert.Equal(t, "monster", result.Deaths[0].KilledBy)
	assert.Equal(t, 0, result.Deaths[0].IsPlayer)
	assert.Equal(t, "monster2", result.Deaths[0].MostDamageBy)
	assert.Equal(t, 1, result.Pagination.CurrentPage)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/scraper/ -run "TestAdaptWorldDetail|TestAdaptCharacter|TestAdaptDeaths" -v`
Expected: FAIL — adapter functions not defined

- [ ] **Step 3: Implement the adapters**

Add to `internal/scraper/rubinidata_adapters.go`:

```go
// --- World detail adapter ---

func adaptWorldDetailResponse(body string) (string, error) {
	var src struct {
		World struct {
			Name          string `json:"name"`
			Status        string `json:"status"`
			PlayersOnline int    `json:"players_online"`
			OnlineRecord  struct {
				Players int    `json:"players"`
				Date    string `json:"date"`
			} `json:"online_record"`
			CreationDate string `json:"creation_date"`
			PVPType      string `json:"pvp_type"`
			PVPTypeLabel string `json:"pvp_type_label"`
			WorldType    string `json:"world_type"`
			Locked       bool   `json:"locked"`
			ID           int    `json:"id"`
		} `json:"world"`
		Players []struct {
			Name       string `json:"name"`
			Level      int    `json:"level"`
			Vocation   string `json:"vocation"`
			VocationID int    `json:"vocation_id"`
		} `json:"players"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata world detail: %w", err)
	}

	type player struct {
		Name       string `json:"name"`
		Level      int    `json:"level"`
		Vocation   string `json:"vocation"`
		VocationID int    `json:"vocationId"`
	}
	players := make([]player, 0, len(src.Players))
	for _, p := range src.Players {
		players = append(players, player{Name: p.Name, Level: p.Level, Vocation: p.Vocation, VocationID: p.VocationID})
	}

	out := struct {
		World struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			PVPType      string `json:"pvpType"`
			PVPTypeLabel string `json:"pvpTypeLabel"`
			WorldType    string `json:"worldType"`
			Locked       bool   `json:"locked"`
			CreationDate int64  `json:"creationDate"`
		} `json:"world"`
		PlayersOnline int      `json:"playersOnline"`
		Record        int      `json:"record"`
		RecordTime    int64    `json:"recordTime"`
		Players       []player `json:"players"`
	}{
		PlayersOnline: src.World.PlayersOnline,
		Record:        src.World.OnlineRecord.Players,
		RecordTime:    parseBRTDate(src.World.OnlineRecord.Date),
		Players:       players,
	}
	out.World.ID = src.World.ID
	out.World.Name = src.World.Name
	out.World.PVPType = src.World.PVPType
	out.World.PVPTypeLabel = src.World.PVPTypeLabel
	out.World.WorldType = src.World.WorldType
	out.World.Locked = src.World.Locked
	out.World.CreationDate = parseSimpleDate(src.World.CreationDate)

	return marshalJSON(out)
}

// --- Character adapter ---

func adaptCharacterResponse(body string) (string, error) {
	var src struct {
		Characters struct {
			Character struct {
				ID                int      `json:"id"`
				Name              string   `json:"name"`
				Traded            bool     `json:"traded"`
				Level             int      `json:"level"`
				Vocation          string   `json:"vocation"`
				VocationID        int      `json:"vocation_id"`
				WorldID           int      `json:"world_id"`
				WorldName         string   `json:"world_name"`
				Sex               string   `json:"sex"`
				AchievementPoints int      `json:"achievement_points"`
				Residence         string   `json:"residence"`
				LastLogin         string   `json:"last_login"`
				AccountStatus     string   `json:"account_status"`
				Comment           string   `json:"comment"`
				House             string   `json:"house"`
				FormerNames       []string `json:"former_names"`
				FoundByOldName    bool     `json:"found_by_old_name"`
				OutfitURL         string   `json:"outfit_url"`
				LoyaltyPoints     int      `json:"loyalty_points"`
				Created           string   `json:"created"`
			} `json:"character"`
			OtherCharacters []struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				Level     int    `json:"level"`
				Vocation  string `json:"vocation"`
				WorldID   int    `json:"world_id"`
				WorldName string `json:"world_name"`
			} `json:"other_characters"`
		} `json:"characters"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata character: %w", err)
	}

	c := src.Characters.Character
	looktype, lookhead, lookbody, looklegs, lookfeet, lookaddons := parseOutfitURL(c.OutfitURL)

	var createdUnix int64
	if c.Created != "" {
		if t, err := time.Parse(time.RFC3339, c.Created); err == nil {
			createdUnix = t.Unix()
		}
	}

	type otherChar struct {
		Name     string `json:"name"`
		World    string `json:"world"`
		WorldID  int    `json:"world_id"`
		Level    int    `json:"level"`
		Vocation string `json:"vocation"`
		IsOnline bool   `json:"isOnline"`
	}
	others := make([]otherChar, 0, len(src.Characters.OtherCharacters))
	for _, oc := range src.Characters.OtherCharacters {
		others = append(others, otherChar{
			Name: oc.Name, World: oc.WorldName, WorldID: oc.WorldID,
			Level: oc.Level, Vocation: oc.Vocation, IsOnline: false,
		})
	}

	formerNames := c.FormerNames
	if formerNames == nil {
		formerNames = []string{}
	}

	out := struct {
		Player *struct {
			ID                int      `json:"id"`
			AccountID         int      `json:"account_id"`
			Name              string   `json:"name"`
			Level             int      `json:"level"`
			Vocation          string   `json:"vocation"`
			VocationID        int      `json:"vocationId"`
			WorldID           int      `json:"world_id"`
			Sex               string   `json:"sex"`
			Residence         string   `json:"residence"`
			LastLogin         string   `json:"lastlogin"`
			Created           int64    `json:"created"`
			Comment           string   `json:"comment"`
			AccountCreated    int64    `json:"account_created"`
			LoyaltyPoints     int      `json:"loyalty_points"`
			IsHidden          bool     `json:"isHidden"`
			AchievementPoints int      `json:"achievementPoints"`
			VIPTime           int64    `json:"vip_time"`
			LookType          int      `json:"looktype"`
			LookHead          int      `json:"lookhead"`
			LookBody          int      `json:"lookbody"`
			LookLegs          int      `json:"looklegs"`
			LookFeet          int      `json:"lookfeet"`
			LookAddons        int      `json:"lookaddons"`
			FormerNames       []string `json:"formerNames"`
			Traded            bool     `json:"traded"`
		} `json:"player"`
		Deaths                []any      `json:"deaths"`
		OtherCharacters       []otherChar `json:"otherCharacters"`
		AccountBadges         []any      `json:"accountBadges"`
		DisplayedAchievements []any      `json:"displayedAchievements"`
		BanInfo               map[string]any `json:"banInfo"`
		FoundByOldName        bool       `json:"foundByOldName"`
	}{
		Deaths:                []any{},
		OtherCharacters:       others,
		AccountBadges:         []any{},
		DisplayedAchievements: []any{},
		BanInfo:               map[string]any{},
		FoundByOldName:        c.FoundByOldName,
	}
	out.Player = &struct {
		ID                int      `json:"id"`
		AccountID         int      `json:"account_id"`
		Name              string   `json:"name"`
		Level             int      `json:"level"`
		Vocation          string   `json:"vocation"`
		VocationID        int      `json:"vocationId"`
		WorldID           int      `json:"world_id"`
		Sex               string   `json:"sex"`
		Residence         string   `json:"residence"`
		LastLogin         string   `json:"lastlogin"`
		Created           int64    `json:"created"`
		Comment           string   `json:"comment"`
		AccountCreated    int64    `json:"account_created"`
		LoyaltyPoints     int      `json:"loyalty_points"`
		IsHidden          bool     `json:"isHidden"`
		AchievementPoints int      `json:"achievementPoints"`
		VIPTime           int64    `json:"vip_time"`
		LookType          int      `json:"looktype"`
		LookHead          int      `json:"lookhead"`
		LookBody          int      `json:"lookbody"`
		LookLegs          int      `json:"looklegs"`
		LookFeet          int      `json:"lookfeet"`
		LookAddons        int      `json:"lookaddons"`
		FormerNames       []string `json:"formerNames"`
		Traded            bool     `json:"traded"`
	}{
		Name: c.Name, Level: c.Level, Vocation: c.Vocation, VocationID: c.VocationID,
		WorldID: c.WorldID, Sex: c.Sex, Residence: c.Residence, LastLogin: c.LastLogin,
		Created: createdUnix, Comment: c.Comment, LoyaltyPoints: c.LoyaltyPoints,
		AchievementPoints: c.AchievementPoints,
		LookType: looktype, LookHead: lookhead, LookBody: lookbody,
		LookLegs: looklegs, LookFeet: lookfeet, LookAddons: lookaddons,
		FormerNames: formerNames, Traded: c.Traded,
	}

	return marshalJSON(out)
}

// --- Deaths adapter ---

func adaptDeathsResponse(body string) (string, error) {
	var src struct {
		Deaths struct {
			Entries []struct {
				Name     string `json:"name"`
				Level    int    `json:"level"`
				Killers  []struct {
					Name   string `json:"name"`
					Player bool   `json:"player"`
				} `json:"killers"`
				Time     string `json:"time"`
				DateTime string `json:"datetime"`
				WorldID  int    `json:"world_id"`
				PlayerID int    `json:"player_id"`
			} `json:"entries"`
		} `json:"deaths"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata deaths: %w", err)
	}

	type death struct {
		PlayerID           int    `json:"player_id"`
		Time               string `json:"time"`
		Level              int    `json:"level"`
		KilledBy           string `json:"killed_by"`
		IsPlayer           int    `json:"is_player"`
		MostDamageBy       string `json:"mostdamage_by"`
		MostDamageIsPlayer int    `json:"mostdamage_is_player"`
		Victim             string `json:"victim"`
		WorldID            int    `json:"world_id"`
	}

	deaths := make([]death, 0, len(src.Deaths.Entries))
	for _, e := range src.Deaths.Entries {
		d := death{
			PlayerID: e.PlayerID, Time: e.Time, Level: e.Level,
			Victim: e.Name, WorldID: e.WorldID,
		}
		if len(e.Killers) > 0 {
			d.KilledBy = e.Killers[0].Name
			if e.Killers[0].Player {
				d.IsPlayer = 1
			}
		}
		if len(e.Killers) > 1 {
			d.MostDamageBy = e.Killers[len(e.Killers)-1].Name
			if e.Killers[len(e.Killers)-1].Player {
				d.MostDamageIsPlayer = 1
			}
		}
		deaths = append(deaths, d)
	}

	out := struct {
		Deaths     []death `json:"deaths"`
		Pagination struct {
			CurrentPage  int `json:"currentPage"`
			TotalPages   int `json:"totalPages"`
			TotalCount   int `json:"totalCount"`
			ItemsPerPage int `json:"itemsPerPage"`
		} `json:"pagination"`
	}{Deaths: deaths}
	out.Pagination.CurrentPage = 1
	out.Pagination.TotalPages = 1
	out.Pagination.TotalCount = len(deaths)
	out.Pagination.ItemsPerPage = len(deaths)

	return marshalJSON(out)
}

// --- Outfit URL parser ---

func parseOutfitURL(outfitURL string) (looktype, head, body, legs, feet, addons int) {
	if outfitURL == "" {
		return
	}
	parsed, err := url.Parse(outfitURL)
	if err != nil {
		return
	}
	q := parsed.Query()
	looktype, _ = strconv.Atoi(q.Get("type"))
	head, _ = strconv.Atoi(q.Get("head"))
	body, _ = strconv.Atoi(q.Get("body"))
	legs, _ = strconv.Atoi(q.Get("legs"))
	feet, _ = strconv.Atoi(q.Get("feet"))
	addons, _ = strconv.Atoi(q.Get("addons"))
	return
}
```

Note: add `"net/url"` and `"strconv"` to the imports in `rubinidata_adapters.go`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/scraper/ -run "TestAdaptWorldDetail|TestAdaptCharacter|TestAdaptDeaths" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scraper/rubinidata_adapters.go internal/scraper/rubinidata_client_test.go
git commit -m "feat(rubinidata): add world detail, character, deaths adapters"
```

---

### Task 5: Response adapters — highscores, guilds list, guild detail, banishments, transfers

The remaining adapters to complete full Tier 1+2 coverage.

**Files:**
- Modify: `internal/scraper/rubinidata_adapters.go`
- Modify: `internal/scraper/rubinidata_client_test.go`

- [ ] **Step 1: Write failing tests**

```go
// Add to internal/scraper/rubinidata_client_test.go

func TestAdaptHighscoresResponse(t *testing.T) {
	rubinidataJSON := `{
		"highscores": {
			"category": "experience", "world": "elysian",
			"highscore_list": [
				{"rank": 1, "id": 0, "name": "Player", "vocation": "Elite Knight",
				 "world_id": 1, "world_name": "Elysian", "level": 2503, "value": 260936620480}
			]
		}
	}`

	adapted, err := adaptHighscoresResponse(rubinidataJSON)
	require.NoError(t, err)

	var result highscoresAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	require.Len(t, result.Players, 1)
	assert.Equal(t, 1, result.Players[0].Rank)
	assert.Equal(t, "Player", result.Players[0].Name)
	assert.Equal(t, 5, result.Players[0].Vocation) // Elite Knight -> 5
	assert.Equal(t, "Elysian", result.Players[0].WorldName)
	assert.Equal(t, 1, result.TotalCount)
}

func TestAdaptGuildsListResponse(t *testing.T) {
	rubinidataJSON := `{
		"guilds": {
			"guilds": [
				{"name": "TestGuild", "logo_url": "https://static.rubinot.com/guilds/guild_123.gif",
				 "description": "desc", "id": 0, "world_id": 0}
			]
		}
	}`

	adapted, err := adaptGuildsListResponse(rubinidataJSON)
	require.NoError(t, err)

	var result guildsAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	require.Len(t, result.Guilds, 1)
	assert.Equal(t, "TestGuild", result.Guilds[0].Name)
	assert.Equal(t, "guild_123.gif", result.Guilds[0].LogoName)
	assert.Equal(t, 1, result.TotalPages)
}

func TestAdaptGuildDetailResponse(t *testing.T) {
	rubinidataJSON := `{
		"guild": {
			"name": "Ascended Belaria", "world_id": 0,
			"logo_url": "https://static.rubinot.com/guilds/default.webp",
			"description": "desc", "founded": "Dec 29 2025", "active": true,
			"guild_bank_balance": "1000000",
			"members_total": 2, "members_online": 1,
			"members": [
				{"name": "Leader", "title": "", "rank": "Leader", "vocation": "Elite Knight",
				 "level": 1423, "joining_date": "Feb 10 2026", "is_online": true, "id": 0},
				{"name": "Member", "title": "", "rank": "Vice-Leader", "vocation": "Elder Druid",
				 "level": 800, "joining_date": "Jan 01 2026", "is_online": false, "id": 0}
			]
		}
	}`

	adapted, err := adaptGuildDetailResponse(rubinidataJSON)
	require.NoError(t, err)

	var result guildAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	assert.Equal(t, "Ascended Belaria", result.Guild.Name)
	assert.Equal(t, "desc", result.Guild.Description)
	assert.True(t, result.Guild.CreationData > 0)
	require.Len(t, result.Guild.Members, 2)
	assert.Equal(t, "Leader", result.Guild.Members[0].Name)
	assert.Equal(t, 5, result.Guild.Members[0].Vocation) // Elite Knight -> 5
	assert.True(t, result.Guild.Members[0].IsOnline)
	require.NotNil(t, result.Guild.Owner)
	assert.Equal(t, "Leader", result.Guild.Owner.Name)
}

func TestAdaptBanishmentsResponse(t *testing.T) {
	rubinidataJSON := `{
		"banishments": {
			"entries": [
				{"account_id": 0, "account_name": "Acc", "character": "Char",
				 "reason": "Bug abuse", "banned_at": "2026-04-01", "expires_at": "2026-05-01",
				 "banned_by": "GM", "is_permanent": false}
			]
		}
	}`

	adapted, err := adaptBanishmentsResponse(rubinidataJSON)
	require.NoError(t, err)

	var result banishmentsAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	require.Len(t, result.Bans, 1)
	assert.Equal(t, "Char", result.Bans[0].MainCharacter)
	assert.Equal(t, "Bug abuse", result.Bans[0].Reason)
	assert.Equal(t, 1, result.TotalPages)
}

func TestAdaptTransfersResponse(t *testing.T) {
	rubinidataJSON := `{
		"transfers": {
			"entries": [
				{"name": "Player", "level": 100, "from_world": "Elysian",
				 "to_world": "Bellum", "transfer_date": "2026-04-08T15:30:21-03:00"}
			]
		}
	}`

	adapted, err := adaptTransfersResponse(rubinidataJSON)
	require.NoError(t, err)

	var result transfersAPIResponse
	require.NoError(t, json.Unmarshal([]byte(adapted), &result))

	require.Len(t, result.Transfers, 1)
	assert.Equal(t, "Player", result.Transfers[0].PlayerName)
	assert.Equal(t, 100, result.Transfers[0].PlayerLevel)
	assert.Equal(t, "Elysian", result.Transfers[0].FromWorld)
	assert.Equal(t, "Bellum", result.Transfers[0].ToWorld)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/scraper/ -run "TestAdaptHighscores|TestAdaptGuildsList|TestAdaptGuildDetail|TestAdaptBanishments|TestAdaptTransfers" -v`
Expected: FAIL

- [ ] **Step 3: Implement the adapters**

Add to `internal/scraper/rubinidata_adapters.go`:

```go
// --- Highscores adapter ---

func adaptHighscoresResponse(body string) (string, error) {
	var src struct {
		Highscores struct {
			Category      string `json:"category"`
			World         string `json:"world"`
			HighscoreList []struct {
				Rank      int    `json:"rank"`
				ID        int    `json:"id"`
				Name      string `json:"name"`
				Vocation  string `json:"vocation"`
				WorldID   int    `json:"world_id"`
				WorldName string `json:"world_name"`
				Level     int    `json:"level"`
				Value     int64  `json:"value"`
			} `json:"highscore_list"`
		} `json:"highscores"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata highscores: %w", err)
	}

	type player struct {
		Rank      int    `json:"rank"`
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Level     int    `json:"level"`
		Vocation  int    `json:"vocation"`
		WorldID   int    `json:"world_id"`
		WorldName string `json:"worldName"`
		Value     int64  `json:"value"`
	}

	players := make([]player, 0, len(src.Highscores.HighscoreList))
	for _, h := range src.Highscores.HighscoreList {
		players = append(players, player{
			Rank: h.Rank, ID: h.ID, Name: h.Name, Level: h.Level,
			Vocation: vocationNameToUpstreamID(h.Vocation), WorldID: h.WorldID,
			WorldName: h.WorldName, Value: h.Value,
		})
	}

	out := struct {
		Players          []player `json:"players"`
		TotalCount       int      `json:"totalCount"`
		CachedAt         int64    `json:"cachedAt"`
		AvailableSeasons []int    `json:"availableSeasons"`
	}{
		Players:          players,
		TotalCount:       len(players),
		CachedAt:         time.Now().UnixMilli(),
		AvailableSeasons: []int{},
	}

	return marshalJSON(out)
}

// --- Guilds list adapter ---

func adaptGuildsListResponse(body string) (string, error) {
	var src struct {
		Guilds struct {
			Guilds []struct {
				Name        string `json:"name"`
				LogoURL     string `json:"logo_url"`
				Description string `json:"description"`
				ID          int    `json:"id"`
				WorldID     int    `json:"world_id"`
			} `json:"guilds"`
		} `json:"guilds"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata guilds list: %w", err)
	}

	type guild struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		WorldID     int    `json:"world_id"`
		LogoName    string `json:"logo_name"`
	}

	guilds := make([]guild, 0, len(src.Guilds.Guilds))
	for _, g := range src.Guilds.Guilds {
		guilds = append(guilds, guild{
			ID: g.ID, Name: g.Name, Description: g.Description,
			WorldID: g.WorldID, LogoName: extractFilename(g.LogoURL),
		})
	}

	out := struct {
		Guilds      []guild `json:"guilds"`
		TotalCount  int     `json:"totalCount"`
		TotalPages  int     `json:"totalPages"`
		CurrentPage int     `json:"currentPage"`
	}{
		Guilds: guilds, TotalCount: len(guilds), TotalPages: 1, CurrentPage: 1,
	}

	return marshalJSON(out)
}

// --- Guild detail adapter ---

func adaptGuildDetailResponse(body string) (string, error) {
	var src struct {
		Guild struct {
			Name             string `json:"name"`
			WorldID          int    `json:"world_id"`
			LogoURL          string `json:"logo_url"`
			Description      string `json:"description"`
			Founded          string `json:"founded"`
			Active           bool   `json:"active"`
			GuildBankBalance string `json:"guild_bank_balance"`
			MembersTotal     int    `json:"members_total"`
			MembersOnline    int    `json:"members_online"`
			Members          []struct {
				Name        string `json:"name"`
				Title       string `json:"title"`
				Rank        string `json:"rank"`
				Vocation    string `json:"vocation"`
				Level       int    `json:"level"`
				JoiningDate string `json:"joining_date"`
				IsOnline    bool   `json:"is_online"`
				ID          int    `json:"id"`
			} `json:"members"`
		} `json:"guild"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata guild detail: %w", err)
	}

	type member struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Level     int    `json:"level"`
		Vocation  int    `json:"vocation"`
		Rank      string `json:"rank"`
		RankLevel int    `json:"rankLevel"`
		Nick      string `json:"nick"`
		JoinDate  int64  `json:"joinDate"`
		IsOnline  bool   `json:"isOnline"`
	}

	members := make([]member, 0, len(src.Guild.Members))
	rankMap := make(map[string]bool)
	for _, m := range src.Guild.Members {
		members = append(members, member{
			ID: m.ID, Name: m.Name, Level: m.Level,
			Vocation: vocationNameToUpstreamID(m.Vocation), Rank: m.Rank,
			Nick: m.Title, JoinDate: parseSimpleDate(m.JoiningDate), IsOnline: m.IsOnline,
		})
		rankMap[m.Rank] = true
	}

	type rank struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Level int    `json:"level"`
	}
	ranks := make([]rank, 0)
	for r := range rankMap {
		ranks = append(ranks, rank{Name: r})
	}

	var owner *struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Level    int    `json:"level"`
		Vocation int    `json:"vocation"`
	}
	for _, m := range src.Guild.Members {
		if m.Rank == "Leader" {
			owner = &struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Level    int    `json:"level"`
				Vocation int    `json:"vocation"`
			}{ID: m.ID, Name: m.Name, Level: m.Level, Vocation: vocationNameToUpstreamID(m.Vocation)}
			break
		}
	}

	out := struct {
		Guild struct {
			ID           int      `json:"id"`
			Name         string   `json:"name"`
			MOTD         string   `json:"motd"`
			Description  string   `json:"description"`
			Homepage     string   `json:"homepage"`
			WorldID      int      `json:"world_id"`
			LogoName     string   `json:"logo_name"`
			Balance      string   `json:"balance"`
			CreationData int64    `json:"creationdata"`
			Owner        *struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Level    int    `json:"level"`
				Vocation int    `json:"vocation"`
			} `json:"owner"`
			Members []member `json:"members"`
			Ranks   []rank   `json:"ranks"`
		} `json:"guild"`
	}{}
	out.Guild.Name = src.Guild.Name
	out.Guild.Description = src.Guild.Description
	out.Guild.WorldID = src.Guild.WorldID
	out.Guild.LogoName = extractFilename(src.Guild.LogoURL)
	out.Guild.Balance = src.Guild.GuildBankBalance
	out.Guild.CreationData = parseSimpleDate(src.Guild.Founded)
	out.Guild.Owner = owner
	out.Guild.Members = members
	out.Guild.Ranks = ranks

	return marshalJSON(out)
}

// --- Banishments adapter ---

func adaptBanishmentsResponse(body string) (string, error) {
	var src struct {
		Banishments struct {
			Entries []struct {
				AccountID   int    `json:"account_id"`
				AccountName string `json:"account_name"`
				Character   string `json:"character"`
				Reason      string `json:"reason"`
				BannedAt    string `json:"banned_at"`
				ExpiresAt   string `json:"expires_at"`
				BannedBy    string `json:"banned_by"`
				IsPermanent bool   `json:"is_permanent"`
			} `json:"entries"`
		} `json:"banishments"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata banishments: %w", err)
	}

	type ban struct {
		AccountID     int    `json:"account_id"`
		AccountName   string `json:"account_name"`
		MainCharacter string `json:"main_character"`
		Reason        string `json:"reason"`
		BannedAt      string `json:"banned_at"`
		ExpiresAt     string `json:"expires_at"`
		BannedBy      string `json:"banned_by"`
		IsPermanent   bool   `json:"is_permanent"`
	}

	bans := make([]ban, 0, len(src.Banishments.Entries))
	for _, e := range src.Banishments.Entries {
		bans = append(bans, ban{
			AccountID: e.AccountID, AccountName: e.AccountName, MainCharacter: e.Character,
			Reason: e.Reason, BannedAt: e.BannedAt, ExpiresAt: e.ExpiresAt,
			BannedBy: e.BannedBy, IsPermanent: e.IsPermanent,
		})
	}

	out := struct {
		Bans        []ban `json:"bans"`
		TotalCount  int   `json:"totalCount"`
		TotalPages  int   `json:"totalPages"`
		CurrentPage int   `json:"currentPage"`
		CachedAt    int64 `json:"cachedAt"`
	}{Bans: bans, TotalCount: len(bans), TotalPages: 1, CurrentPage: 1, CachedAt: time.Now().UnixMilli()}

	return marshalJSON(out)
}

// --- Transfers adapter ---

func adaptTransfersResponse(body string) (string, error) {
	var src struct {
		Transfers struct {
			Entries []struct {
				Name         string `json:"name"`
				Level        int    `json:"level"`
				FromWorld    string `json:"from_world"`
				ToWorld      string `json:"to_world"`
				TransferDate string `json:"transfer_date"`
			} `json:"entries"`
		} `json:"transfers"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("parse rubinidata transfers: %w", err)
	}

	type transfer struct {
		ID            int    `json:"id"`
		PlayerID      int    `json:"player_id"`
		PlayerName    string `json:"player_name"`
		PlayerLevel   int    `json:"player_level"`
		FromWorldID   int    `json:"from_world_id"`
		ToWorldID     int    `json:"to_world_id"`
		FromWorld     string `json:"from_world"`
		ToWorld       string `json:"to_world"`
		TransferredAt string `json:"transferred_at"`
	}

	transfers := make([]transfer, 0, len(src.Transfers.Entries))
	for _, e := range src.Transfers.Entries {
		transfers = append(transfers, transfer{
			PlayerName: e.Name, PlayerLevel: e.Level,
			FromWorld: e.FromWorld, ToWorld: e.ToWorld, TransferredAt: e.TransferDate,
		})
	}

	out := struct {
		Transfers    []transfer `json:"transfers"`
		TotalResults int        `json:"totalResults"`
		TotalPages   int        `json:"totalPages"`
		CurrentPage  int        `json:"currentPage"`
	}{Transfers: transfers, TotalResults: len(transfers), TotalPages: 1, CurrentPage: 1}

	return marshalJSON(out)
}

// --- Helpers ---

// vocationNameToUpstreamID maps vocation display names to rubinot.com.br vocation IDs
// used in highscores and guild member structs.
func vocationNameToUpstreamID(name string) int {
	switch name {
	case "Knight", "Elite Knight":
		return 5
	case "Paladin", "Royal Paladin":
		return 4
	case "Sorcerer", "Master Sorcerer":
		return 2
	case "Druid", "Elder Druid":
		return 3
	case "Monk", "Exalted Monk":
		return 9
	case "None":
		return 1
	default:
		return 0
	}
}

func extractFilename(logoURL string) string {
	if logoURL == "" {
		return ""
	}
	parts := strings.Split(logoURL, "/")
	return parts[len(parts)-1]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/scraper/ -run "TestAdaptHighscores|TestAdaptGuildsList|TestAdaptGuildDetail|TestAdaptBanishments|TestAdaptTransfers" -v`
Expected: PASS

- [ ] **Step 5: Run ALL adapter tests**

Run: `go test ./internal/scraper/ -run "TestAdapt|TestTranslate" -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scraper/rubinidata_adapters.go internal/scraper/rubinidata_client_test.go
git commit -m "feat(rubinidata): add highscores, guilds, banishments, transfers adapters"
```

---

### Task 6: Wire RubinidataClient into CachedFetcher

This is the integration point. Modify `CachedFetcher.FetchJSON()` to route to RubinidataClient when `UPSTREAM_PROVIDER=rubinidata`.

**Files:**
- Modify: `internal/scraper/cached_fetcher.go`

- [ ] **Step 1: Add rubinidata field to CachedFetcher**

Add a `rubinidata *RubinidataClient` field to the `CachedFetcher` struct and update `NewCachedFetcher`:

```go
// In cached_fetcher.go, modify the struct:
type CachedFetcher struct {
	pool       *CDPPool
	rubinidata *RubinidataClient   // NEW
	group      singleflight.Group
	cache      sync.Map
	ttl        time.Duration
	warmMu     sync.Mutex
	lastWarmAt time.Time
	cfBlocked  atomic.Bool
}

// Modify NewCachedFetcher to accept optional rubinidata client:
func NewCachedFetcher(pool *CDPPool, ttl time.Duration) *CachedFetcher {
	var rc *RubinidataClient
	if IsRubinidataProvider() {
		rc = NewRubinidataClient()
	}
	return &CachedFetcher{pool: pool, rubinidata: rc, ttl: ttl}
}
```

- [ ] **Step 2: Add rubinidata fetch path to FetchJSON**

At the start of the singleflight callback in `FetchJSON()`, add a branch for rubinidata. Insert this before the existing CDP fetch loop (line 92 in current code):

```go
// Inside the f.group.Do callback, BEFORE the CDP retry loop:
if f.rubinidata != nil {
	started := time.Now()
	body, fetchErr := f.rubinidata.Fetch(ctx, cacheKey)
	CDPFetchDuration.Observe(time.Since(started).Seconds())

	if fetchErr != nil {
		CDPFetchRequests.WithLabelValues("error").Inc()
		return nil, fetchErr
	}

	CDPFetchRequests.WithLabelValues("ok").Inc()
	UpstreamStatus.WithLabelValues(endpointFromURL(apiURL), "200").Inc()

	f.cache.Store(cacheKey, &cacheEntry{
		value:     body,
		expiresAt: time.Now().Add(f.ttl),
	})
	return body, nil
}
```

- [ ] **Step 3: Add SetWorldMapping method pass-through**

```go
// Add to CachedFetcher:
func (f *CachedFetcher) SetRubinidataWorldMapping(m map[int]string) {
	if f.rubinidata != nil {
		f.rubinidata.SetWorldMapping(m)
	}
}
```

- [ ] **Step 4: Make IsReady always return true for rubinidata**

Modify `IsReady()`:

```go
func (f *CachedFetcher) IsReady() bool {
	if f.rubinidata != nil {
		return true
	}
	return !f.cfBlocked.Load()
}
```

- [ ] **Step 5: Verify existing tests still pass**

Run: `go test ./internal/scraper/ -v -count=1`
Expected: ALL PASS (existing tests should not be affected since `UPSTREAM_PROVIDER` env var is not set)

- [ ] **Step 6: Commit**

```bash
git add internal/scraper/cached_fetcher.go
git commit -m "feat(rubinidata): wire RubinidataClient into CachedFetcher fetch path"
```

---

### Task 7: Modify router to skip CDP init when using rubinidata

When `UPSTREAM_PROVIDER=rubinidata`, we don't need CDP/FlareSolverr. Modify `initOptimizedClient` to create a fetcher without a CDPPool.

**Files:**
- Modify: `internal/api/router.go`

- [ ] **Step 1: Modify initOptimizedClient**

Replace the existing `initOptimizedClient` function:

```go
func initOptimizedClient(ctx context.Context) (*scraper.OptimizedClient, error) {
	if scraper.IsRubinidataProvider() {
		log.Println("[router] UPSTREAM_PROVIDER=rubinidata — skipping CDP/FlareSolverr init")
		cacheTTLSeconds := getEnvInt("CDP_CACHE_TTL_SECONDS", 5)
		fetcher := scraper.NewCachedFetcher(nil, time.Duration(cacheTTLSeconds)*time.Second)
		return scraper.NewOptimizedClient(fetcher), nil
	}

	cdpURL := getEnv("CDP_URL", "")
	if cdpURL == "" {
		return nil, fmt.Errorf("CDP_URL not set")
	}
	poolSize := getEnvInt("CDP_POOL_SIZE", 4)
	pool := scraper.NewCDPPool(cdpURL, resolvedBaseURL, poolSize)
	if err := pool.Init(ctx); err != nil {
		return nil, fmt.Errorf("cdp pool init: %w", err)
	}
	cacheTTLSeconds := getEnvInt("CDP_CACHE_TTL_SECONDS", 5)
	fetcher := scraper.NewCachedFetcher(pool, time.Duration(cacheTTLSeconds)*time.Second)
	fetcher.SetLastWarmAt(time.Now())
	return scraper.NewOptimizedClient(fetcher), nil
}
```

- [ ] **Step 2: Update bootstrapValidatorV2 to set world mapping**

After the validator bootstraps (which fetches the worlds list), pass the world ID→name mapping to the rubinidata client. Add after `currentValidator.Store(validator)` in `NewRouter()`:

```go
if scraper.IsRubinidataProvider() && oc.Fetcher != nil {
	worldMap := make(map[int]string)
	for _, w := range worlds.Worlds {
		worldMap[w.ID] = w.Name
	}
	oc.Fetcher.SetRubinidataWorldMapping(worldMap)
}
```

Note: This requires that `bootstrapValidatorV2` returns the worlds result. Check the current function — if it doesn't expose `worlds`, we need to capture it. Looking at the current code, `bootstrapValidatorV2` fetches worlds internally via `V2FetchWorlds`. We should save the world mapping after the validator is created.

Alternative: Set the world mapping inside `bootstrapValidatorV2` or in `startValidatorRefresh`. The simplest approach: after `currentValidator.Store(validator)`, read the validator's world list and build the mapping:

```go
if scraper.IsRubinidataProvider() && oc.Fetcher != nil {
	if v := currentValidator.Load(); v != nil {
		oc.Fetcher.SetRubinidataWorldMapping(v.WorldIDToName())
	}
}
```

This requires adding a `WorldIDToName()` method to the Validator. Check `internal/validation/validator.go` for the world data structure and add the method there.

- [ ] **Step 3: Verify build compiles**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/api/router.go
git commit -m "feat(rubinidata): skip CDP init when UPSTREAM_PROVIDER=rubinidata"
```

---

### Task 8: Handle batch operations for rubinidata

The batch fetch methods (`BatchFetchJSON`, `FetchBinary`) need rubinidata support. `BatchFetchJSON` is used by the V2 batch handlers. For rubinidata, synthesize from individual calls.

**Files:**
- Modify: `internal/scraper/cached_fetcher.go`
- Modify: `internal/scraper/rubinidata_client.go`

- [ ] **Step 1: Add BatchFetch to RubinidataClient**

```go
// Add to rubinidata_client.go:

func (c *RubinidataClient) BatchFetch(ctx context.Context, upstreamPaths []string) (map[string]string, error) {
	results := make(map[string]string, len(upstreamPaths))
	var mu sync.Mutex
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	var firstErr error

	for _, path := range upstreamPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			body, err := c.Fetch(ctx, p)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					log.Printf("[rubinidata] batch item failed for %s: %v", p, err)
				}
				return
			}
			results[p] = body
		}(path)
	}

	wg.Wait()
	return results, firstErr
}
```

Add `"sync"` to the imports.

- [ ] **Step 2: Route BatchFetchJSON through rubinidata**

In `cached_fetcher.go`, modify `BatchFetchJSON` to check for rubinidata provider:

```go
func (f *CachedFetcher) BatchFetchJSON(ctx context.Context, apiURLs []string) (map[string]string, error) {
	// ... existing cache check logic stays ...

	if f.rubinidata != nil {
		// Build full upstream URLs for uncached items, fetch via rubinidata
		return f.rubinidata.BatchFetch(ctx, pendingKeys)
		// Note: need to integrate with the existing cache-check logic
	}

	// ... existing CDP batch logic ...
}
```

The integration needs care — the existing `BatchFetchJSON` has cache-check logic at the top. The rubinidata path should reuse that cache logic. The cleanest approach: extract the cache-hit portion, then branch on the fetch-miss path.

- [ ] **Step 3: Verify build compiles**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/scraper/cached_fetcher.go internal/scraper/rubinidata_client.go
git commit -m "feat(rubinidata): add batch fetch synthesis from individual calls"
```

---

### Task 9: Add Validator.WorldIDToName helper

The path translator needs world ID → name mapping. Add a helper to the Validator.

**Files:**
- Modify: `internal/validation/validator.go`

- [ ] **Step 1: Check current Validator struct for world data**

Read `internal/validation/validator.go` to find how worlds are stored (likely a map or slice of World structs).

- [ ] **Step 2: Add WorldIDToName method**

```go
func (v *Validator) WorldIDToName() map[int]string {
	m := make(map[int]string, len(v.worlds))
	for _, w := range v.worlds {
		m[w.ID] = w.Name
	}
	return m
}
```

Adjust field names based on the actual Validator struct.

- [ ] **Step 3: Verify build compiles**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/validation/validator.go
git commit -m "feat(validation): add WorldIDToName helper for rubinidata path translation"
```

---

### Task 10: Update docker-compose for rubinidata mode

Add rubinidata env vars to docker-compose for local testing.

**Files:**
- Modify: `docker-compose.yaml`

- [ ] **Step 1: Add rubinidata env vars to api service**

Add these environment variables to the `api` service:

```yaml
UPSTREAM_PROVIDER: "${UPSTREAM_PROVIDER:-rubinot}"
RUBINIDATA_URL: "https://api.rubinidata.com"
RUBINIDATA_API_KEY: "${RUBINIDATA_API_KEY:-}"
```

- [ ] **Step 2: Make CDP_URL optional**

When using rubinidata, CDP_URL is not needed. Change it to:

```yaml
CDP_URL: "${CDP_URL:-ws://localhost:9222}"
```

- [ ] **Step 3: Test locally with rubinidata**

```bash
UPSTREAM_PROVIDER=rubinidata RUBINIDATA_API_KEY=rbd_1a1f95224e93d9c3601f85aecb7e43ed docker-compose up --build
```

Then test:
```bash
curl -s http://localhost:18080/v2/worlds | python3 -m json.tool | head -20
curl -s http://localhost:18080/v2/character/Bubble | python3 -m json.tool | head -20
curl -s http://localhost:18080/v2/highscores/Elysian/experience/all | python3 -m json.tool | head -20
curl -s http://localhost:18080/readyz
```

Expected: All return valid JSON data, readyz returns `{"status": "ok"}`

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yaml
git commit -m "chore: add rubinidata env vars to docker-compose"
```

---

### Task 11: Update Kubernetes deployment for rubinidata

Add rubinidata env vars to the k8s deployment in platform-gitops.

**Files:**
- Modify: `~/git/github/rubinot-lab/platform-gitops/apps/rubinot/manifests/prod/rubinot-data.yaml`

- [ ] **Step 1: Add env vars to the API container**

Add to the API container's `env` section:

```yaml
- name: UPSTREAM_PROVIDER
  value: "rubinidata"
- name: RUBINIDATA_URL
  value: "https://api.rubinidata.com"
- name: RUBINIDATA_API_KEY
  valueFrom:
    secretKeyRef:
      name: rubinot-data-rubinidata
      key: api-key
```

- [ ] **Step 2: Create the secret (or use 1Password sync)**

Either create a k8s secret manually or via 1Password sync:

```bash
kubectl create secret generic rubinot-data-rubinidata -n rubinot \
  --from-literal=api-key=rbd_1a1f95224e93d9c3601f85aecb7e43ed
```

- [ ] **Step 3: Make CDP_URL optional in the deployment**

The CDP_URL env var can stay — it won't be used when UPSTREAM_PROVIDER=rubinidata, but removing it would cause the FlareSolverr sidecar to be pointless. For the bridge, keep the sidecar but it won't receive traffic.

- [ ] **Step 4: Commit and push platform-gitops changes**

```bash
git -C ~/git/github/rubinot-lab/platform-gitops add apps/rubinot/manifests/prod/rubinot-data.yaml
git -C ~/git/github/rubinot-lab/platform-gitops commit -m "feat(rubinot-data): add rubinidata bridge env vars"
git -C ~/git/github/rubinot-lab/platform-gitops push origin main
```

---

### Task 12: Tag, deploy, and verify

- [ ] **Step 1: Tag the rubinidata-bridge branch**

```bash
git tag vX.Y.Z-rubinidata-bridge
git push origin vX.Y.Z-rubinidata-bridge
```

- [ ] **Step 2: Wait for CI to build and push image**

```bash
gh run list --repo rubinot-lab/rubinot-data --limit 5
```

- [ ] **Step 3: Force ArgoCD refresh**

```bash
kubectl -n argocd patch application rubinot-lab-rubinot-prod \
  --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

- [ ] **Step 4: Verify pods come up ready**

```bash
kubectl get pods -n rubinot -l app=rubinot-data --no-headers \
  -o custom-columns='NAME:.metadata.name,READY:.status.containerStatuses[?(@.name=="api")].ready'
```

Expected: All pods show `true`

- [ ] **Step 5: Re-enable workers (platform-gitops)**

Restore worker replicas to original values and push. Monitor rubinot-data pods stay ready.

- [ ] **Step 6: Monitor logs for adapter errors**

```bash
kubectl logs -n rubinot deploy/rubinot-data -c api --tail=50 | grep -E "error|rubinidata|adapt"
```

Expected: No adapter errors, clean request handling

# rubinot-data: Full Parity Implementation Plan

## Context

**rubinot-data** is a greenfield Go API that scrapes rubinot.com.br (a Tibia private server) and serves structured JSON. It replaces two existing TypeScript services (`rubinot-live` + `rubinot-api`) with a single Go service using TibiaData (`tibiadata-api-go`) as the architectural reference.

**Current state:** 2 data endpoints (`/v1/world/:name`, `/v1/houses/:world/:town`) with OTel tracing and Prometheus metrics. Code has duplication across handlers.

**Goal:** Full rubinot-live parity (~25 data endpoints), shared infrastructure (validation, error taxonomy, telemetry), golden-fixture tests, docker-compose, and documentation.

**Key decisions:**
- Refactor shared patterns before adding endpoints
- Scrape-and-cache validation data at startup (worlds with IDs, towns, highscore categories)
- Adopt TibiaData error code ranges (10xxx-30xxx)
- Golden HTML fixtures for parser testing
- No Redis caching yet (placeholder metrics pre-wired)
- Docker-compose for local development
- Metric prefix: `rubinotdata_`

---

## Execution Protocol

1. Work **commit-by-commit** following the commit list in Section F. Do NOT skip commits or combine them.
2. After each commit, run `go test ./... -v -count=1`. Only proceed to the next commit when **all tests pass**.
3. If a test fails, fix the failure in the same commit scope. Do not create separate fix commits within a phase.
4. If a golden HTML fixture is needed and does not exist in `testdata/`, capture it using the fixture capture script (Section K). Do NOT write fake/synthetic HTML. If FlareSolverr is not available, create a minimal but structurally accurate fixture based on the parser's expectations and document it with `// FIXTURE: synthetic, needs real capture` in the test file.
5. Run `go vet ./...` and `go build ./...` after every commit. Zero warnings, zero errors.
6. When renaming existing metrics from `rubinot_` to `rubinotdata_`, update the K8s manifests and any Grafana dashboard references in the same commit.

## No Giving Up Rule

If an implementation detail is unclear (e.g., HTML structure for a rubinot-specific endpoint, CSS selector for a field, URL parameter format):
1. First: inspect the golden HTML fixture in `testdata/`. The fixture is the source of truth.
2. Second: inspect existing working code in this repo (`internal/scraper/world.go`, `internal/scraper/houses.go`) for established patterns.
3. Third: inspect the rubinot-live parser code referenced in Section H for the equivalent parser's approach.
4. Fourth: inspect the TibiaData parser code for the equivalent endpoint (see file names in Section H references).
5. If still unclear after all four sources: document the ambiguity as a `// TODO: AMBIGUOUS — <description>` comment, implement the most reasonable approach based on available evidence, and add a test that will fail if the assumption is wrong.
6. **Never guess. Never skip. Never leave a placeholder that compiles but does nothing.**

---

## API Contract

### Response Envelope (JSON)

Every response MUST use this envelope. No exceptions.

**Success (HTTP 200):**
```json
{
  "information": {
    "api": {
      "version": 1,
      "release": "v0.2.0",
      "commit": "abc1234"
    },
    "timestamp": "2026-02-22T15:04:05Z",
    "status": {
      "http_code": 200,
      "message": "ok"
    },
    "sources": [
      "https://www.rubinot.com.br/?subtopic=worlds&world=Belaria"
    ]
  },
  "world": { ... }
}
```

**Validation Error (HTTP 400):**
```json
{
  "information": {
    "api": { "version": 1, "release": "v0.2.0", "commit": "abc1234" },
    "timestamp": "2026-02-22T15:04:05Z",
    "status": {
      "http_code": 400,
      "error": 11001,
      "message": "world does not exist"
    },
    "sources": []
  }
}
```

**Upstream Error (HTTP 502):**
```json
{
  "information": {
    "api": { "version": 1, "release": "v0.2.0", "commit": "abc1234" },
    "timestamp": "2026-02-22T15:04:05Z",
    "status": {
      "http_code": 502,
      "error": 20001,
      "message": "flaresolverr request failed: connection refused"
    },
    "sources": [
      "https://www.rubinot.com.br/?subtopic=worlds&world=Belaria"
    ]
  }
}
```

**Not Found (HTTP 404):**
```json
{
  "information": {
    "api": { "version": 1, "release": "v0.2.0", "commit": "abc1234" },
    "timestamp": "2026-02-22T15:04:05Z",
    "status": {
      "http_code": 404,
      "error": 20004,
      "message": "character not found"
    },
    "sources": [
      "https://www.rubinot.com.br/?subtopic=characters&name=FakePlayer"
    ]
  }
}
```

**Payload key** varies by endpoint: `"world"`, `"worlds"`, `"houses"`, `"house"`, `"character"`, `"guild"`, `"guilds"`, `"highscores"`, `"killstatistics"`, `"news"`, `"newslist"`, `"deaths"`, `"transfers"`, `"banishments"`, `"events"`, `"auctions"`, `"auction"`.

### Error → HTTP Status Mapping

| Error Code Range | HTTP Status | Category |
|-----------------|-------------|----------|
| 10001-10007 | 400 | Character name validation |
| 11001-11008 | 400 | World/town/house/vocation/category existence |
| 14001-14007 | 400 | Guild name validation |
| 20001 | 502 | FlareSolverr connection failure |
| 20002 | 502 | FlareSolverr returned non-200 |
| 20003 | 502 | Cloudflare challenge still present |
| 20004 | 404 | Entity not found on upstream (character, guild, etc.) |
| 20005 | 503 | Upstream maintenance mode |
| 20006 | 502 | Upstream returned 403 (rate limited) |
| 20007 | 502 | Upstream unknown error |
| 20008 | 504 | FlareSolverr timeout |
| 30001-30010 | 400 | Rubinot-specific validation (page bounds, filter params) |

### Input Normalization Rules

| Input | Rule | Example |
|-------|------|---------|
| World name | Title case (first char upper, rest lower) | `bElArIa` → `Belaria` |
| Town name | Title case each word, preserve apostrophes | `ab dendriel` → `Ab Dendriel` |
| Character name | Trim whitespace, URL-encode for upstream | `Test Name` → `Test+Name` in URL |
| Guild name | Trim whitespace, URL-encode for upstream | `My Guild` → `My+Guild` in URL |
| Vocation | Lowercase, resolve aliases | `ek` → `Knights`, `ms` → `Sorcerers` |
| Highscore category | Lowercase, resolve aliases | `exp` → category ID 6 |
| Page number | Parse as int, must be ≥ 1 | `0` → error 30001 |
| House ID | Parse as int, must be ≥ 1 | `abc` → error 11006 |

### Timeout / Retry / Concurrency Policy

| Setting | Value | Rationale |
|---------|-------|-----------|
| FlareSolverr HTTP client timeout | 140s | Must exceed FlareSolverr's own maxTimeout |
| FlareSolverr `maxTimeout` parameter | 120000ms (120s) | Cloudflare challenges can take 30-60s |
| Resty retries | 0 (no retries) | FlareSolverr is itself a retry layer; double-retry wastes resources |
| Gin request timeout | None (rely on client timeout) | FlareSolverr is the bottleneck |
| Concurrent requests | Unlimited (no semaphore) | FlareSolverr manages its own concurrency |
| Validator refresh interval | At startup only (manual refresh via restart) | No background goroutine in v1 |

---

## Endpoint Contract Table

Every endpoint with full contract details.

### E1: GET /v1/worlds

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=worlds` |
| ID/Name mapping | None — page lists all worlds |
| Not-found detection | N/A (always returns a list) |
| Validation | None |
| Payload key | `"worlds"` |
| Fixtures | `testdata/worlds/overview.html` |
| Tests | `TestParseWorldsHTML_Normal`, `TestParseWorldsHTML_Empty` |

**Response fields:** `total_players_online` (int), `worlds[]` (name, status, players_online, location, pvp_type)

### E2: GET /v1/world/:name

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=worlds&world={normalizedName}` |
| ID/Name mapping | Name (string, title-cased) |
| Not-found detection | World table is empty or status row missing |
| Validation | `WorldExists(name)` → error 11001 |
| Payload key | `"world"` |
| Fixtures | `testdata/world/belaria.html`, `testdata/world/offline.html` |
| Tests | `TestParseWorldHTML_Normal`, `TestParseWorldHTML_Offline`, `TestParseWorldHTML_Empty` |

**Response fields:** name, info (status, players_online, location, pvp_type, creation_date), players_online[] (name, level, vocation)

### E3: GET /v1/character/:name

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=characters&name={urlEncoded(name)}` |
| ID/Name mapping | Name (string, URL-encoded) |
| Not-found detection | Page contains "Could not find character" or section header matches error pattern |
| Validation | `IsCharacterNameValid(name)` → error 10001-10007 |
| Payload key | `"character"` |
| Fixtures | `testdata/character/normal.html`, `testdata/character/with_deaths.html`, `testdata/character/traded.html`, `testdata/character/not_found.html`, `testdata/character/deleted.html`, `testdata/character/with_house.html`, `testdata/character/guild_member.html`, `testdata/character/banned.html` |
| Tests | `TestParseCharacterHTML_Normal`, `_WithDeaths`, `_Traded`, `_NotFound`, `_Deleted`, `_WithHouse`, `_GuildMember`, `_Banned` |

**Response fields:** character_info (name, former_names[], traded, deletion_date, sex, vocation, level, achievement_points, world, former_worlds[], residence, married_to, houses[], guild{name,rank}, last_login, account_status, comment, is_banned, ban_reason), deaths[] (time, level, killers[], assists[], reason), account_information (created, loyalty_title), other_characters[] (name, world, status, main, traded)

**Note:** Section headers may be in Portuguese ("Personagens", "Mortes"). Parser must handle both English and Portuguese.

### E4: GET /v1/guild/:name

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=guilds&page=view&GuildName={urlEncoded(name)}` |
| ID/Name mapping | Name (string, URL-encoded) |
| Not-found detection | "A guild by that name was not found" or `.ErrorMessage` present or empty `h1` |
| Validation | `IsGuildNameValid(name)` → error 14001-14007 |
| Payload key | `"guild"` |
| Fixtures | `testdata/guild/active.html`, `testdata/guild/disbanded.html`, `testdata/guild/not_found.html`, `testdata/guild/with_guildhall.html`, `testdata/guild/in_war.html` |
| Tests | `TestParseGuildHTML_Active`, `_Disbanded`, `_NotFound`, `_WithGuildhall`, `_InWar` |

**Response fields:** name, world, description, guildhalls[], active, founded, open_applications, homepage, in_war, disband_date, disband_condition, players_online, players_offline, members_total, members_invited, members[] (name, title, rank, vocation, level, joined, status), invites[] (name, date)

### E5: GET /v1/guilds/:world

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=guilds&world={worldId}` |
| ID/Name mapping | World name → **numeric worldId** via `Validator.WorldExists()` |
| Not-found detection | N/A (empty list if world has no guilds) |
| Validation | `WorldExists(world)` → error 11001 |
| Payload key | `"guilds"` |
| Fixtures | `testdata/guilds/list.html`, `testdata/guilds/empty.html` |
| Tests | `TestParseGuildsHTML_Normal`, `_Empty` |

**Response fields:** world, active[] (name, logo_url, description), formation[] (name, logo_url, description)

### E6: GET /v1/houses/:world/:town

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=houses&world={worldId}&town={townId}&state=&type=houses&order=name` (+ second request with `type=guildhalls`) |
| ID/Name mapping | World name → **numeric worldId**, Town name → **numeric townId** |
| Not-found detection | N/A (empty list if no houses) |
| Validation | `WorldExists(world)` → 11001, `TownExists(town)` → 11002 |
| Payload key | `"houses"` |
| Fixtures | `testdata/houses/venore_list.html`, `testdata/houses/guildhalls_list.html`, `testdata/houses/empty.html` |
| Tests | `TestParseHouseRows_Normal`, `_Guildhalls`, `_Empty`, `_WithAuctions` |

**Response fields:** world, town, house_list[] (house_id, name, size, rent, rented, auctioned, auction{current_bid, time_left, finished}), guildhall_list[] (same shape)

### E7: GET /v1/house/:world/:house_id

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=houses&page=view&world={worldId}&town={townId}&state=&type=houses&order=name&houseid={houseId}` |
| ID/Name mapping | World name → **numeric worldId**. Town is resolved from houseId→town static map. HouseId is path param. |
| Not-found detection | Main content table missing or regex captures nothing |
| Validation | `WorldExists(world)` → 11001, houseId must be int ≥ 1 → 11006 |
| Payload key | `"house"` |
| Fixtures | `testdata/house/rented.html`, `testdata/house/auctioned.html`, `testdata/house/auctioned_no_bid.html`, `testdata/house/vacant.html`, `testdata/house/moving.html`, `testdata/house/transfer.html` |
| Tests | `TestParseHouseDetailHTML_Rented`, `_Auctioned`, `_AuctionNoBid`, `_Vacant`, `_Moving`, `_Transfer` |

**Response fields:** houseid, world, town, name, type (house/guildhall), beds, size, rent, img, status{is_auctioned, is_rented, is_moving, is_transfering, auction{current_bid, current_bidder, auction_ongoing, auction_end}, rental{owner, owner_sex, paid_until, moving_date, transfer_receiver, transfer_price, transfer_accept}, original}

### E8: GET /v1/highscores/:world/:category/:vocation/:page

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=highscores&world={worldName}&category={categoryId}&currentpage={page}&profession={vocationString}` |
| ID/Name mapping | World name (string), category slug → **numeric categoryId**, vocation alias → vocation string |
| Not-found detection | N/A (empty list if no results) |
| Validation | `WorldExists(world)` → 11001, `HighscoreCategoryValid(cat)` → 11005, `VocationValid(voc)` → 11003, page ≥ 1 → 30001 |
| Payload key | `"highscores"` |
| Fixtures | `testdata/highscores/experience_page1.html`, `testdata/highscores/last_page.html`, `testdata/highscores/empty.html` |
| Tests | `TestParseHighscoresHTML_Experience`, `_LastPage`, `_Empty`, `_DynamicColumns` |

**Redirect routes:**
- `/v1/highscores/:world` → 302 to `/v1/highscores/:world/experience/all/1`
- `/v1/highscores/:world/:category` → 302 to `/v1/highscores/:world/:category/all/1`
- `/v1/highscores/:world/:category/:vocation` → handled with default page=1

**Response fields:** world, category, vocation, highscore_age, highscore_list[] (rank, name, vocation, world, level, value, title), highscore_page{current_page, total_pages, total_records}

### E9: GET /v1/killstatistics/:world

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=killstatistics&world={worldId}` |
| ID/Name mapping | World name → **numeric worldId** |
| Not-found detection | N/A |
| Validation | `WorldExists(world)` → 11001 |
| Payload key | `"killstatistics"` |
| Fixtures | `testdata/killstatistics/normal.html` |
| Tests | `TestParseKillstatisticsHTML_Normal`, `_FiltersZeroEntries` |

**Response fields:** world, entries[] (race, last_day_players_killed, last_day_killed, last_week_players_killed, last_week_killed), total{last_day_players_killed, last_day_killed, last_week_players_killed, last_week_killed}

### E10: GET /v1/news/id/:news_id

| Field | Value |
|-------|-------|
| Upstream URL | Needs HTML inspection — likely `{baseURL}/news` with specific article ID. Inspect fixture. |
| ID/Name mapping | news_id is numeric |
| Not-found detection | Article content missing or empty |
| Validation | news_id must be int > 0 → 30002 |
| Payload key | `"news"` |
| Fixtures | `testdata/news/article.html`, `testdata/news/ticker.html` |
| Tests | `TestParseNewsHTML_Article`, `_Ticker` |

**Response fields:** id, date, title, category, type, content, content_html

### E11: GET /v1/news/archive, /v1/news/latest, /v1/news/newsticker

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/news` (path-based, NOT `?subtopic=`) |
| ID/Name mapping | None |
| Not-found detection | N/A |
| Validation | /archive/:days → days must be int > 0, default 90 |
| Payload key | `"newslist"` |
| Fixtures | `testdata/news/list.html` |
| Tests | `TestParseNewsListHTML_Normal`, `_Empty` |

**Response fields:** news[] (id, date, title, category, type, excerpt)

### E12: GET /v1/deaths/:world

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=latestdeaths&world={worldId}` |
| ID/Name mapping | World name → **numeric worldId** |
| Not-found detection | N/A (empty list) |
| Validation | `WorldExists(world)` → 11001 |
| Payload key | `"deaths"` |
| Fixtures | `testdata/deaths/normal.html`, `testdata/deaths/pvp.html`, `testdata/deaths/empty.html` |
| Tests | `TestParseDeathsHTML_Normal`, `_PvP`, `_Empty` |

**Response fields:** world, deaths[] (date, victim{name, level}, killers[], is_pvp)

**Date format note:** rubinot.com.br uses Brazilian `DD/MM/YYYY, HH:MM:SS` with BRA timezone (UTC-3). Convert to UTC RFC3339.

### E13: GET /v1/transfers

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=transferstatistics` |
| ID/Name mapping | Optional world filter → **numeric worldId** |
| Not-found detection | N/A (empty list) |
| Validation | Optional world param: `WorldExists(world)` → 11001 |
| Payload key | `"transfers"` |
| Fixtures | `testdata/transfers/normal.html`, `testdata/transfers/empty.html` |
| Tests | `TestParseTransfersHTML_Normal`, `_Empty` |

**Response fields:** transfers[] (date, character, level, from_world, to_world)

### E14: GET /v1/banishments/:world

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=bans&world={worldId}` |
| ID/Name mapping | World name → **numeric worldId** |
| Not-found detection | N/A (empty list) |
| Validation | `WorldExists(world)` → 11001 |
| Payload key | `"banishments"` |
| Fixtures | `testdata/banishments/normal.html`, `testdata/banishments/empty.html` |
| Tests | `TestParseBanishmentsHTML_Normal`, `_Empty`, `_Permanent` |

**Response fields:** world, banishments[] (date, character, reason, duration, is_permanent, expires_at)

**Date format note:** Portuguese format `DD/MM/YYYY, HH:MM:SS`. Convert to UTC.

### E15: GET /v1/events/schedule

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?subtopic=eventcalendar` (optional: `&calendarmonth={m}&calendaryear={y}`) |
| ID/Name mapping | None |
| Not-found detection | N/A |
| Validation | Optional month (1-12), year (> 2000) |
| Payload key | `"events"` |
| Fixtures | `testdata/events/schedule.html`, `testdata/events/empty.html` |
| Tests | `TestParseEventsHTML_Normal`, `_Empty`, `_EndingEvents` |

**Response fields:** month, year, days[] (day, events[], active_events[], ending_events[]), all_events[]

### E16: GET /v1/auctions/current/:page

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/currentcharactertrades` (page 1) or `{baseURL}/currentcharactertrades?currentpage={page}` (page 2+) |
| ID/Name mapping | None |
| Not-found detection | N/A (empty list) |
| Validation | page ≥ 1 → 30001 |
| Payload key | `"auctions"` |
| Fixtures | `testdata/auctions/current.html`, `testdata/auctions/current_empty.html` |
| Tests | `TestParseAuctionsHTML_Current`, `_Empty`, `_Pagination` |

**Response fields:** auctions[] (auction_id, character_name, level, vocation, world, current_bid, auction_end), page{current_page, total_pages}

### E17: GET /v1/auctions/history/:page

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/pastcharactertrades` (page 1) or `{baseURL}/pastcharactertrades?currentpage={page}` (page 2+) |
| ID/Name mapping | None |
| Not-found detection | N/A |
| Validation | page ≥ 1 → 30001 |
| Payload key | `"auctions"` |
| Fixtures | `testdata/auctions/history.html` |
| Tests | `TestParseAuctionsHTML_History` |

**Response fields:** Same as E16.

### E18: GET /v1/auctions/:id

| Field | Value |
|-------|-------|
| Upstream URL | `{baseURL}/?currentcharactertrades/{id}` (try first), then `{baseURL}/?pastcharactertrades/{id}` (fallback) |
| ID/Name mapping | Auction ID (string) |
| Not-found detection | Neither URL returns valid auction data |
| Validation | id must be non-empty string |
| Payload key | `"auction"` |
| Fixtures | `testdata/auctions/detail_active.html`, `testdata/auctions/detail_ended.html` |
| Tests | `TestParseAuctionDetailHTML_Active`, `_Ended` |

**Response fields:** auction_id, character_name, level, vocation, sex, world, bid_type, bid_amount, auction_start, auction_end, status, stats{...20+ fields}, skills{...8 types}, blessings, mounts, outfits, titles

---

## ID Mapping Reference

### World IDs (dynamically discovered at startup from `/?subtopic=latestdeaths` dropdown)

| ID | World Name |
|----|-----------|
| 1 | Elysian |
| 9 | Lunarian |
| 10 | Spectrum |
| 11 | Auroria |
| 12 | Solarian |
| 15 | Belaria |
| 16 | Vesperia |
| 17 | Bellum |
| 18 | Mystian |
| 21 | Tenebrium |
| 22 | Serenian |

### Town IDs (static, hardcoded)

| ID | Town Name |
|----|-----------|
| 1 | Venore |
| 2 | Thais |
| 3 | Kazordoon |
| 4 | Carlin |
| 5 | Ab Dendriel |
| 7 | Liberty Bay |
| 8 | Port Hope |
| 9 | Ankrahmun |
| 10 | Darashia |
| 11 | Edron |
| 12 | Svargrond |
| 13 | Yalahar |
| 14 | Farmine |
| 33 | Rathleton |
| 63 | Issavi |
| 66 | Moonfall |
| 67 | Silvertides |

### Highscore Category IDs

| ID | Slug | Aliases |
|----|------|---------|
| 0 | achievements | achievements |
| 2 | axe | axe, axefighting |
| 4 | club | club, clubfighting |
| 5 | distance | distance, dist, distancefighting |
| 6 | experience | experience, exp |
| 7 | fishing | fishing |
| 8 | fist | fist, fistfighting |
| 10 | loyalty | loyalty, loyaltypoints |
| 11 | magic | magic, mlvl, magiclevel |
| 12 | shielding | shielding, shield |
| 13 | sword | sword, swordfighting |
| 14 | drome | drome, dromescore |
| 15 | linked-tasks | linked-tasks |
| 16 | daily-xp | daily-xp |
| 18 | battle-pass | battle-pass |
| 19 | charm | charm, charmpoints |
| 20 | prestige | prestige |
| 21 | weekly-tasks | weekly-tasks |
| 22 | bounty | bounty, bountypoints |

### Vocations

| Canonical | Aliases |
|-----------|---------|
| (all) | all, (all) |
| None | none, no vocation |
| Knights | knight, knights, ek |
| Paladins | paladin, paladins, rp |
| Sorcerers | sorcerer, sorcerers, ms |
| Druids | druid, druids, ed |
| Monks | monk, monks |

---

## Metrics Contract

**Prefix:** `rubinotdata_`

### Full Metrics Inventory

**HTTP (Gin middleware):**
- `rubinotdata_http_requests_total{method,route,status_code}` — Counter. `route` uses Gin template path (e.g., `/v1/character/:name`), NOT actual URL.
- `rubinotdata_http_request_duration_seconds{method,route}` — Histogram.
- `rubinotdata_http_response_size_bytes{method,route}` — Histogram.
- `rubinotdata_http_requests_in_flight` — Gauge.

**Scraper:**
- `rubinotdata_scrape_requests_total{endpoint,status}` — Counter.
- `rubinotdata_scrape_duration_seconds{endpoint}` — Histogram (buckets: 0.25, 0.5, 1, 2, 3, 5, 8, 13, 21).
- `rubinotdata_parse_duration_seconds{endpoint}` — Histogram (buckets: 0.01, 0.03, 0.06, 0.1, 0.2, 0.5, 1, 2).
- `rubinotdata_flaresolverr_requests_total{status}` — Counter (status: ok, error, timeout, cf_challenge).
- `rubinotdata_flaresolverr_duration_seconds` — Histogram.
- `rubinotdata_cloudflare_challenges_total` — Counter.

**Upstream:**
- `rubinotdata_upstream_status_total{endpoint,status_code}` — Counter.
- `rubinotdata_upstream_maintenance_total` — Counter.

**Validation:**
- `rubinotdata_validation_rejections_total{endpoint,error_code}` — Counter.

**Parser:**
- `rubinotdata_parse_errors_total{endpoint,error_type}` — Counter.
- `rubinotdata_parse_items_total{endpoint}` — Gauge.

**Business:**
- `rubinotdata_worlds_discovered` — Gauge.
- `rubinotdata_world_players_online{world}` — Gauge.
- `rubinotdata_worlds_total_players_online` — Gauge.
- `rubinotdata_validator_refresh_total{status}` — Counter.
- `rubinotdata_validator_refresh_duration_seconds` — Histogram.

**Cache placeholders (registered, never incremented until Redis):**
- `rubinotdata_cache_requests_total{endpoint,result}` — Counter (result: hit, miss, stale).
- `rubinotdata_cache_duration_seconds{endpoint}` — Histogram.
- `rubinotdata_cache_entries{endpoint}` — Gauge.
- `rubinotdata_cache_stale_serves_total{endpoint}` — Counter.

### Grafana Dashboards

**Dashboard 1: API Overview**
- Row 1 — Traffic: request rate by endpoint, in-flight gauge, error rate %
- Row 2 — Latency: p50/p95/p99 by endpoint, response size distribution
- Row 3 — Validation: rejection rate by endpoint+error_code, top rejected codes table

**Dashboard 2: Scraper Health**
- Row 1 — FlareSolverr: request rate by status, p95 latency, CF challenge rate
- Row 2 — Scrape Ops: success ratio by endpoint, scrape p95 duration, parse p95 duration
- Row 3 — Upstream: status code distribution, maintenance events, parse error rate

**Dashboard 3: Business & Game Metrics**
- Row 1 — World Status: online players per world (stacked area), total online, worlds discovered
- Row 2 — Validator: refresh success/failure rate, refresh duration
- Row 3 — Cache Readiness: hit/miss/stale ratio (placeholder), entries, stale serves
- Row 4 — Endpoint Usage: top endpoints by volume (bar), response size comparison

---

## File Layout

```
rubinot-data/
├── cmd/server/main.go                    # EXISTS - modify
├── internal/
│   ├── api/
│   │   ├── router.go                    # EXISTS - refactor
│   │   ├── handler.go                   # NEW: shared handler
│   │   ├── envelope.go                  # NEW: response envelope
│   │   ├── middleware.go                # NEW: metrics middleware
│   │   └── handlers/                    # NEW: per-endpoint handlers
│   │       ├── world.go, worlds.go, character.go, guild.go, guilds.go
│   │       ├── house.go, houses.go, highscores.go, killstatistics.go
│   │       ├── news.go, newslist.go
│   │       ├── deaths.go, transfers.go, banishments.go, events.go, auctions.go
│   ├── scraper/
│   │   ├── client.go                   # NEW: shared FlareSolverr client
│   │   ├── telemetry.go               # EXISTS - rename metrics
│   │   ├── world.go                    # EXISTS - refactor
│   │   ├── houses.go                   # EXISTS - refactor
│   │   ├── worlds.go, house.go, character.go, guild.go, guilds.go
│   │   ├── highscores.go, killstatistics.go, news.go, newslist.go
│   │   ├── deaths.go, transfers.go, banishments.go, events.go, auctions.go
│   ├── validation/
│   │   ├── validator.go               # NEW: startup loader with ID maps
│   │   ├── errors.go                  # NEW: error codes 10xxx-30xxx
│   │   ├── names.go                   # NEW: name validation
│   │   ├── highscores.go             # NEW: category/vocation validation
│   │   └── limits.go                  # NEW: static limits
│   ├── observability/otel.go          # EXISTS - no changes
│   └── domain/                        # NEW: all domain structs
│       ├── world.go, character.go, guild.go, house.go, highscores.go
│       ├── killstatistics.go, news.go, deaths.go, transfers.go
│       ├── banishments.go, events.go, auctions.go
├── testdata/                          # NEW: golden HTML fixtures
│   └── (per endpoint subdirectories as listed in endpoint contracts above)
├── deploy/k8s/rubinot-data.yaml      # EXISTS - update
├── docker-compose.yaml               # NEW
├── Dockerfile                        # EXISTS
├── Makefile                          # NEW
├── go.mod, go.sum                    # EXISTS - update
└── README.md                         # NEW
```

---

## Fixture Capture Script Spec

To capture a golden HTML fixture from rubinot.com.br via FlareSolverr:

```bash
# Requires: FlareSolverr running (docker compose up flaresolverr)
# Usage: ./scripts/capture-fixture.sh <rubinot_url> <output_path>
# Example:
#   ./scripts/capture-fixture.sh "https://www.rubinot.com.br/?subtopic=worlds" testdata/worlds/overview.html

curl -s -X POST http://localhost:8191/v1 \
  -H "Content-Type: application/json" \
  -d '{
    "cmd": "request.get",
    "url": "'"$1"'",
    "maxTimeout": 120000,
    "headers": {
      "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
      "Accept-Language": "en-US,en;q=0.9,pt-BR;q=0.8"
    }
  }' | jq -r '.solution.response' > "$2"

echo "Captured $(wc -c < "$2") bytes to $2"
```

The Makefile target `make fixture URL=... OUT=...` wraps this script.

---

## Commit Plan

### Phase 1: Foundation (4 commits)

**C1: `refactor(scraper): extract shared FlareSolverr client`**
- Create `internal/scraper/client.go` — FlareSolverrClient struct, Fetch method
- Move flareSolverrRequest/Response structs from world.go
- Update world.go and houses.go to use shared client
- Tests: `TestFetch_Success`, `_Down`, `_Non200`, `_TargetNon200`, `_CloudflareChallenge`

**C2: `refactor(api): extract response envelope and handler pattern`**
- Create `internal/api/envelope.go` — Information, Status, APIDetails structs; NewSuccess, NewError builders
- Create `internal/api/handler.go` — DataHandler struct with client + baseURL + validator
- Create `internal/api/middleware.go` — HTTP metrics middleware (rubinotdata_http_*)
- Move handlers to `internal/api/handlers/world.go`, `handlers/houses.go`
- Rename metrics from `rubinot_` to `rubinotdata_`
- Tests: `TestNewSuccess`, `TestNewError_ValidationError`, `TestNewError_GenericError`

**C3: `refactor(domain): extract domain structs from scraper`**
- Create `internal/domain/` package with world.go, house.go
- Move WorldResult, WorldInfo, PlayerOnline, HousesResult, HouseEntry
- Update all imports
- Tests: all existing tests pass unchanged

**C4: `feat(validation): add startup validator with world/town/category/vocation validation`**
- Create `internal/validation/validator.go` — InitValidator (scrapes latestdeaths dropdown for world IDs), WorldExists, TownExists, Refresh
- Create `internal/validation/errors.go` — Error interface, all codes 10001-30010
- Create `internal/validation/names.go` — IsCharacterNameValid, IsGuildNameValid
- Create `internal/validation/highscores.go` — HighscoreCategoryValid, VocationValid
- Create `internal/validation/limits.go` — static name length limits
- Wire into main.go and handlers
- Tests: 16 validation tests as specified in endpoint contracts

### Phase 2: Core Endpoints (8 commits)

**C5: `feat(api): add worlds list endpoint`** — E1 contract
**C6: `feat(api): add house details endpoint`** — E7 contract
**C7: `feat(api): add character endpoint`** — E3 contract
**C8: `feat(api): add guild and guilds list endpoints`** — E4 + E5 contracts
**C9: `feat(api): add highscores endpoint`** — E8 contract
**C10: `feat(api): add killstatistics endpoint`** — E9 contract
**C11: `feat(api): add news endpoints`** — E10 + E11 contracts
**C12: `refactor(api): enrich existing houses endpoint and add golden fixtures for world/houses`** — E2 + E6 contracts (update existing parsers, add fixtures + tests)

### Phase 3: Rubinot-Specific Endpoints (5 commits)

**C13: `feat(api): add world deaths endpoint`** — E12 contract
**C14: `feat(api): add transfers endpoint`** — E13 contract
**C15: `feat(api): add banishments endpoint`** — E14 contract
**C16: `feat(api): add events schedule endpoint`** — E15 contract
**C17: `feat(api): add auctions endpoints`** — E16 + E17 + E18 contracts

### Phase 4: Infrastructure (4 commits)

**C18: `chore(infra): add docker-compose for local development`**
- docker-compose.yaml with rubinot-data + flaresolverr services
- Health check for FlareSolverr readiness

**C19: `chore(build): add Makefile`**
- Targets: build, test, test-cover, lint, run, docker-build, docker-up, docker-down, fixture

**C20: `chore(infra): update k8s manifests`**
- Update deploy/k8s/rubinot-data.yaml with new env vars, APP_VERSION

**C21: `docs: add project documentation`**
- README.md: overview, architecture, endpoints, getting started, dev guide, env vars, error codes, caching strategy, observability guide

### Phase 5: Quality Gate (3 commits)

**C22: `test: add comprehensive integration test suite`**
- HTTP-level tests for all endpoints (happy + error paths)

**C23: `chore: PR self-review and code cleanup`**
- Review all changes, remove unnecessary comments, run linters

**C24: `fix: address PR review feedback`**
- Apply fixes, iterate tests

---

## Definition of Done

Every item must be checked before the plan is considered complete. Binary — no partial credit.

- [ ] All 18 data endpoints (E1-E18) return valid JSON matching the response envelope contract
- [ ] Every endpoint has at least one golden HTML fixture in `testdata/`
- [ ] Every parser has unit tests against golden fixtures — all passing
- [ ] Every endpoint has integration tests (HTTP-level) — all passing
- [ ] `go test ./... -v -count=1` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go build ./...` exits 0
- [ ] All metrics use `rubinotdata_` prefix (no `rubinot_` metrics remain)
- [ ] All metrics listed in the metrics contract are registered (including cache placeholders)
- [ ] HTTP metrics middleware uses `route` label with Gin template path (not actual URL)
- [ ] Validation rejects invalid input with correct error codes before any FlareSolverr call
- [ ] World ID discovery works at startup from `/?subtopic=latestdeaths` dropdown
- [ ] Town IDs are hardcoded with the 17-town static map
- [ ] Highscore category IDs are mapped with all aliases
- [ ] Brazilian dates (DD/MM/YYYY BRA) are converted to UTC RFC3339
- [ ] Error responses use the correct HTTP status per the error→HTTP mapping table
- [ ] docker-compose.yaml starts rubinot-data + FlareSolverr successfully
- [ ] Makefile has working targets: build, test, test-cover, lint, run, docker-build, docker-up
- [ ] K8s manifests updated with correct env vars and APP_VERSION
- [ ] README.md documents all endpoints, env vars, error codes, and getting started
- [ ] Fixture capture script exists and works (`make fixture URL=... OUT=...`)
- [ ] No unnecessary comments in code
- [ ] No `// TODO` items remain (except documented ambiguities per the No Giving Up Rule)
- [ ] All commits follow semantic commit format
- [ ] PR self-review completed and feedback applied

---

## Caching Strategy (Future Reference)

### When to Add
- FlareSolverr p95 > 5s consistently
- rubinot.com.br rate-limiting (repeated 403s)
- Multiple consumers querying same data

### Architecture
```
Client → Gin → Redis check
  ├── HIT → return + X-Cache-Status: HIT
  └── MISS → FlareSolverr → parse → Redis store → return + X-Cache-Status: MISS
      └── ERROR + stale exists → stale + X-Cache-Status: STALE
```

### TTLs
| Endpoint | TTL | Rationale |
|----------|-----|-----------|
| /v1/worlds | 5m | Rarely changes |
| /v1/world/:name | 30s | Player count dynamic |
| /v1/character/:name | 2m | Moderately dynamic |
| /v1/guild/:name | 5m | Infrequent changes |
| /v1/houses/:world/:town | 15m | Slow changes |
| /v1/highscores/... | 10m | Periodic updates |
| /v1/killstatistics/:world | 1h | Daily aggregates |
| /v1/news/* | 30m | Infrequent |
| /v1/deaths/:world | 30s | Real-time |
| /v1/transfers | 5m | Moderate |
| /v1/banishments/:world | 15m | Rare changes |
| /v1/events/schedule | 1h | Static schedule |
| /v1/auctions/* | 2m | Auction state |

### Cache Key Pattern
`rubinotdata:v1:<endpoint>:<normalized_params>`

### Headers
`X-Cache-Status: HIT | MISS | STALE`, `X-Cache-Age`, `X-Cache-TTL`

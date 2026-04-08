# rubinidata.com Bridge — Design Doc

## Context

rubinot-data proxies rubinot.com.br via CDP/FlareSolverr to bypass Cloudflare. When rubinot-api workers come online, they generate burst traffic (790+ URLs in batch requests within seconds). Cloudflare detects this as automated behavior and blocks all pods within ~15 seconds.

We confirmed this by:
1. Running rubinot-data locally on Mac — works perfectly (no burst consumers)
2. Scaling 16 pods on cluster with workers at 0 — all 16 stay ready
3. Re-enabling workers — all 16 pods go not-ready within minutes

rubinidata.com (api.rubinidata.com) is a third-party API by "Duarte" that serves the same rubinot.com.br data as JSON. With our API key (`X-API-Key` header), there are no rate limits and no Cloudflare challenge — plain HTTP.

## Decision: Temporary Bridge (Option B)

- Throwaway branch — deploy from it, don't merge to main
- Use rubinidata.com as upstream for Tier 1+2 worker jobs
- Disable workers that need unsupported endpoints
- Revert when whitelisted on rubinot.com.br

## rubinidata.com API Details

- Base URL: `https://api.rubinidata.com`
- Auth: `X-API-Key: rbd_1a1f95224e93d9c3601f85aecb7e43ed`
- Rate limit: 25 req/min without key, **unlimited with key**
- Caching: `X-Cache: HIT` header present, cached responses don't consume rate limit
- No Cloudflare challenge on API endpoints

## Architecture

### Flow Change

```
Current (rubinot.com.br):
  Handler -> OptimizedClient -> CachedFetcher -> CDPPool -> CDP tab -> rubinot.com.br/api/*
  
Bridge (rubinidata.com):
  Handler -> OptimizedClient -> CachedFetcher -> RubinidataClient -> net/http -> api.rubinidata.com/v1/*
```

The CachedFetcher is the integration point. When `UPSTREAM_PROVIDER=rubinidata`, its `FetchJSON()` method routes to the RubinidataClient instead of acquiring a CDP tab. The cache and singleflight deduplication still apply on top — no change to those layers.

### New Files

```
internal/scraper/
├── rubinidata_client.go      # HTTP client + path router + response adapter orchestration
├── rubinidata_adapters.go    # Per-endpoint response format converters
```

### Modified Files

```
internal/scraper/cached_fetcher.go  # Route to rubinidata or CDP based on provider
internal/api/router.go              # Skip CDP/FlareSolverr init when provider=rubinidata
                                    # Readiness probe always returns ready
```

### Env Vars

```
UPSTREAM_PROVIDER=rubinidata    # "rubinot" (default) | "rubinidata"
RUBINIDATA_URL=https://api.rubinidata.com   # default
RUBINIDATA_API_KEY=rbd_1a1f95224e93d9c3601f85aecb7e43ed
```

## Endpoint Mapping — Full Detail

### 1. Worlds List

```
rubinot.com.br:  GET /api/worlds
rubinidata.com:  GET /v1/worlds
```

**rubinot.com.br returns:**
```json
{
  "worlds": [
    {"id": 1, "name": "Elysian", "pvpType": "no-pvp", "pvpTypeLabel": "Optional PvP", 
     "worldType": "yellow", "locked": false, "playersOnline": 1313}
  ],
  "totalOnline": 12902,
  "overallRecord": 27884,
  "overallRecordTime": 1770583857
}
```

**rubinidata.com returns:**
```json
{
  "worlds": {
    "overview": {"total_players_online": 12902, "overall_maximum": 27884, 
                 "maximum_date": "Feb 08 2026, 17:50:57 BRT"},
    "regular_worlds": [
      {"name": "Elysian", "players_online": 1313, "pvp_type": "no-pvp",
       "pvp_type_label": "Optional PvP", "world_type": "yellow", "locked": false, "id": 0}
    ]
  }
}
```

**Adapter logic:**
- `worlds.overview.total_players_online` → `totalOnline`
- `worlds.overview.overall_maximum` → `overallRecord`
- Parse `worlds.overview.maximum_date` string → unix timestamp for `overallRecordTime`
- `worlds.regular_worlds[]` → `worlds[]` with field renames:
  - `players_online` → `playersOnline`
  - `pvp_type` → `pvpType`
  - `pvp_type_label` → `pvpTypeLabel`
  - `world_type` → `worldType`
  - `id` stays 0 (rubinidata doesn't assign IDs)
  - Add `status: "online"` (not in rubinidata response, assume online)

### 2. World Detail

```
rubinot.com.br:  GET /api/worlds/{name}
rubinidata.com:  GET /v1/world/{name}
```

**rubinot.com.br returns:**
```json
{
  "world": {"id": 1, "name": "Elysian", "pvpType": "no-pvp", "pvpTypeLabel": "Optional PvP",
            "worldType": "yellow", "locked": false, "creationDate": 1692259800},
  "playersOnline": 1339,
  "record": 3816,
  "recordTime": 1717109676,
  "players": [{"name": "Bubble", "level": 8, "vocation": "Knight", "vocationId": 5}]
}
```

**rubinidata.com returns:**
```json
{
  "world": {"name": "Elysian", "status": "online", "players_online": 1339,
            "online_record": {"players": 3816, "date": "May 30 2024, 19:54:36 BRT"},
            "creation_date": "Aug 17 2023", "pvp_type": "no-pvp", 
            "pvp_type_label": "Optional PvP", "world_type": "yellow", "locked": false, "id": 0},
  "players": [{"name": "Bubble", "level": 8, "vocation": "Knight", "vocation_id": 0}]
}
```

**Adapter logic:**
- Restructure `world` block: flatten `online_record` into `record`/`recordTime`
- Parse `online_record.date` string → unix timestamp
- Parse `creation_date` string ("Aug 17 2023") → unix timestamp
- `players_online` moves to top-level `playersOnline`
- Field renames in world: `pvp_type` → `pvpType`, etc.
- `players[].vocation_id` is always 0 from rubinidata — leave as-is

### 3. Character

```
rubinot.com.br:  GET /api/characters/search?name={name}
rubinidata.com:  GET /v1/characters/{name}
```

**rubinot.com.br returns:**
```json
{
  "player": {
    "id": 123, "account_id": 456, "name": "Bubble", "level": 8,
    "vocation": "Knight", "vocationId": 5, "world_id": 17, "sex": "female",
    "residence": "Thais", "lastlogin": "2023-04-27T14:33:48-03:00",
    "created": 1681930831, "comment": "0", "account_created": 1681930831,
    "loyalty_points": 0, "isHidden": false, "achievementPoints": 13,
    "vip_time": 1773063352, "foundByOldName": false,
    "looktype": 136, "lookhead": 0, "lookbody": 0, "looklegs": 0, "lookfeet": 0, "lookaddons": 0,
    "guild": {"id": 1, "name": "GuildName", "rank": "Leader", "nick": ""},
    "house": {"id": 789, "name": "HouseName", "town_id": 1, "rent": 5000, "size": 100},
    "formerNames": [], "title": null, "partner": null, "auction": null
  },
  "deaths": [
    {"time": "2026-04-08T21:31:48Z", "level": 62, "killed_by": "monster", 
     "is_player": 0, "mostdamage_by": "monster2", "mostdamage_is_player": 0}
  ],
  "otherCharacters": [
    {"name": "Alt", "world": "Elysian", "world_id": 1, "level": 23, "vocation": "Knight", "isOnline": false}
  ],
  "accountBadges": [],
  "displayedAchievements": [],
  "banInfo": {},
  "foundByOldName": false
}
```

**rubinidata.com returns:**
```json
{
  "characters": {
    "character": {
      "id": 0, "name": "Bubble", "traded": false, "level": 8,
      "vocation": "Knight", "vocation_id": 0, "world_id": 17, "world_name": "Bellum",
      "sex": "female", "achievement_points": 13, "residence": "Thais",
      "last_login": "2023-04-27T14:33:48-03:00", "account_status": "Free Account",
      "comment": "0", "outfit_url": "/v1/outfit?type=136&...", "loyalty_points": 0, "created": ""
    },
    "other_characters": [
      {"id": 0, "name": "Alt", "level": 23, "vocation": "Knight", "world_id": 1, "world_name": "Elysian"}
    ]
  }
}
```

**Adapter logic — construct `characterAPIResponse` shape:**
- `characters.character` → `player` with field renames:
  - `last_login` → `lastlogin`
  - `achievement_points` → `achievementPoints`
  - `vocation_id` → `vocationId` (will be 0)
  - Parse `outfit_url` query params → `looktype`, `lookhead`, etc.
  - Set `world_id` from character data
- `found_by_old_name` → `foundByOldName` (now available from rubinidata!)
  - `former_names` → `formerNames` (now available!)
  - `created` → parse timestamp string to unix for `created`
  - `house` is a string in rubinidata (e.g. "Main Street 9b") — set structured `house` to nil, keep string for display
- **Missing fields set to zero/empty:**
  - `account_id`: 0
  - `vip_time`: 0
  - `account_created`: 0
  - `guild`: null (not embedded in rubinidata character response)
  - `isHidden`: false
- `characters.other_characters` → `otherCharacters` with:
  - `world_name` → `world`
  - `isOnline`: false (not provided by rubinidata)
- **deaths**: empty array (rubinidata doesn't include deaths in character response)
- **accountBadges**: empty array
- **displayedAchievements**: empty array
- **banInfo**: empty map

### 4. Guild List

```
rubinot.com.br:  GET /api/guilds?world={worldId}&page={page}
rubinidata.com:  GET /v1/guilds/{worldName}?page={page}
```

**rubinot.com.br returns:**
```json
{
  "guilds": [
    {"id": 1, "name": "GuildName", "description": "...", "world_id": 1, "logo_name": "logo.gif"}
  ],
  "totalCount": 100, "totalPages": 5, "currentPage": 1
}
```

**rubinidata.com returns:**
```json
{
  "guilds": {
    "guilds": [
      {"name": "GuildName", "logo_url": "https://static.rubinot.com/guilds/logo.gif",
       "description": "...", "id": 0, "world_id": 0}
    ]
  }
}
```

**Adapter logic:**
- `guilds.guilds[]` → `guilds[]`
- Extract `logo_name` from `logo_url` (parse filename from URL)
- **Missing:** `totalCount`, `totalPages`, `currentPage` — rubinidata returns all guilds at once (no pagination info). Set totalPages=1, currentPage=1, totalCount=len(guilds).

### 5. Guild Detail

```
rubinot.com.br:  GET /api/guilds/{name}
rubinidata.com:  GET /v1/guild/{name}
```

**rubinot.com.br returns:**
```json
{
  "guild": {
    "id": 1, "name": "Name", "motd": "", "description": "...", "homepage": "",
    "world_id": 1, "logo_name": "logo.gif", "balance": 0,
    "creationdata": 1735430400,
    "owner": {"id": 1, "name": "Owner", "level": 200, "vocation": 1},
    "members": [
      {"id": 1, "name": "Member", "level": 150, "vocation": 1, 
       "rank": "Leader", "rankLevel": 3, "nick": "", "joinDate": 1735430400, "isOnline": true}
    ],
    "ranks": [{"id": 1, "name": "Leader", "level": 3}],
    "residence": {"id": 1, "name": "ResName", "town": "TownName"}
  }
}
```

**rubinidata.com returns:**
```json
{
  "guild": {
    "name": "Name", "world_id": 0, "logo_url": "...", "description": "...",
    "founded": "Dec 29 2025", "active": true, "guild_bank_balance": "0",
    "members_total": 791, "members_online": 175,
    "members": [
      {"name": "Member", "title": "", "rank": "Leader", "vocation": "Elite Knight",
       "level": 1423, "joining_date": "Feb 10 2026", "is_online": false, "id": 0}
    ]
  }
}
```

**Adapter logic:**
- Parse `founded` date string → unix timestamp for `creationdata`
- `guild_bank_balance` string → `balance` interface{}
- `logo_url` → extract filename for `logo_name`
- `members[]`: vocation is string in rubinidata, int in upstream — map string to vocation ID
- `members[].joining_date` → parse to unix timestamp for `joinDate`
- `members[].rank` stays, add `rankLevel` = 0
- **Missing:** `owner` (not in rubinidata) — derive from first Leader member
- **Missing:** `ranks` — derive from unique rank names in members
- **Missing:** `residence` — set to nil
- **Missing:** `motd`, `homepage`, `world_id` (proper) — set to zero/empty

### 6. Highscores

```
rubinot.com.br:  GET /api/highscores?world={worldId}&category={slug}&vocation={vocId}
rubinidata.com:  GET /v1/highscores?world={worldName}&category={slug}&vocation={vocId}
```

**rubinot.com.br returns:**
```json
{
  "players": [
    {"rank": 1, "id": 123, "name": "Player", "level": 2604, "vocation": 5,
     "world_id": 1, "worldName": "Elysian", "value": 293812753136}
  ],
  "totalCount": 500, "cachedAt": 1775679362230, "availableSeasons": []
}
```

**rubinidata.com returns:**
```json
{
  "highscores": {
    "category": "experience", "world": "elysian",
    "highscore_list": [
      {"rank": 1, "id": 0, "name": "Player", "vocation": "Elite Knight",
       "world_id": 1, "world_name": "Elysian", "level": 2503, "value": 260936620480}
    ]
  }
}
```

**Adapter logic:**
- `highscores.highscore_list[]` → `players[]`
- `vocation` is string in rubinidata → map to vocation ID int
- `world_name` → `worldName`
- `value` is number in rubinidata, can be number or string in upstream — keep as number
- **Missing:** `totalCount` — set to len(players)
- **Missing:** `cachedAt` — set to current time millis
- **Missing:** `availableSeasons` — set to empty []

**Note on vocation mapping (rubinidata → upstream int):**
```go
var vocationNameToID = map[string]int{
    "None": 1, "Knight": 5, "Elite Knight": 5, "Paladin": 4, "Royal Paladin": 4,
    "Sorcerer": 2, "Master Sorcerer": 2, "Druid": 3, "Elder Druid": 3,
    "Monk": 9, "Exalted Monk": 9,
}
```

**Available categories on rubinidata (10/20):**
achievements, magic, shielding, club, axe, fist, experience, distance, sword, fishing

**Missing categories:**
dromelevel, linked_tasks, exp_today, battlepass, charmunlockpoints, prestigepoints, totalweeklytasks, totalbountypoints, charmtotalpoints, bosstotalpoints

### 7. Kill Statistics

```
rubinot.com.br:  GET /api/killstats?world={worldId}
rubinidata.com:  GET /v1/killstatistics/{worldName}
```

**rubinot.com.br returns:**
```json
{
  "entries": [
    {"race_name": "Demon", "players_killed_24h": 150, "creatures_killed_24h": 2500,
     "players_killed_7d": 1200, "creatures_killed_7d": 18000}
  ],
  "totals": {"players_killed_24h": 5000, "creatures_killed_24h": 50000,
             "players_killed_7d": 40000, "creatures_killed_7d": 400000}
}
```

**rubinidata.com returns:**
```json
{
  "killstatistics": {
    "entries": [
      {"race": "Demon", "killed_players_last_day": 150, "killed_by_players_last_day": 2500,
       "killed_players_last_week": 1200, "killed_by_players_last_week": 18000}
    ]
  }
}
```

**Adapter logic:**
- `killstatistics.entries[]` → `entries[]` with field renames:
  - `race` → `race_name`
  - `killed_players_last_day` → `players_killed_24h`
  - `killed_by_players_last_day` → `creatures_killed_24h`
  - `killed_players_last_week` → `players_killed_7d`
  - `killed_by_players_last_week` → `creatures_killed_7d`
- Compute `totals` by summing all entries

### 8. Deaths

```
rubinot.com.br:  GET /api/deaths?world={worldId}&page={page}
rubinidata.com:  GET /v1/deaths/{worldName}?page={page}
```

**rubinot.com.br returns:**
```json
{
  "deaths": [
    {"player_id": 0, "time": "08.04.2026, 18:31:48", "level": 62,
     "killed_by": "monster", "is_player": 0, "mostdamage_by": "monster2",
     "mostdamage_is_player": 0, "victim": "PlayerName", "world_id": 1}
  ],
  "pagination": {"currentPage": 1, "totalPages": 5, "totalCount": 100, "itemsPerPage": 20}
}
```

**rubinidata.com returns:**
```json
{
  "deaths": {
    "entries": [
      {"name": "PlayerName", "level": 62,
       "killers": [{"name": "monster", "player": false}, {"name": "monster2", "player": false}],
       "time": "08.04.2026, 18:31:48", "datetime": "2026-04-08T18:31:48-03:00",
       "world_id": 0, "player_id": 0}
    ]
  }
}
```

**Adapter logic:**
- `deaths.entries[]` → `deaths[]`
- `name` → `victim`
- `killers[0].name` → `killed_by`, `killers[0].player` → `is_player` (as 0/1 int)
- `killers[1].name` → `mostdamage_by` (if exists), `killers[1].player` → `mostdamage_is_player`
- `time` field format matches — pass through
- **Missing:** `pagination` — rubinidata pagination info not in the response body. Set totalPages=1, currentPage=1, totalCount=len(deaths), itemsPerPage=len(deaths).

### 9. Banishments

```
rubinot.com.br:  GET /api/bans?world={worldId}&page={page}
rubinidata.com:  GET /v1/banishments/{worldName}?page={page}
```

**rubinot.com.br returns:**
```json
{
  "bans": [
    {"account_id": 1, "account_name": "AccName", "main_character": "CharName",
     "reason": "Reason", "banned_at": "2026-04-01", "expires_at": "2026-05-01",
     "banned_by": "GM", "is_permanent": false}
  ],
  "totalCount": 50, "totalPages": 3, "currentPage": 1, "cachedAt": 1775679362230
}
```

**rubinidata.com returns:**
```json
{
  "banishments": {
    "entries": [
      {"account_id": 0, "account_name": "AccName", "character": "CharName",
       "reason": "Reason", "banned_at": "2026-04-01", "expires_at": "2026-05-01",
       "banned_by": "GM", "is_permanent": false}
    ]
  }
}
```

**Adapter logic:**
- `banishments.entries[]` → `bans[]`
- `character` → `main_character`
- **Missing:** `totalCount`, `totalPages`, `currentPage`, `cachedAt` — derive from len or set defaults

### 10. Transfers

```
rubinot.com.br:  GET /api/transfers?page={page}
rubinidata.com:  GET /v1/transfers?page={page}&from={world}&to={world}
```

**rubinot.com.br returns:**
```json
{
  "transfers": [
    {"id": 1, "player_id": 2, "player_name": "Name", "player_level": 100,
     "from_world_id": 1, "to_world_id": 2, "from_world": "Elysian",
     "to_world": "Bellum", "transferred_at": "2026-04-08T15:30:21-03:00"}
  ],
  "totalResults": 200, "totalPages": 10, "currentPage": 1
}
```

**rubinidata.com returns:**
```json
{
  "transfers": {
    "entries": [
      {"name": "Name", "level": 100, "from_world": "Elysian",
       "to_world": "Bellum", "transfer_date": "2026-04-08T15:30:21-03:00"}
    ]
  }
}
```

**Adapter logic:**
- `transfers.entries[]` → `transfers[]`
- `name` → `player_name`, `level` → `player_level`
- `transfer_date` → `transferred_at`
- **Missing:** `id`, `player_id`, `from_world_id`, `to_world_id` — set to 0
- **Missing:** `totalResults`, `totalPages`, `currentPage` — derive from len or set defaults

### 11. Boosted

```
rubinot.com.br:  GET /api/boosted
rubinidata.com:  GET /v1/boosted
```

**rubinot.com.br returns:**
```json
{
  "boss": {"id": 2683, "name": "Tropical Desolator", "looktype": 1589},
  "monster": {"id": 1322, "name": "Twisted Shaper", "looktype": 932}
}
```

**rubinidata.com returns:**
```json
{
  "boosted": {
    "creature": {"name": "Twisted Shaper", "image_url": "/v1/outfit?type=932&...", 
                 "id": 1322, "looktype": 932},
    "boss": {"name": "Tropical Desolator", "image_url": "/v1/outfit?type=1589&...",
             "id": 2683, "looktype": 1589}
  }
}
```

**Adapter logic:**
- `boosted.creature` → `monster` (rename key, drop `image_url`)
- `boosted.boss` → `boss` (drop `image_url`)

## Batch Endpoint Synthesis

Batch endpoints don't exist on rubinidata.com. The RubinidataClient synthesizes them by fanning out individual calls.

### Characters Batch
- Input: `{"names": ["char1", "char2", ...]}` (max 500)
- Fan out: N concurrent calls to `/v1/characters/{name}`
- Concurrency: semaphore of 10 to avoid overwhelming rubinidata
- Each response adapted to `characterAPIResponse` shape
- Failures silently skipped (matching CDP batch behavior)
- Return combined JSON array

### Guilds Batch
- Input: `{"names": ["guild1", "guild2", ...]}` (max 200)
- Fan out: N concurrent calls to `/v1/guild/{name}`
- Same concurrency/error handling as characters

### Killstatistics Batch
- Input: `{"world_ids": [1, 2, ...]}` (max 50)
- Resolve world IDs to names via validator
- Fan out: N concurrent calls to `/v1/killstatistics/{worldName}`
- Same concurrency/error handling

## Path Routing

The RubinidataClient receives the same upstream path that would go to rubinot.com.br (e.g. `/api/characters/search?name=Bubble`) and must route to the correct rubinidata.com endpoint.

```go
func (c *RubinidataClient) translatePath(upstreamPath string) (string, error) {
    // /api/worlds -> /v1/worlds
    // /api/worlds/{name} -> /v1/world/{name}
    // /api/characters/search?name=X -> /v1/characters/{X}
    // /api/guilds?world=X&page=Y -> /v1/guilds/{worldName}?page=Y
    // /api/guilds/{name} -> /v1/guild/{name}
    // /api/highscores?world=X&category=Y&vocation=Z -> /v1/highscores?world={name}&category=Y&vocation=Z
    // /api/killstats?world=X -> /v1/killstatistics/{worldName}
    // /api/deaths?world=X&page=Y -> /v1/deaths/{worldName}?page=Y
    // /api/bans?world=X&page=Y -> /v1/banishments/{worldName}?page=Y
    // /api/transfers?page=X -> /v1/transfers?page=X
    // /api/boosted -> /v1/boosted
}
```

Note: rubinot.com.br uses world IDs (integers) in query params, but rubinidata.com uses world names. The translator needs access to the world ID-to-name mapping from the validator.

## Readiness Probe

When `UPSTREAM_PROVIDER=rubinidata`:
- Skip CDP/FlareSolverr initialization entirely (no Chrome needed)
- Readiness probe always returns 200 OK
- No `cfBlocked` flag management

## Workers to Keep Disabled

These workers call endpoints that don't exist on rubinidata.com:
- `news` — `/api/news` doesn't exist on rubinidata
- `newsticker` — same
- `events` — `/api/events` doesn't exist
- `auctions-current` — `/api/bazaar` doesn't exist
- `auctions-enrichment` — same
- `auction-history` — same
- `maintenance` — `/api/maintenance` doesn't exist
- `upstream-schema-monitor` — uses raw upstream proxy
- `daily-snapshot` — uses multiple endpoints, some missing

Workers that CAN run with rubinidata bridge:
- `name-change-detection` — `found_by_old_name` now available from rubinidata
- `world:online`, `world-online-snapshot`, `world:info`, `discovery:worlds` — use worlds/world endpoints
- `discovery:categories` — hardcode the 10 available categories
- `highscores-fast`, `highscores-slow` — only 10 of 20 categories will work
- `killstats:global` — uses killstatistics
- `deaths` (per world), `deaths-global` — uses deaths endpoint
- `guilds` — uses guilds list + guild detail
- `transfers` — uses transfers endpoint
- `banishments-global` — uses banishments endpoint
- `boosted` — uses boosted endpoint
- `character-enrichment` — synthesized from individual character calls (missing some fields)
- `guilds/batch` — synthesized from individual guild calls
- `killstatistics/batch` — synthesized from individual calls

## Testing Strategy

Since this is a throwaway branch:
- Manual testing: run locally with `UPSTREAM_PROVIDER=rubinidata`, hit each endpoint
- Verify response format matches what rubinot-api workers expect
- No automated tests (throwaway branch)
- Deploy to cluster, enable workers one at a time, monitor logs

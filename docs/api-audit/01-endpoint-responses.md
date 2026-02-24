# Endpoint Response Analysis

**Tested against:** api.rubinot.dev v1.3.1 (commit: ebe7b00)
**Date:** 2026-02-24 (post-fix deployment)

## System Endpoints

### GET /ping
```json
{"message":"pong"}
```
- **Time:** instant | **Status:** 200

### GET /healthz
```json
{"status":"ok"}
```
- **Time:** instant | **Status:** 200

### GET /versions
```json
{"commit":"ebe7b00","service":"rubinot-data","version":"v1.3.1"}
```
- **Time:** instant | **Status:** 200
- Commit SHA now populated (was "unknown" in previous audit)

---

## Worlds

### GET /v1/worlds
- **Time:** 3.1s | **Status:** 200
- **14 worlds** returned, all online, 16,736 total players
- Fields: `name`, `status`, `players_online` (int), `location`, `pvp_type`

Sample:
```json
{"name":"Belaria","status":"online","players_online":1458,"location":"South America","pvp_type":"Open PvP"}
```

### GET /v1/world/Belaria
- **Time:** 2.4s | **Status:** 200
- Structure: `world.name`, `world.info` (nested object), `world.players_online` (array of 1,459 players)
- `world.info` contains: `status` ("online"), `players_online` (1459), `location`, `pvp_type`, `creation_date`
- Each online player: `name`, `level`, `vocation`
- `info.status` is now lowercase "online" (was capitalized "Online" — FIXED)

---

## Character

### GET /v1/character/Rubini%20GM — 404
```json
{"information":{"status":{"http_code":404,"error":20004,"message":"character not found"}}}
```

### GET /v1/character/Abadias%20On%20Belaria — 200
- **Time:** 2.2s | **Status:** 200
- `character_info`: name, sex, vocation, level (966), achievement_points (273), world, residence, houses, last_login, account_status
- `deaths`: 2 entries with `time`, `level`, `killers`
- `account_information`: **empty `{}`** — not parsing account data

---

## Guilds

### GET /v1/guilds/Belaria — FIXED
- **Time:** 2.4s | **Status:** 200
- **212 active guilds, 0 formation** (was returning empty arrays)
- Each entry: `name`, `logo_url`, optional `description`
- `logo_url` uses relative paths: `./images/guilds/default.gif?v=1745624827`

Sample:
```json
{"name":"Panq Alliance","logo_url":"./images/guilds/default.gif?v=1745624827","description":"NOVA GUILD ASCENDED BELARIA..."}
```

### GET /v1/guild/Reapers
- **Time:** 2.2s | **Status:** 200
- Guild: Reapers, World: Tenebrium, Founded: 2025-11-13, Active: True
- 3 members, 0 invitees
- No header row bug for this guild
- 3rd member ("Lady Of King") missing `rank` field

Sample member:
```json
{"name":"King Of Lady","rank":"Leader","vocation":"Elite Knight","level":1193,"joined":"2025-11-13T03:00:00Z","status":"offline","is_online":false}
```

---

## Houses

### GET /v1/houses/Belaria/Thais
- **Time:** 4.3s | **Status:** 200
- **129 houses, 3 guildhalls**
- Fields: `house_id`, `name`, `size`, `rent`, `status`, `rented` (bool), `auctioned` (bool)

### GET /v1/house/Belaria/1986
- **Time:** 2.3s | **Status:** 200
- House: "Alai Flats, Flat 01", Town: Venore
- `paid_until` now ISO 8601: `"2026-02-27T13:32:13Z"` (was raw string — FIXED)
- Owner: `{"name":"Tankamax","paid_until":"2026-02-27T13:32:13Z"}`

### Invalid house IDs (1, 35001) — 404
- **Time:** 34-43s — FlareSolverr timeout on nonexistent houses

---

## Highscores

### GET /v1/highscores/Belaria/experience/all/1
- **Time:** 2.6s | **Status:** 200
- 50 entries per page, 1,000 total records, 20 pages
- Fields: `rank` (int), `name`, `vocation`, `world`, `level` (int), `value` (int)
- All types correct

### GET /v1/highscores/Belaria (redirect)
- **Time:** 0.14s | **Status:** 302 to `/v1/highscores/Belaria/experience/all/1`

---

## Kill Statistics

### GET /v1/killstatistics/Belaria
- **Time:** 2.8s | **Status:** 200
- **1,480 creature entries**
- Fields: `race`, `last_day_players_killed`, `last_day_killed`, `last_week_players_killed`, `last_week_killed`
- Totals: 1,145 day kills by players, 3,772,348 day creature kills

---

## News

### GET /v1/news/latest
- **Time:** 2.3s | **Status:** 200
- 1 entry from 2024-07-01
- Fields: `date`, `title`, `category`, `type`
- **Missing: `id`, `url`** (archive has both)

### GET /v1/news/newsticker
- **Time:** 2.6s | **Status:** 200
- 5 entries, most recent 2026-01-26
- Fields: `id`, `date`, `title`, `category`, `type`
- **Missing: `url`** (archive has it)

### GET /v1/news/archive?days=30
- **Time:** 2.1s | **Status:** 200
- 1 entry with all fields: `id`, `date`, `title`, `category`, `type`, `url`
- `url`: `"https://rubinot.com.br/?news/archive/140"`

### GET /v1/news/id/140
- **Time:** 2.5s | **Status:** 200
- Full article: `id`, `date`, `title`, `type`, `content` (plain text), `content_html`

---

## Events

### GET /v1/events/schedule?month=2&year=2026
- **Time:** 2.4s | **Status:** 200
- **16 day entries** for February 2026 (only days with events)
- No duplicate day numbers in this test (was reported as bug previously)
- Each day: `day`, `events`, `active_events`, `ending_events`
- `all_events`: ["A Piece of Cake", "Castle", "Double Skill", "Gaz'Haragoth"]

---

## Deaths

### GET /v1/deaths/Belaria
- **Time:** 11.7s | **Status:** 200
- **300 death entries**
- Fields: `date` (ISO 8601), `victim` {name, level}, `killers` (array), `is_pvp` (bool)

### GET /v1/deaths/Belaria?pvp=1&level=500 — BUG: FILTERS NOT APPLIED
- **Time:** 2.2s | **Status:** 200
- Response metadata: `"filters": {"min_level": 500, "pvp_only": true}`
- **Actual results: 238 non-PvP entries, 218 entries below level 500**
- Filters passed to upstream URL (`&level=500&pvp=1`) but upstream ignores them
- No server-side filtering applied after scraping

---

## Banishments

### GET /v1/banishments/Belaria
- **Time:** 12.4s | **Status:** 200
- 0 bans (may be legitimately empty for Belaria)

---

## Transfers

### GET /v1/transfers
- **Time:** 2.1s | **Status:** 200
- 50 entries per page, 1,000 total
- Fields: `player_name`, `level`, `former_world`, `destination_world`, `transfer_date`

---

## Auctions

### GET /v1/auctions/current/1
- **Time:** 3.2s | **Status:** 200
- 25 entries per page, 1,577 total, 64 pages
- `auction_id` is now **integer** (was string — FIXED)
- Status: "active" for current auctions

### GET /v1/auctions/history/1
- **Time:** 2.2s | **Status:** 200
- 25 entries per page, 15,568 total, 623 pages
- `status` is now **"ended"** (was "active" — FIXED)
- `bid_type`: "winning" for successful auctions

### GET /v1/auctions/164073
- **Time:** 4.2s | **Status:** 200
- Full detail: `auction_start`, `auction_end`, `bid_type`, `bid_value`, `status`

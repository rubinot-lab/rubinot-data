# Endpoint Response Analysis

**Tested against:** https://api.rubinot.dev on 2026-02-24

## System Endpoints

### `GET /`
```json
{"message": "rubinot-data api up"}
```
No envelope. Simple health indicator. OK.

### `GET /ping`
```json
{"message": "pong"}
```
No envelope. OK.

### `GET /healthz` / `GET /readyz`
```json
{"status": "ok"}
```
No envelope. OK for k8s probes.

### `GET /versions`
```json
{"commit": "unknown", "service": "rubinot-data", "version": "v0.2.0"}
```
`commit` is always "unknown" - build pipeline doesn't inject git SHA via ldflags.

---

## Worlds

### `GET /v1/worlds`
```json
{
  "worlds": {
    "total_players_online": 15603,
    "worlds": [
      {"name": "Auroria", "status": "online", "players_online": 1025, "location": "South America", "pvp_type": "Open PvP"},
      ...
    ]
  }
}
```
- 14 worlds returned
- All fields present and reasonable
- **OK**

### `GET /v1/world/Belaria`
```json
{
  "world": {
    "name": "Belaria",
    "info": {
      "status": "Online",
      "players_online": 1292,
      "location": "South America",
      "pvp_type": "Open PvP",
      "creation_date": "2025-06-11T03:00:00Z"
    },
    "players_online": [ {"name": "...", "level": 966, "vocation": "Royal Paladin"}, ... ]
  }
}
```
- 1292 online players listed
- **Minor:** `info.status` is "Online" (capitalized) while worlds list has "online" (lowercase)
- **OK**

---

## Character

### `GET /v1/character/Prensa`
```json
{
  "character": {
    "character_info": {
      "name": "Prensa",
      "sex": "male",
      "vocation": "Exalted Monk",
      "level": 1475,
      "achievement_points": 224,
      "world": "Belaria",
      "residence": "Kazordoon",
      "houses": [{"house_id": 70, "name": "Loot Lane 1 (Shop)", "world": "Belaria"}],
      "guild": {"name": "Ascended Belaria", "rank": "Member"},
      "last_login": "2026-02-24T18:35:34Z",
      "account_status": "VIP Account"
    },
    "deaths": [
      {"time": "2026-02-20T02:34:08Z", "level": 1470, "killers": ["roaming dread", "cyclursus"]},
      {"time": "2026-02-19T23:44:36Z", "level": 1472, "killers": ["roaming dread", "cyclursus"]}
    ],
    "account_information": {}
  }
}
```

**Missing fields (all omitted via omitempty):**
- `character_info.former_names` - parser not matching HTML labels
- `character_info.title` - parser not matching
- `character_info.unlocked_titles` - parser not matching
- `character_info.traded` - parser not matching
- `character_info.married_to` - parser not matching
- `character_info.comment` - parser not matching
- `character_info.is_banned` - parser not matching
- `character_info.former_worlds` - parser not matching
- `character_info.auction_url` - parser not matching
- `character_info.deletion_date` - parser not matching
- `account_information` - empty object `{}` (parser not extracting created/loyalty_title)
- `other_characters` - entirely missing

**Deaths missing field:**
- `assists` array not present (omitempty hides it when empty, but field is documented)
- `reason` field not present

---

## Guilds

### `GET /v1/guild/Ascended%20Belaria`

**BUG: Table header parsed as first member:**
```json
{
  "name": "Name and Title",
  "rank": "Rank [ sort]",
  "vocation": "Vocation [ sort]",
  "level": null,
  "joined": null,
  "status": "offline",
  "is_online": null
}
```
The header-skip check in `guild.go` looks at the wrong cell index.

**BUG: `is_online` is `null` for offline members instead of `false`:**
- 533 members have `is_online: null` (offline)
- 173 members have `is_online: true` (online)
- 0 members have `is_online: false`
- Go zero-value `false` + `omitempty` tag causes null serialization

**Missing guild metadata (all null):**
- `description` - not parsed from HTML
- `guildhall` - not parsed
- `homepage` - not parsed
- `open_applications` - not parsed
- `in_war` - not parsed
- `disband_date` / `disband_condition` - not parsed

### `GET /v1/guilds/Belaria`
```json
{
  "guilds": {
    "world": "Belaria",
    "active": [],
    "formation": []
  }
}
```
**BUG:** Returns empty arrays. Belaria definitely has active guilds (e.g. Ascended Belaria with 706 members). The HTML container-finding logic fails to match section headers, and the fallback doesn't find guild links.

---

## Houses

### `GET /v1/houses/Belaria/Venore`
```json
{
  "houses": {
    "world": "Belaria",
    "town": "Venore",
    "house_list": [
      {"house_id": 51, "name": "Dagger Alley 1", "size": 126, "rent": 200000, "status": "rented", "rented": true, "auctioned": false},
      ...
    ],
    "guildhall_list": [
      {"house_id": 50, "name": "Blessed Shield Guildhall", "size": 298, "rent": 500000, "status": "rented", "rented": true, "auctioned": false}
    ]
  }
}
```
- 68 houses + 2 guildhalls
- **OK**

### `GET /v1/house/Belaria/70`
```json
{
  "house": {
    "house_id": 70,
    "name": "Loot Lane 1 (Shop)",
    "world": "Belaria",
    "town": "Venore",
    "size": 198,
    "beds": 3,
    "rent": 600000,
    "status": "rented",
    "owner": {
      "name": "Prensa",
      "paid_until": "2 March 2026, 10:09:29 BRA"
    }
  }
}
```

**BUG: `paid_until` is a raw localized string, not ISO 8601.**
Expected: `"2026-03-02T13:09:29Z"` (or similar UTC)

**Missing owner fields:**
- `owner.level` - regex not matching HTML
- `owner.vocation` - regex not matching HTML
- `owner.moving_date` - regex not matching

**Missing `auction` object** - may be correct if house is not auctioned.

---

## Highscores

### `GET /v1/highscores/Belaria/experience/all/1`
```json
{
  "highscores": {
    "world": "Belaria",
    "category": "experience",
    "vocation": "(all)",
    "highscore_age": 0,
    "highscore_list": [
      {"rank": 1, "name": "Razer Ascended", "vocation": "Elder Druid", "world": "Belaria", "level": 1718, "value": 84277359367},
      ...
    ],
    "highscore_page": {"current_page": 1, "total_pages": 20, "total_records": 1000}
  }
}
```
- 50 entries per page, 1000 total
- `highscore_age` is 0 (should it indicate data freshness?)
- **Missing fields per entry:** `title`, `traded`, `auction_url` (documented in test but not in response)
- **OK otherwise**

### Redirects
- `/v1/highscores/Belaria` -> 302 to `/v1/highscores/Belaria/experience/all/1` **OK**
- `/v1/highscores/Belaria/magic` -> 302 to `/v1/highscores/Belaria/magic/all/1` **OK**

---

## Kill Statistics

### `GET /v1/killstatistics/Belaria`
```json
{
  "killstatistics": {
    "world": "Belaria",
    "entries": [
      {"race": "(elemental forces)", "last_day_players_killed": 31, "last_day_killed": 0, ...},
      ...
    ],
    "total": {"last_day_players_killed": 1051, "last_day_killed": 3593557, ...}
  }
}
```
- 1479 creature entries
- Totals present and reasonable
- **OK**

---

## News

### `GET /v1/news/latest`
```json
{
  "newslist": {
    "mode": "latest",
    "entries": [
      {"date": "2024-07-01T03:00:00Z", "title": "Conheça o RubinOT", "category": "news", "type": "article"}
    ]
  }
}
```
**Issues:**
- Only 1 entry and it's from 2024-07-01 (very old)
- Missing `id` field (present in newsticker/archive)
- Missing `url` field (present in archive)

### `GET /v1/news/newsticker`
```json
{
  "newslist": {
    "mode": "newsticker",
    "entries": [
      {"id": 1, "date": "2026-01-26T03:00:00Z", "title": "Event Schedule", "category": "Event Schedule", "type": "ticker"},
      ...
    ]
  }
}
```
- 5 entries, most recent from 2026-01-26
- Missing `url` field (present in archive entries)
- `category` duplicates the `title` for tickers

### `GET /v1/news/archive`
```json
{
  "newslist": {
    "mode": "archive",
    "archive_days": 90,
    "entries": [
      {"id": 140, "date": "2024-07-01T03:00:00Z", "title": "Conheça o RubinOT", "category": "news", "type": "article", "url": "https://rubinot.com.br/?news/archive/140"}
    ]
  }
}
```
- Only 1 article in the archive (the site may genuinely have few news articles)
- Has `id` and `url` fields unlike latest/newsticker

### `GET /v1/news/id/1`
```json
{
  "news": {
    "id": 1,
    "date": "2026-01-26T03:00:00Z",
    "title": "Event Schedule",
    "category": "Event Schedule",
    "type": "ticker",
    "content": "[Event Schedule] - O calendário de eventos...",
    "content_html": "<p>...</p>"
  }
}
```
- Returns a newsticker item (id=1 maps to ticker not article)
- `source` URL is `https://www.rubinot.com.br/?news` (generic, not specific)
- **OK**

---

## Events

### `GET /v1/events/schedule`
```json
{
  "events": {
    "month": "February",
    "year": 2026,
    "last_update": "2026-02-24T19:01:00Z",
    "days": [
      {"day": 26, "events": ["Castle"], "active_events": [], "ending_events": ["Castle"]},
      {"day": 1, "events": ["Castle", "Gaz'Haragoth"], ...},
      ...
    ],
    "all_events": ["A Piece of Cake", "Castle", "Double Skill", "Gaz'Haragoth"]
  }
}
```

**Issues:**
- **Duplicate day numbers:** days 1, 2, 4, 8, 26 each appear twice. This is because the calendar shows days from adjacent months (Jan/March days bleeding into February view)
- Days are NOT sorted chronologically
- No way to distinguish which month a day belongs to when duplicates exist
- Day 26 appears at index 0 (January 26?) and index 17 (February 26?)

---

## Deaths

### `GET /v1/deaths/Belaria`
```json
{
  "deaths": {
    "world": "Belaria",
    "filters": {},
    "entries": [
      {"date": "2026-02-24T19:01:17Z", "victim": {"name": "Mario Duplantier", "level": 467}, "killers": ["flimsy lost soul"], "is_pvp": false},
      ...
    ],
    "total_deaths": 300
  }
}
```
- 300 entries returned
- Filters work (tested with `?pvp=1&level=100`)
- Missing `assists` array per death entry
- **OK otherwise**

### `GET /v1/deaths/Belaria?pvp=1&level=100`
- Filters correctly applied: `"filters": {"min_level": 100, "pvp_only": true}`
- Only PvP deaths with victims level >= 100 returned
- **OK**

---

## Banishments

### `GET /v1/banishments/Belaria`
```json
{
  "banishments": {
    "world": "Belaria",
    "page": 1,
    "total_bans": 0,
    "entries": []
  }
}
```
- 0 bans - may be legitimate if Belaria has no active bans
- **OK (needs verification with a world that has bans)**

---

## Transfers

### `GET /v1/transfers`
```json
{
  "transfers": {
    "filters": {},
    "page": 1,
    "total_transfers": 1000,
    "entries": [
      {"player_name": "Jeffzin Palaa", "level": 302, "former_world": "Serenian III", "destination_world": "Serenian II", "transfer_date": "2026-02-24T19:00:57Z"},
      ...
    ]
  }
}
```
- 50 entries per page, 1000 total
- Recent transfers (same day)
- **OK**

---

## Auctions

### `GET /v1/auctions/current/1`
```json
{
  "auctions": {
    "type": "current",
    "page": 1,
    "total_results": 1603,
    "total_pages": 65,
    "entries": [
      {"auction_id": "164960", "character_name": "Fersefone", "level": 624, "vocation": "Elite Knight", "sex": "Female", "world": "Elysian", "bid_type": "minimum", "bid_value": 500, "auction_end": "2026-02-24T20:00:00Z", "status": "active"},
      ...
    ]
  }
}
```
- 25 entries per page, 1603 total
- **Issue:** `auction_id` is string `"164960"` not integer `164960`
- **OK otherwise**

### `GET /v1/auctions/history/1`
```json
{
  "auctions": {
    "type": "history",
    "page": 1,
    "total_results": 15582,
    "total_pages": 624,
    "entries": [
      {"auction_id": "136030", ..., "status": "active"},
      ...
    ]
  }
}
```
- **BUG:** History entries show `"status": "active"` - should be "ended" or "cancelled"
- First entry has `bid_type: "minimum"` suggesting no bids were placed
- **Issue:** `auction_id` string type

### `GET /v1/auctions/1`
```json
{
  "auction": {
    "auction_id": "1",
    "character_name": "Unstopabble",
    "level": 237,
    "vocation": "Master Sorcerer",
    "sex": "Male",
    "world": "Bellum",
    "auction_start": "2023-05-21T23:42:00Z",
    "auction_end": "2023-06-20T23:41:00Z",
    "bid_type": "minimum",
    "bid_value": 200,
    "status": "active"
  }
}
```
- Auction from 2023 still shows `status: "active"` (clearly ended long ago)
- Source URL tried `/?currentcharactertrades/1` (current first, then falls back to past)
- **BUG:** Status not correctly determined for historical/ended auctions
- **Issue:** `auction_id` string type

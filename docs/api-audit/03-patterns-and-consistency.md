# Patterns and Consistency Analysis

## Response Envelope

All v1 endpoints wrap responses in a consistent envelope:
```json
{
  "information": {
    "api": {"version": 1, "release": "v0.2.0", "commit": "unknown"},
    "timestamp": "2026-02-24T19:01:00Z",
    "status": {"http_code": 200, "message": "ok"},
    "sources": ["https://..."]
  },
  "<payload_key>": { ... }
}
```

**Verdict:** Consistent across all v1 endpoints. System endpoints (`/`, `/ping`, `/healthz`, `/readyz`, `/versions`) intentionally skip the envelope.

---

## Date/Time Formatting

| Endpoint | Field | Format | Correct? |
|----------|-------|--------|----------|
| character | `last_login` | ISO 8601 UTC | Yes |
| character | `deaths[].time` | ISO 8601 UTC | Yes |
| guild | `founded` | ISO 8601 UTC | Yes |
| guild | `members[].joined` | ISO 8601 UTC | Yes |
| world | `creation_date` | ISO 8601 UTC | Yes |
| house | `owner.paid_until` | **Raw string "2 March 2026, 10:09:29 BRA"** | **NO** |
| news | `date` | ISO 8601 UTC | Yes |
| events | `last_update` | ISO 8601 UTC | Yes |
| deaths | `date` | ISO 8601 UTC | Yes |
| transfers | `transfer_date` | ISO 8601 UTC | Yes |
| auctions | `auction_start`, `auction_end` | ISO 8601 UTC | Yes |

**Pattern:** All dates are ISO 8601 UTC **except `house.owner.paid_until`** which is a raw localized string.

---

## Null vs Omitted vs Empty

The API uses `omitempty` JSON tags throughout, which creates inconsistency:

| Scenario | Current Behavior | Expected Behavior |
|----------|-----------------|-------------------|
| Boolean false | Omitted (null in some cases) | Should be `false` |
| Empty string | Omitted | OK for optional fields |
| Zero integer | Omitted | Problematic for meaningful zeros |
| Empty array | Omitted | Should be `[]` |
| Empty object | `{}` (account_information) | Should be omitted or populated |

**Key problems:**
- `guild.members[].is_online: null` when offline (should be `false`)
- `guild.members[].rank: null` for members without custom rank (should be empty string or a default)
- `guild.members[].level: null` for header row (this is the header bug)
- `character.account_information: {}` (empty object serialized, inconsistent with omitting)

**Recommendation:** Audit all `omitempty` tags. Boolean fields that represent state (is_online, is_pvp, traded, rented, auctioned) should always be present. Consider removing `omitempty` from these fields.

---

## ID Type Consistency

| Entity | Field | Type | Expected |
|--------|-------|------|----------|
| House | `house_id` | `int` | Correct |
| News | `id` | `int` | Correct |
| Auction | `auction_id` | **`string`** | Should be `int` |
| Highscore | `rank` | `int` | Correct |

**Pattern break:** `auction_id` is the only ID serialized as a string. All other IDs are integers.

---

## Pagination Consistency

| Endpoint | Page Field | Total Field | Items/Page |
|----------|-----------|-------------|------------|
| Highscores | `highscore_page.current_page` | `highscore_page.total_pages`, `total_records` | 50 |
| Auctions current | `page` | `total_pages`, `total_results` | 25 |
| Auctions history | `page` | `total_pages`, `total_results` | 25 |
| Transfers | `page` | `total_transfers` | 50 |
| Banishments | `page` | `total_bans` | ? |
| Deaths | - | `total_deaths` | 300 (all) |

**Inconsistencies:**
- Highscores nests pagination in `highscore_page` sub-object, others use flat fields
- Field naming varies: `total_records` vs `total_results` vs `total_transfers` vs `total_bans` vs `total_deaths`
- Deaths returns all 300 entries without pagination
- Some endpoints report `total_pages`, others don't

**Recommendation:** Standardize pagination to a common pattern:
```json
{
  "page": 1,
  "total_pages": 20,
  "total_items": 1000,
  "items_per_page": 50
}
```

---

## Naming Conventions

**Snake_case used consistently** across all field names. Good.

**Inconsistencies found:**
- `players_online` (world) vs `players_online` (guild) - same name, different context (count vs list) in world detail
- `highscore_list` (array) vs `house_list` / `guildhall_list` - consistent pattern
- `total_players_online` (worlds) vs `players_online` (world info) - number in both, OK

---

## Data Quality Observations

### Worlds
- 14 worlds active, all South America, all Open PvP
- Total online: ~15,600 players
- Data looks correct and fresh

### Transfers
- `total_transfers: 1000` is suspiciously round - likely a hard cap on the data
- Most recent transfer is same-day, data is fresh

### Deaths
- `total_deaths: 300` is also a hard cap
- Data is real-time (most recent death was minutes before the request)
- Filtered queries also return `total_deaths: 300` which seems like the same cap applied to filtered results

### Auctions
- Current: 1603 active auctions across 65 pages
- History: 15,582 past auctions across 624 pages
- Data volumes look reasonable

### News
- Very sparse: only 1 article and 5 tickers
- This appears to be a characteristic of the source site, not a parsing bug

---

## Performance Notes

All endpoints responded successfully within the timeout. The API proxies requests through FlareSolverr to bypass Cloudflare challenges on the source site, so response times depend on upstream scraping speed. No timeout or error responses were encountered during this audit.

---

## Priority Fix Order

1. **Guild header row bug** (BUG-01) - corrupts data, easy fix
2. **Guilds list empty** (BUG-02) - entire endpoint broken
3. **Auction status wrong** (BUG-03) - misleading data
4. **Character missing fields** (BUG-04) - most visible endpoint, many missing fields
5. **House paid_until format** (BUG-05) - breaks date parsing for consumers
6. **Guild is_online null** (BUG-07) - affects boolean logic for consumers
7. **Events duplicate days** (BUG-09) - confusing data structure
8. **Auction ID type** (BUG-10) - inconsistent with other IDs
9. **House owner fields** (BUG-06) - missing useful data
10. **News latest fields** (BUG-08) - incomplete response
11. Remaining medium/low issues

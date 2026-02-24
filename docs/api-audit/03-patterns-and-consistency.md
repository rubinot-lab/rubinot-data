# Patterns and Consistency Analysis

**Updated:** 2026-02-24 (post-fix deployment, v1.3.1)

## Response Envelope

All v1 endpoints wrap responses in a consistent envelope:
```json
{
  "information": {
    "api": {"version": 1, "release": "v1.3.1", "commit": "ebe7b00"},
    "timestamp": "2026-02-24T21:22:22Z",
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
| house | `owner.paid_until` | ISO 8601 UTC | **Yes (FIXED)** |
| news | `date` | ISO 8601 UTC | Yes |
| events | `last_update` | ISO 8601 UTC | Yes |
| deaths | `date` | ISO 8601 UTC | Yes |
| transfers | `transfer_date` | ISO 8601 UTC | Yes |
| auctions | `auction_start`, `auction_end` | ISO 8601 UTC | Yes |

**Pattern:** All dates are now consistently ISO 8601 UTC.

---

## Null vs Omitted vs Empty

The API uses `omitempty` JSON tags throughout:

| Scenario | Current Behavior | Notes |
|----------|-----------------|-------|
| Boolean false (is_online) | Now serialized as `false` | Improved |
| Empty string | Omitted | OK for optional fields |
| Zero integer | Omitted | Could be problematic for meaningful zeros |
| Empty array | Omitted or `[]` | Inconsistent |
| Empty object | `{}` (account_information) | Should be omitted or populated |

**Remaining issue:**
- `character.account_information: {}` (empty object, should be omitted or populated)
- Guild members sometimes missing `rank` field entirely

---

## ID Type Consistency

| Entity | Field | Type | Status |
|--------|-------|------|--------|
| House | `house_id` | `int` | Correct |
| News | `id` | `int` | Correct |
| Auction | `auction_id` | `int` | **Correct (FIXED)** |
| Highscore | `rank` | `int` | Correct |

All IDs now consistently use integer types.

---

## Pagination Consistency

| Endpoint | Page Field | Total Field | Items/Page |
|----------|-----------|-------------|------------|
| Highscores | `highscore_page.current_page` | `highscore_page.total_pages`, `total_records` | 50 |
| Auctions current | `page` | `total_pages`, `total_results` | 25 |
| Auctions history | `page` | `total_pages`, `total_results` | 25 |
| Transfers | `page` | `total_transfers` | 50 |
| Banishments | `page` | `total_bans` | ? |
| Deaths | - | `total_deaths` | 300 (all, no pagination) |

**Inconsistencies:**
- Highscores nests pagination in `highscore_page` sub-object, others use flat fields
- Field naming varies: `total_records` vs `total_results` vs `total_transfers` vs `total_bans` vs `total_deaths`
- Deaths returns all 300 entries without pagination
- Some endpoints report `total_pages`, others don't

---

## Response Time Analysis

| Tier | Range | Endpoints | Count |
|------|-------|-----------|-------|
| Instant | <0.2s | /ping, /healthz, /versions, highscores redirect | 4 |
| Fast | 2-4s | worlds, world, character, guilds, guild, highscores, kills, news, events, auctions, transfers | 16 |
| Slow | 4-12s | houses list, deaths, banishments, auction detail | 4 |
| Very slow | 30-43s | house 404 (invalid IDs) | edge case |

**Notes:**
- Most scraping endpoints take 2-4s (FlareSolverr overhead)
- Deaths and banishments are slower (~12s) — possibly larger HTML pages
- House 404s with invalid IDs are extremely slow (30-43s) due to full FlareSolverr timeout before determining the house doesn't exist

---

## Data Quality Observations

### Worlds
- 14 worlds active, all South America
- Mix of PvP types: Open PvP, Optional PvP, Retro Open PvP
- Total online: ~16,700 players
- Data fresh and correct

### Guilds (FIXED)
- 212 active guilds on Belaria
- Guild detail returns proper member data
- Logo URLs are relative paths (minor issue)

### Transfers
- `total_transfers: 1000` is a hard cap on the data
- Most recent transfer is same-day, data is fresh

### Deaths
- `total_deaths: 300` is a hard cap
- Data is real-time (most recent death within minutes)
- **Filters not applied** — critical bug, returns unfiltered results

### Auctions
- Current: 1,577 active auctions across 64 pages
- History: 15,568 past auctions across 623 pages
- Status now correctly shows "ended" for history

### News
- Very sparse: only 1 article and 5 tickers
- Characteristic of the source site, not a parsing bug

---

## Priority Fix Order (Remaining)

1. **Deaths filters not applied** (BUG-NEW-01) — critical, misleading data
2. **Character missing fields** (BUG-04) — high visibility endpoint
3. **News latest missing id/url** (BUG-08) — inconsistent with other news endpoints
4. **Guild members missing rank** (BUG-NEW-02) — data quality
5. **Newsticker missing url** (BUG-13) — consistency
6. **Guild logo_url relative** (BUG-NEW-03) — consumer usability
7. **House 404 slow** (BUG-NEW-04) — performance edge case

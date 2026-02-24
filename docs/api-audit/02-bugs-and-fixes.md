# Bugs and Recommended Fixes

## Critical

### BUG-01: Guild members list includes table header as first member

**Endpoint:** `GET /v1/guild/:name`
**Symptom:** First member entry is:
```json
{"name": "Name and Title", "rank": "Rank [ sort]", "vocation": "Vocation [ sort]", "level": null}
```
**Root Cause:** `guild.go` header-skip check looks at the wrong cell index. It checks if `cells.Eq(0)` equals "Rank", but the actual header row has "Name and Title" in that position.
**Fix:** Update the header detection to check for multiple header patterns or skip the first `<tr>` in the members table unconditionally.

---

### BUG-02: Guilds list returns empty arrays

**Endpoint:** `GET /v1/guilds/:world`
**Symptom:** `{"active": [], "formation": []}` for Belaria (which has guilds)
**Root Cause:** `guilds.go` container-finding logic (`findContainerByHeaders`) fails to match the HTML section headers on the live site. The fallback parser also doesn't find guild links in the expected positions.
**Fix:** Debug against the live HTML structure. Capture the actual HTML from `https://www.rubinot.com.br/?subtopic=guilds&world=15` and update the header text matching and table parsing selectors.

---

### BUG-03: Auction history entries show status "active"

**Endpoint:** `GET /v1/auctions/history/:page`, `GET /v1/auctions/:id`
**Symptom:** Auctions that have clearly ended (from 2023, or in history list) still show `"status": "active"`.
**Root Cause:** The status parser doesn't correctly determine ended/cancelled status from the HTML. It likely defaults to "active" when no explicit ended indicator is found.
**Fix:** For history list entries, default status should be "ended". For detail view, parse the actual auction status text from the HTML (look for "finished", "cancelled", or bid outcome indicators).

---

## High

### BUG-04: Character missing most fields

**Endpoint:** `GET /v1/character/:name`
**Symptom:** Missing: `former_names`, `title`, `unlocked_titles`, `traded`, `married_to`, `comment`, `is_banned`, `former_worlds`, `auction_url`, `deletion_date`. `account_information` is `{}`. `other_characters` entirely absent.
**Root Cause:** The parsing logic exists in `character.go` but the HTML row label matching doesn't match the live site's actual labels (likely case sensitivity, extra whitespace, or structural differences). Combined with `omitempty` JSON tags, unmatched fields are silently omitted.
**Fix:** Capture the live HTML for a character page and compare the row labels against the expected patterns in the parser. Update the label matchers to handle the actual format.

---

### BUG-05: House `paid_until` is raw localized string

**Endpoint:** `GET /v1/house/:world/:id`
**Symptom:** `"paid_until": "2 March 2026, 10:09:29 BRA"` instead of ISO 8601
**Root Cause:** `house.go` captures the regex match but doesn't convert to UTC via `parseRubinotDateTimeToUTC()`.
**Fix:** Parse the captured date string through the date conversion function, similar to how character death times are parsed. Handle the "BRA" timezone identifier.

---

### BUG-06: House owner missing level, vocation, moving_date

**Endpoint:** `GET /v1/house/:world/:id`
**Symptom:** Owner object only has `name` and `paid_until`, missing `level`, `vocation`, `moving_date`.
**Root Cause:** Regex patterns in `house.go` don't match the actual HTML text. The patterns look for specific text like "level (\d+)" but the live HTML format differs.
**Fix:** Capture live HTML and update the regex patterns.

---

### BUG-07: Guild `is_online` is null instead of false for offline members

**Endpoint:** `GET /v1/guild/:name`
**Symptom:** 533 members have `"is_online": null`, 0 have `"is_online": false`
**Root Cause:** Go zero-value for `bool` is `false`, and the `omitempty` JSON tag treats `false` as empty, serializing it as `null` (pointer) or omitting it. The else branch doesn't explicitly set `IsOnline`.
**Fix:** Either:
- Change the domain field to `*bool` and explicitly set `IsOnline = &falseVal` in the else branch
- Remove `omitempty` from the `is_online` JSON tag so `false` is always serialized
- Preferred: remove `omitempty` since `is_online` should always be present

---

### BUG-08: News `/latest` returns incomplete entries

**Endpoint:** `GET /v1/news/latest`
**Symptom:** Only 1 entry from 2024-07-01. Missing `id` and `url` fields compared to archive entries.
**Root Cause:** The latest news page on the source site may only show a single article. The parser doesn't extract `id` or `url` for this mode.
**Fix:** Ensure the parser extracts the news `id` from the article link and constructs the `url`. If the site genuinely only has 1 article, this may be correct data but the missing fields are still a bug.

---

### BUG-09: Events schedule has duplicate day numbers

**Endpoint:** `GET /v1/events/schedule`
**Symptom:** Days 1, 2, 4, 8, 26 each appear twice in the array. Calendar shows days from adjacent months.
**Root Cause:** The event calendar HTML includes "overflow" days from the previous/next month. The parser captures all days without filtering to the requested month.
**Fix:** Either:
- Filter out days that don't belong to the requested month
- Add a `month` field to each day entry so consumers can distinguish
- Add a `date` (full YYYY-MM-DD) field instead of just `day`

---

### BUG-10: `auction_id` is string type everywhere

**Endpoints:** All auction endpoints
**Symptom:** `"auction_id": "164960"` (string) instead of `164960` (integer)
**Root Cause:** Domain model defines `AuctionID` as `string` instead of `int`.
**Fix:** Change to `int` in the domain model and parse as integer in the scraper. This is a breaking API change - consider versioning.

---

## Medium

### BUG-11: Highscores entries missing title, traded, auction_url

**Endpoint:** `GET /v1/highscores/:world/:category/:vocation/:page`
**Symptom:** Each entry only has `rank`, `name`, `vocation`, `world`, `level`, `value`. Missing `title`, `traded`, `auction_url`.
**Root Cause:** These fields exist in the domain model with `omitempty` but the parser doesn't extract them from the HTML (the highscores table may not include this data on the source site).
**Fix:** Verify if the source HTML contains this data. If not, remove from domain model/documentation. If yes, update parser.

---

### BUG-12: Deaths entries missing `assists` and `reason` fields

**Endpoint:** `GET /v1/deaths/:world`
**Symptom:** Each death only has `date`, `victim`, `killers`, `is_pvp`. No `assists` or `reason` array.
**Root Cause:** The source site may not provide assist/reason data in the deaths list, or the parser doesn't extract it.
**Fix:** Verify source HTML. If data exists, update parser. If not, remove from documentation.

---

### BUG-13: News newsticker entries missing `url` field

**Endpoint:** `GET /v1/news/newsticker`
**Symptom:** Entries have `id` but no `url`, unlike archive entries.
**Root Cause:** Parser doesn't construct URL for ticker entries.
**Fix:** Construct URL from id (e.g., `https://rubinot.com.br/?news/archive/{id}` or similar).

---

### BUG-14: Commit SHA always "unknown"

**Endpoint:** `GET /versions`, all `information.api.commit` envelopes
**Symptom:** `"commit": "unknown"`
**Root Cause:** Build pipeline doesn't inject git SHA via `-ldflags`.
**Fix:** Add `-ldflags "-X main.commit=$(git rev-parse --short HEAD)"` to the build step in CI/CD and Dockerfile.

---

## Low

### BUG-15: World status casing inconsistency

**Endpoint:** `GET /v1/worlds` vs `GET /v1/world/:name`
**Symptom:** Worlds list uses lowercase `"online"`, world detail uses capitalized `"Online"`.
**Fix:** Normalize to consistent casing (lowercase preferred).

### BUG-16: Newsticker `category` duplicates `title`

**Endpoint:** `GET /v1/news/newsticker`
**Symptom:** `{"title": "Event Schedule", "category": "Event Schedule"}` - category mirrors title.
**Root Cause:** Source site may not distinguish category from title for tickers.
**Fix:** Set category to `"ticker"` or `"newsticker"` for consistency if the source doesn't provide a separate category.

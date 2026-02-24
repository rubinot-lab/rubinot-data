# Bugs and Fixes Status

**Updated:** 2026-02-24 (post-fix deployment, v1.3.1)

## Fixed Bugs

### BUG-02: Guilds list returns empty arrays — FIXED
**Fix:** Data URI form submission trick. FlareSolverr navigates to a `data:text/html` URI containing an auto-submitting HTML form that POSTs to the target. Uses `document.forms[0].submit()` and strips `www.` to avoid redirect dropping POST data.
**Result:** 212 active guilds returned for Belaria.

### BUG-03: Auction history status always "active" — FIXED
**Result:** History entries now show `status: "ended"`, `bid_type: "winning"`.

### BUG-05: House `paid_until` raw string — FIXED
**Result:** `paid_until` now ISO 8601 (`"2026-02-27T13:32:13Z"`).

### BUG-09: Events duplicate day numbers — FIXED
**Result:** 16 unique day entries for February 2026, no duplicates observed.

### BUG-10: `auction_id` is string type — FIXED
**Result:** `auction_id` now integer in all auction endpoints.

### BUG-14: Commit SHA always "unknown" — FIXED
**Result:** `/versions` shows `"commit":"ebe7b00"`.

### BUG-15: World status casing inconsistency — FIXED
**Result:** Both worlds list and world detail now use lowercase `"online"`.

---

## Remaining / New Bugs

### BUG-NEW-01: Deaths filters not applied (CRITICAL)

**Endpoint:** `GET /v1/deaths/:world?pvp=1&level=500`
**Symptom:** Response metadata shows `"filters": {"min_level": 500, "pvp_only": true}` but actual entries are unfiltered:
- 238 out of 300 entries have `is_pvp: false`
- 218 out of 300 entries have victim level below 500
**Root Cause:** Filters are appended to the upstream URL as GET params (`&level=500&pvp=1`) but the upstream rubinot.com.br `latestdeaths` page ignores these params. No server-side filtering is applied after scraping.
**Fix:** Apply filters in `parseDeathsHTML` after scraping:
```go
if filters.MinLevel > 0 && entry.Victim.Level < filters.MinLevel {
    continue
}
if filters.PvPOnly != nil && *filters.PvPOnly && !entry.IsPvP {
    continue
}
```
Also filter by guild name if provided.

---

### BUG-04: Character `account_information` always empty (HIGH)

**Endpoint:** `GET /v1/character/:name`
**Symptom:** `"account_information": {}` for all characters tested.
**Status:** Still present. Parser likely not matching the HTML labels for account creation date, loyalty title, etc.

---

### BUG-08: News `/latest` missing `id` and `url` fields (HIGH)

**Endpoint:** `GET /v1/news/latest`
**Symptom:** Entry has `date`, `title`, `category`, `type` but no `id` or `url`. Archive entries have both. Newsticker has `id` but no `url`.
**Status:** Still present. The latest endpoint parses the main news page which may not expose article IDs in the same way as the archive page.

---

### BUG-NEW-02: Guild members missing `rank` field (MEDIUM)

**Endpoint:** `GET /v1/guild/:name`
**Symptom:** Some members are missing the `rank` field entirely. Example: "Lady Of King" in guild Reapers has `name`, `vocation`, `level`, `joined`, `status`, `is_online` but no `rank`.
**Root Cause:** Parser may not be extracting rank for all members, possibly due to HTML structure variations when member has no explicit rank or rank column is empty.

---

### BUG-13: News newsticker missing `url` field (MEDIUM)

**Endpoint:** `GET /v1/news/newsticker`
**Symptom:** Entries have `id` but no `url`. Archive entries include `url`.
**Fix:** Construct URL from id (e.g., `"https://rubinot.com.br/?news/archive/{id}"`).

---

### BUG-NEW-03: Guild list `logo_url` uses relative paths (LOW)

**Endpoint:** `GET /v1/guilds/:world`
**Symptom:** `logo_url` is `"./images/guilds/default.gif?v=1745624827"` (relative to site root). API consumers can't use this directly.
**Fix:** Prepend base URL: `"https://rubinot.com.br/images/guilds/..."`.

---

### BUG-NEW-04: House 404s take 30-40 seconds (LOW)

**Endpoint:** `GET /v1/house/:world/:id` with invalid house ID
**Symptom:** Invalid house IDs (1, 35001) return 404 after 30-43 seconds because FlareSolverr fully loads the page before the API can determine the house doesn't exist.
**Fix:** Consider caching known house IDs from the houses list endpoint, or adding a faster validation step before hitting FlareSolverr.

---

## Issue Summary

| Severity | Count | Bugs |
|----------|-------|------|
| Critical | 1 | Deaths filters not applied |
| High | 2 | Character empty account_info, News latest missing id/url |
| Medium | 2 | Guild members missing rank, Newsticker missing url |
| Low | 2 | Guild logo_url relative, House 404 slow |
| **Fixed** | **7** | Guilds empty, Auction status, House paid_until, Events dupes, Auction ID type, Commit SHA, World status casing |

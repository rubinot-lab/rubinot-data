# Highscores V2 Migration: rubinot-data batch fix + rubinot-api consumer switch

## Summary

Migrate rubinot-api's highscores processor from v1 to v2 rubinot-data endpoints. Requires fixing rubinot-data's v2 cross-world highscores handler (currently broken — sequential CDP loop causes broken pipe) and adding a batch fetch function. rubinot-api switches its processor fetch calls from v1 to v2 with a ~20-line adapter.

## Problem

1. **rubinot-api** processor calls v1 cross-world endpoints (`/v1/highscores/all/{category}/{vocation}/all`) which re-paginate upstream data into 50/page, forcing 20 round-trips per world. v2 returns all 1000 entries per call.
2. **rubinot-data** v2 `world=all` highscores handler is broken — loops sequentially through 14 worlds making individual CDP calls, which breaks the WebSocket connection mid-loop (error: `write CDP message: broken pipe`).

## Scope

- **In scope**: Processor fetch path (`processHighscoresCrossWorld`) — the hot path running every 60-600s across 20 categories.
- **Out of scope**: `getHighscores` (paginated API), `getHighscoreCategories`, `postHighscoresCrossWorld`, `postHighscoresMultiCategory`. These can migrate incrementally later.

## Decision: Approach B — rubinot-api adapts to v2 native response shape

- rubinot-data v2 returns `[]HighscoresResult` array (per-world results)
- rubinot-api adds new client method + minimal adapter in processor
- No v1 compatibility wrapper in v2 — clean API
- Old v1 client method kept for rollback

Alternatives considered:
- A) Match v1 response shape in v2 — rejected (couples v2 to v1 baggage)
- C) rubinot-api calls 70 per-world v2 endpoints in parallel — rejected (wasteful, defeats CDP pool)

## Changes

### rubinot-data (this repo)

#### 1. `V2FetchHighscoresBatch` in `internal/scraper/v2_fetch.go`

New function following `V2FetchKillstatisticsBatch` pattern:
- Input: `worlds []validation.World`, `category HighscoreCategory`, `vocation HighscoreVocation`
- Builds 14 URLs: `{baseURL}/api/highscores?world={id}&category={slug}&vocation={professionID}`
- Calls `oc.BatchFetchJSON(ctx, urls)` — single CDP `Promise.allSettled()` for all 14 worlds
- Parses each response, maps to `[]domain.HighscoresResult`
- Returns `([]domain.HighscoresResult, []string, error)`

#### 2. Fix `v2GetHighscores` handler in `internal/api/handlers_v2.go`

Replace sequential loop (lines 145-161) with call to `V2FetchHighscoresBatch`. Fixes broken pipe on `world=all`.

### rubinot-api (rubinot-lab/rubinot-api)

#### 3. New client method in `src/services/rubinot-data-client.ts`

```typescript
async getHighscoresCrossWorldV2(
  category: string,
  vocation: HighscoreVocation,
): Promise<HighscoresPayload[]> {
  return this.fetchPayload<HighscoresPayload[]>(
    `/v2/highscores/all/${encodeURIComponent(category)}/${encodeURIComponent(vocation)}`,
    { endpoint: "highscores_cross_world_v2", timeoutMs: 30000, attempts: 3, retryDelayMs: 3000 },
  );
}
```

#### 4. Processor adapter in `src/jobs/processors/highscores.processor.ts`

Current (v1):
```typescript
const payload = await client.getHighscoresCrossWorldByVocation(category, vocation);
for (const worldData of payload.worlds) { ... }
```

New (v2):
```typescript
const worlds = await client.getHighscoresCrossWorldV2(category, vocation);
for (const worldData of worlds) { ... }
```

Inner loop body unchanged — `worldData` has same `world`, `highscore_list`, `highscore_page` fields.

#### 5. Feature flag

Env var `HIGHSCORE_USE_V2` (boolean, default false). Processor checks flag to choose v1 or v2 path. Remove once stable.

## Response Shape Comparison

### v1 (`/v1/highscores/all/{category}/{vocation}/all`)
```json
{
  "highscores": {
    "world": "unknown",
    "vocation": "knights",
    "total_worlds": 14,
    "worlds": [
      { "world": "Elysian", "highscore_list": [...], "highscore_page": {...} }
    ]
  }
}
```

### v2 (`/v2/highscores/all/{category}/{vocation}`)
```json
{
  "highscores": [
    { "world": "Elysian", "highscore_list": [...], "highscore_page": {...} },
    { "world": "Belaria", "highscore_list": [...], "highscore_page": {...} }
  ]
}
```

Difference: v2 returns flat array under `highscores` key. v1 nests under `highscores.worlds`. Per-world objects have identical structure.

## Testing

### rubinot-data
- Unit test for `V2FetchHighscoresBatch`: mock CDP responses, verify URL construction and response mapping
- Integration: `GET /v2/highscores/all/experience/knights` returns 14 worlds × 1000 entries, no broken pipe

### rubinot-api
- Unit test: processor with `HIGHSCORE_USE_V2=true` produces identical `byWorld` map as v1 path
- Smoke test: run one highscore-cycle, compare `highscorePhaseSeconds` fetch phase — expect drop from ~12s to ~2-3s

## Deployment Order

1. Deploy rubinot-data with `V2FetchHighscoresBatch` + handler fix (tag + push)
2. Verify `GET /v2/highscores/all/experience/knights` returns 200 with all 14 worlds
3. Deploy rubinot-api with `HIGHSCORE_USE_V2=false` (flag off)
4. Flip `HIGHSCORE_USE_V2=true`, monitor one cycle
5. If stable for 24h, remove flag and old v1 method

## Expected Impact

| Metric | v1 (current) | v2 (expected) |
|--------|-------------|---------------|
| Calls per vocation | 1 (cross-world, but 20 upstream pages internally) | 1 (cross-world, 14 batch CDP fetches) |
| Fetch phase per category | ~12s | ~2-3s |
| Total calls per category | 5 (one per vocation) | 5 (one per vocation) |
| Response entries per call | 14 worlds × 1000 | 14 worlds × 1000 |
| CDP connection stability | N/A (v1 uses FlareSolverr) | Batch fetch — single CDP call, no broken pipe |

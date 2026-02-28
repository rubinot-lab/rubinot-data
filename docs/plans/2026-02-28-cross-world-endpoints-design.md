# Cross-World Endpoints & Boosted Image URLs

## Summary

Add cross-world `/all` support to killstatistics, deaths, banishments, and guilds endpoints. Add computed `image_url` to the boosted response. Reduces rubinot-api's upstream HTTP requests by ~60% (from ~31 req/min to ~5 req/min).

## Architecture

```
GET /v1/killstatistics/all
  → handler: isAllWorldsToken("all") = true
  → validator.AllWorlds() → 14 worlds
  → build 14 upstream API URLs
  → CDP BatchFetch (batches of 6, Promise.allSettled)
  → parse → []KillstatisticsResult

GET /v1/deaths/all/all, /v1/banishments/all/all, /v1/guilds/all/all
  → handler: isAllWorldsToken("all") = true
  → sequential iteration over AllWorlds()
  → call existing FetchAll* per world (handles pagination)
  → aggregate → []XResult
```

## Decisions

- **Cross-world fetch strategy**: CDP BatchFetch for killstatistics (1 URL/world, no pagination). Sequential iteration with existing FetchAll* for deaths/banishments/guilds (paginated, variable request count).
- **Response format**: Array of per-world results (consistent with getWorldDetails "all" pattern).
- **Error handling**: Fail entire request if any world fails (no partial results). Same as existing world/all/details.
- **Boosted image_url**: Computed from looktype, points to rubinot-data's own /v1/outfit endpoint. No extra scraping.

## Benchmark (before)

```
Endpoint             Sequential(14)  Parallel(14)  Payload
killstatistics       6-131s          126s          2.5 MB
deaths/all           5-134s          126s          786 KB
banishments/all      18-145s         125s          3.3 MB
guilds/all           6.4s            125s          974 KB
```

Parallel is bottlenecked by FlareSolverr session contention (~125s outlier).
CDP BatchFetch avoids this by reusing the global browser connection.

## Components

### New scraper functions

| Function | File | Strategy |
|---|---|---|
| FetchAllWorldsKillstatistics | killstatistics.go | CDP BatchFetch (14 URLs in 3 batches) |
| (no new function for deaths) | deaths.go | Reuse existing FetchAllDeaths per world |
| (no new function for banishments) | banishments.go | Reuse existing FetchAllBanishments per world |
| (no new function for guilds) | guilds.go | Reuse existing FetchAllGuilds per world |

### Handler modifications (router.go)

| Handler | Change |
|---|---|
| getKillstatistics | Add isAllWorldsToken branch → FetchAllWorldsKillstatistics |
| getAllDeaths | Add isAllWorldsToken branch → loop AllWorlds() calling FetchAllDeaths |
| getAllBanishments | Add isAllWorldsToken branch → loop AllWorlds() calling FetchAllBanishments |
| getAllGuilds | Add isAllWorldsToken branch → loop AllWorlds() calling FetchAllGuilds |
| getAllGuildsDetails | Add isAllWorldsToken branch → loop AllWorlds() calling existing detail fetcher |
| getBoosted | Compute image_url from looktype after fetch |

### Domain changes

| Type | File | Change |
|---|---|---|
| BoostedEntity | upstream.go | Add ImageURL string field |

## Testing

- Unit: TestFetchAllWorldsKillstatistics (mock CDP batch for 3 worlds)
- Unit: TestBoostedImageURL (verify URL computed from looktype)
- Integration: benchmark-cross-world.sh Phase 3 (already written)
- Edge: empty world results, CDP failure mid-batch, isAllWorldsToken case variations

## Commit Plan

### Phase 1: Killstatistics (CDP batch)
1. `feat(scraper): add FetchAllWorldsKillstatistics via CDP batch` — killstatistics.go + test
2. `feat(api): support "all" token on killstatistics endpoint` — router.go

### Phase 2: Deaths (sequential loop)
3. `feat(api): support "all" token on deaths/all endpoint` — router.go + test

### Phase 3: Banishments (sequential loop)
4. `feat(api): support "all" token on banishments/all endpoint` — router.go + test

### Phase 4: Guilds (sequential loop)
5. `feat(api): support "all" token on guilds/all and guilds/all/details` — router.go + test

### Phase 5: Boosted image URL
6. `feat(api): add image_url to boosted response` — upstream.go + router.go + test

### Phase 6: Post-deploy
7. Re-run benchmark, document results
8. PR self-review, apply recommendations, remove unnecessary comments

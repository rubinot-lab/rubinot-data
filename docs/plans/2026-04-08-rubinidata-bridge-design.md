# rubinidata.com Bridge — Design Doc

## Problem
Workers slam rubinot-data with burst requests, Cloudflare blocks within 15s. rubinot.com.br upstream unreliable.

## Solution
Temporary bridge: use api.rubinidata.com as upstream instead of rubinot.com.br. Plain HTTP + API key, no Cloudflare bypass needed.

## Scope
Throwaway branch — deploy from it, no merge to main. Tier 1+2 worker jobs.

## Architecture

### Flow Change
```
Before: Handler -> OptimizedClient -> CachedFetcher -> CDPPool -> CDP -> rubinot.com.br/api/*
After:  Handler -> OptimizedClient -> CachedFetcher -> RubinidataClient -> HTTP -> api.rubinidata.com/v1/*
```

### New Files
- `internal/scraper/rubinidata_client.go` — HTTP client for api.rubinidata.com
- `internal/scraper/rubinidata_adapters.go` — response format converters (rubinidata JSON -> rubinot.com.br JSON shape)

### Modified Files
- `internal/scraper/cached_fetcher.go` — route to rubinidata or CDP based on env
- `internal/api/router.go` — skip CDP/FlareSolverr init when provider=rubinidata; readiness probe always ready

### Env Vars
- `UPSTREAM_PROVIDER` — `rubinot` (default) | `rubinidata`
- `RUBINIDATA_URL` — default `https://api.rubinidata.com`
- `RUBINIDATA_API_KEY` — X-API-Key value

## Endpoint Mapping

| rubinot.com.br path | rubinidata.com path | Adapter needed |
|---|---|---|
| `/api/worlds` | `/v1/worlds` | Yes — `overview.total_players_online` -> `total_players_online`, `regular_worlds[]` -> `worlds[]` |
| `/api/worlds/{name}` | `/v1/world/{name}` | Yes — flatten `online_record`, rename fields |
| `/api/characters/search?name=X` | `/v1/characters/{name}` | Yes — restructure character/deaths/other_characters |
| `/api/guilds?world=X&page=Y` | `/v1/guilds/{world}?page=Y` | Yes — field renames |
| `/api/guilds/{name}` | `/v1/guild/{name}` | Yes — member structure differs |
| `/api/highscores?world=X&category=Y&vocation=Z` | `/v1/highscores?world=X&category=Y&vocation=Z` | Yes — field renames, vocation uses int IDs |
| `/api/killstats?world=X` | `/v1/killstatistics/{world}` | Yes — field renames |
| `/api/deaths?world=X&page=Y` | `/v1/deaths/{world}?page=Y` | Yes — restructure entry format |
| `/api/bans?world=X&page=Y` | `/v1/banishments/{world}?page=Y` | Yes — field renames |
| `/api/transfers?page=X` | `/v1/transfers?page=X&from=X&to=X` | Yes — field renames |
| `/api/boosted` | `/v1/boosted` | Yes — `creature`/`boss` structure differs |
| `/api/outfit?...` | `/v1/outfit?...` | No — binary proxy, params compatible |

## Batch Endpoint Synthesis

Batch endpoints don't exist on rubinidata.com. Synthesize from individual calls:
- `characters/batch` — fan out N calls to `/v1/characters/{name}`, adapt each, collect
- `guilds/batch` — fan out N calls to `/v1/guild/{name}`, adapt each, collect
- `killstatistics/batch` — fan out N calls to `/v1/killstatistics/{world}`, adapt each, collect

Concurrency: semaphore of 10 concurrent outbound requests.

## Missing Data

Fields rubinidata.com doesn't return (set to zero/empty):
- Character: `found_by_old_name`, `vip_time`, `house`, `houses`, `deaths`, `account_badges`, `displayed_achievements`, `former_names`, `former_worlds`, `account_created`, `guild` embedded, full `outfit` object
- All IDs are 0 (rubinidata doesn't assign IDs)
- Highscores: only 10 of 20 categories (missing dromelevel, linked_tasks, exp_today, battlepass, charmunlockpoints, prestigepoints, totalweeklytasks, totalbountypoints, charmtotalpoints, bosstotalpoints)

## Disabled Workers (no rubinidata equivalent)
- `name-change-detection` — needs `found_by_old_name`
- News jobs — endpoint doesn't exist
- Events jobs — endpoint doesn't exist
- Auction jobs — endpoint doesn't exist
- Maintenance job — endpoint doesn't exist

## Error Handling
- Retry up to 3 times with backoff on rubinidata.com failures
- No fallback to CDP — rubinot.com.br unreliable
- Unsupported paths return error to caller
- Readiness probe always returns ready when provider=rubinidata

## Deployment
- Separate branch (no merge to main)
- Tag and deploy via existing pipeline
- Scale down disabled workers in platform-gitops

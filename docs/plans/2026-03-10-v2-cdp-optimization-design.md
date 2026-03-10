# v2 API: CDP Connection Pool, Singleflight Cache, and Optimized Scraping

## Summary

Introduce a `/v2` API group that maximizes throughput through CDP connection pooling (4 tabs), request deduplication (singleflight), short-TTL in-memory caching (5s), and parallel fan-out fetching. v1 remains frozen and untouched. Deployment drops from 30 replicas to 5.

## Problem Statement

### Current metrics (observed 2026-03-10)

| Metric | Value |
|---|---|
| Replicas | 30 (hardcoded, no HPA) |
| In-flight requests | 1700-4023 (growing unbounded) |
| Scrape p99 latency | 21s (maxed out) |
| CDP fetch p99 | 5s (timing out) |
| Upstream success rate | collapsed from 100-150 req/s to ~0.5 req/s |
| HTTP p50 latency | 3.3ms (stable — cache/fast path fine) |
| HTTP p99 latency | 5s (scraper tail) |
| Cloudflare challenges | 0 (not a CF issue) |
| Pod resource usage | ~530m CPU + ~620Mi memory each |
| Total cluster cost | 30 × 620Mi = ~18Gi memory, ~16 cores |

### Root cause

30 pods × `SCRAPE_MAX_CONCURRENCY=8` = 240 potential concurrent scrapers hitting rubinot.com.br. The upstream throttles/times out under this load. Failed requests hold connections for up to 120s (`SCRAPE_MAX_TIMEOUT_MS`), causing cascading queue buildup.

### Additional architectural issues

1. **CDP is single-threaded per pod** — one global WebSocket, mutex-serialized `Evaluate()` calls
2. **No response caching** — every inbound request triggers an upstream scrape
3. **Semaphore doesn't guard CDP path** — `acquireSemaphore()` only wraps FlareSolverr `Fetch()`, not `fetchViaCDP()`/`FetchAllPages()`
4. **No `disableMedia`** sent to FlareSolverr — Chrome loads images/CSS/fonts during session init
5. **Fan-out endpoints are sequential** — "all worlds" loops one world at a time
6. **`api` container has `resources: {}`** — no CPU/memory requests or limits
7. **Client-side pagination in highscores** — upstream returns all 1000 entries, rubinot-data slices into 50/page across 20 pages

## Decisions Log

| # | Decision | Rationale | Alternatives considered |
|---|---|---|---|
| D1 | v2 API group, v1 frozen | Don't break existing consumers. v1 can be decommissioned later. | Modify v1 in-place (risk breaking rubinot-api) |
| D2 | Singleflight as core dedup | Zero staleness risk. Deduplicates concurrent identical upstream calls. | Redis pub/sub dedup (adds infra), no dedup |
| D3 | 5s TTL in-memory cache on top | Bounds staleness acceptably. Handles rapid-fire repeated requests. | 1-2s TTL (too aggressive churn), 30s (stale data risk), Redis (infra overhead) |
| D4 | 4 CDP tabs per pod via `Target.createTarget` | True parallelism for concurrent independent requests. | Single WebSocket with batch coalescing (adds latency, head-of-line blocking), 14 tabs (excessive memory for non-fan-out traffic) |
| D5 | One FlareSolverr session, tabs share cookies | Cookies are browser-level. One session init, 3 extra tabs via CDP. | 4 separate FlareSolverr sessions (redundant cookie setup, more overhead) |
| D6 | `disableMedia: true` on FlareSolverr | Skip images/CSS/fonts during session init. Less memory, faster startup. | Leave as-is (wasteful) |
| D7 | BatchFetch size bumped from 6 to 20 | All 14 worlds fit in one `Promise.allSettled()` call. | Keep at 6 (3 batches for 14 worlds), 14 (too exact) |
| D8 | 5 replicas (down from 30), tune from metrics | 83% reduction. 5 × 4 tabs = 20 CDP channels. Singleflight + cache covers the rest. | 3 replicas (less headroom), keep 30 (wasteful) |
| D9 | v2 highscores: drop client-side pagination | Upstream returns all ~1000 entries. No reason to re-paginate. Consumer can slice. | Keep pagination (unnecessary complexity in v2) |
| D10 | v2 fan-outs: parallel BatchFetch | Fetch all worlds/pages in one CDP `Promise.allSettled()` instead of sequential loops. | Keep sequential (slow), goroutine pool per world (complex, unnecessary with BatchFetch) |
| D11 | Cache at scraper level, keyed by upstream URL | Fan-out endpoints populate cache for individual requests. `/v2/world/all` caches each world individually. | Cache at handler level (misses cross-endpoint dedup) |
| D12 | Errors NOT cached | Failed fetches should be retried on next request. Singleflight shares errors to concurrent waiters only. | Cache errors with short TTL (masks transient failures) |
| D13 | Keep 30s per-CDP-call timeout (existing fallback) | Already working. Each `Evaluate()` has 30s deadline. Fan-out total time = batches × 30s. | Reduce to 15s (may kill legitimate slow upstream), increase to 60s (holds resources too long) |
| D14 | `session_ttl_minutes: 30` on FlareSolverr | Auto-rotate stale sessions instead of holding forever. | No TTL (session could go stale), 5 min (too aggressive rotation) |
| D15 | Server-side paginated endpoints (deaths, transfers, banishments, guilds, auctions): fetch page 1, then BatchFetch remaining pages in parallel | Upstream forces 50/page. Parallel page fetch is the only optimization available. | Sequential (current, slow) |

## Architecture

### Component diagram

```
┌─────────────────────────────────────────────────────────┐
│ rubinot-data pod (×5)                                   │
│                                                         │
│  ┌──────────────────┐     ┌───────────────────────────┐ │
│  │ Go API (gin)     │     │ FlareSolverr sidecar      │ │
│  │                  │     │ - disableMedia: true       │ │
│  │ /v1/* ──→ scraper.Client (unchanged)               │ │
│  │                  │     │ - 1 session: rubinot-cdp   │ │
│  │ /v2/* ──→ scraper.OptimizedClient                  │ │
│  │           │      │     │ - session_ttl_minutes: 30  │ │
│  │           │      │     └──────┬────────────────────┘ │
│  │           ▼      │            │                      │
│  │  ┌──────────────────────┐     │                      │
│  │  │ CachedFetcher        │     │ CDP WebSocket (×4)   │
│  │  │ ┌──────────────────┐ │     │                      │
│  │  │ │ sync.Map cache   │ │     │ Tab 0 (session page) │
│  │  │ │ 5s TTL per key   │ │     │ Tab 1 (createTarget) │
│  │  │ ├──────────────────┤ │ ──→ │ Tab 2 (createTarget) │
│  │  │ │ singleflight.Grp │ │     │ Tab 3 (createTarget) │
│  │  │ ├──────────────────┤ │     │                      │
│  │  │ │ CDPPool (4 tabs) │ │     │ Each tab: own WS,    │
│  │  │ └──────────────────┘ │     │ own mutex, own Eval() │
│  │  └──────────────────────┘     └───────────────────────┘
│  └──────────────────┘                                   │
└─────────────────────────────────────────────────────────┘
```

### Request flow: single resource

```
GET /v2/world/Antica
  │
  ├─ v2 handler: validate input via existing validator
  │
  ├─ OptimizedClient.FetchJSON(ctx, "https://rubinot.com.br/api/worlds/Antica")
  │    │
  │    ├─ cacheKey = "/api/worlds/Antica"
  │    │
  │    ├─ cache.Load(cacheKey)
  │    │    hit + not expired? → return cached body (< 1ms)
  │    │
  │    ├─ singleflight.Do(cacheKey, func() {
  │    │       tab, idx := pool.Acquire(ctx)   // blocks until tab free or ctx cancelled
  │    │       defer pool.Release(idx)
  │    │       body := tab.Fetch(ctx, cacheKey) // CDP Runtime.evaluate → fetch()
  │    │       cache.Store(cacheKey, body, 5s)
  │    │       return body
  │    │   })
  │    │
  │    │   // concurrent callers for same URL share this single flight
  │    │
  │    └─ return body
  │
  ├─ parseJSONBody(body, &payload) → domain model (reuse existing parsers)
  │
  └─ return envelope { data: { world: ... }, sources: [...] }
```

### Request flow: fan-out (all worlds)

```
GET /v2/world/all
  │
  ├─ v2 handler: validator.AllWorlds() → 14 world names
  │
  ├─ OptimizedClient.BatchFetchJSON(ctx, [14 upstream URLs])
  │    │
  │    ├─ for each URL: cache.Load(cacheKey)
  │    │    collect hits (instant) + pending misses
  │    │
  │    ├─ if all cached → return immediately
  │    │
  │    ├─ tab, idx := pool.Acquire(ctx)
  │    │   defer pool.Release(idx)
  │    │
  │    ├─ tab.BatchFetch(ctx, pendingPaths)
  │    │   // one CDP Evaluate() call:
  │    │   // Promise.allSettled([fetch('/api/worlds/Antica'), fetch('/api/worlds/Belaria'), ...])
  │    │   // all 14 fire in parallel inside Chrome
  │    │   // returns when all complete (~2-4s total)
  │    │
  │    ├─ for each result: cache.Store(path, body, 5s)
  │    │
  │    └─ return map[url]body
  │
  ├─ parse each body → []domain.WorldResult
  │
  └─ return envelope { data: { worlds: [...] }, sources: [...] }
```

### Request flow: server-side paginated fan-out (all deaths)

```
GET /v2/deaths/Auroria/all
  │
  ├─ v2 handler: validate world
  │
  ├─ Step 1: OptimizedClient.FetchJSON(ctx, "/api/deaths?world=1&page=1")
  │    │    (goes through singleflight + cache as normal)
  │    └─ parse pagination: { totalPages: 6 }
  │
  ├─ Step 2: build URLs for pages 2-6
  │    ["/api/deaths?world=1&page=2", ..., "/api/deaths?world=1&page=6"]
  │
  ├─ Step 3: OptimizedClient.BatchFetchJSON(ctx, [5 remaining URLs])
  │    │    cache check per URL → BatchFetch uncached
  │    └─ returns map[url]body
  │
  ├─ merge page 1 + pages 2-6 → aggregate all deaths
  │
  └─ return envelope { data: { deaths: {...} }, sources: [...] }
```

## New Components

### 1. CDPPool (`internal/scraper/cdp_pool.go`)

Manages a pool of N CDP tab connections, providing acquire/release semantics.

```go
package scraper

import (
    "context"
    "fmt"
    "sync"
)

type CDPPool struct {
    mu        sync.Mutex
    tabs      []*CDPClient
    available chan int // indexes of free tabs
    baseURL   string
    size      int
}

func NewCDPPool(baseURL string, size int) *CDPPool {
    return &CDPPool{
        baseURL:   baseURL,
        size:      size,
        available: make(chan int, size),
    }
}

func (p *CDPPool) Init(ctx context.Context) error {
    // 1. Discover the default page target (tab 0) via /json/list
    defaultWSURL, err := discoverPageTarget(ctx, p.baseURL)
    if err != nil {
        return fmt.Errorf("discover default page target: %w", err)
    }

    p.tabs = make([]*CDPClient, p.size)

    // 2. Connect to tab 0 (the FlareSolverr session page)
    tab0 := NewCDPClient(p.baseURL)
    if err := tab0.ConnectToURL(ctx, defaultWSURL); err != nil {
        return fmt.Errorf("connect tab 0: %w", err)
    }
    p.tabs[0] = tab0
    p.available <- 0

    // 3. Create tabs 1..N-1 via Target.createTarget on tab 0
    for i := 1; i < p.size; i++ {
        targetID, err := tab0.CreateTarget(ctx, "about:blank")
        if err != nil {
            return fmt.Errorf("create tab %d: %w", i, err)
        }

        wsURL := fmt.Sprintf("ws://%s/devtools/page/%s",
            extractHost(p.baseURL), targetID)
        tab := NewCDPClient(p.baseURL)
        if err := tab.ConnectToURL(ctx, wsURL); err != nil {
            return fmt.Errorf("connect tab %d: %w", i, err)
        }

        // Navigate to base URL so fetch() calls are same-origin
        if err := tab.Navigate(ctx, baseURL()); err != nil {
            return fmt.Errorf("navigate tab %d: %w", i, err)
        }

        p.tabs[i] = tab
        p.available <- i
    }

    return nil
}

func (p *CDPPool) Acquire(ctx context.Context) (*CDPClient, int, error) {
    select {
    case idx := <-p.available:
        p.mu.Lock()
        tab := p.tabs[idx]
        p.mu.Unlock()

        if tab == nil || !tab.IsConnected() {
            rebuilt, err := p.rebuildTab(ctx, idx)
            if err != nil {
                p.available <- idx
                return nil, 0, fmt.Errorf("rebuild tab %d: %w", idx, err)
            }
            return rebuilt, idx, nil
        }
        return tab, idx, nil
    case <-ctx.Done():
        return nil, 0, ctx.Err()
    }
}

func (p *CDPPool) Release(idx int) {
    p.available <- idx
}

func (p *CDPPool) rebuildTab(ctx context.Context, idx int) (*CDPClient, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.tabs[idx] != nil {
        p.tabs[idx].Close()
    }

    // Use any connected tab to create a new target
    var creator *CDPClient
    for _, t := range p.tabs {
        if t != nil && t.IsConnected() {
            creator = t
            break
        }
    }
    if creator == nil {
        return nil, fmt.Errorf("no healthy tabs available to create target")
    }

    targetID, err := creator.CreateTarget(ctx, "about:blank")
    if err != nil {
        return nil, fmt.Errorf("create replacement target: %w", err)
    }

    wsURL := fmt.Sprintf("ws://%s/devtools/page/%s",
        extractHost(p.baseURL), targetID)
    tab := NewCDPClient(p.baseURL)
    if err := tab.ConnectToURL(ctx, wsURL); err != nil {
        return nil, err
    }
    if err := tab.Navigate(ctx, baseURL()); err != nil {
        tab.Close()
        return nil, err
    }

    p.tabs[idx] = tab
    return tab, nil
}

func (p *CDPPool) Close() {
    p.mu.Lock()
    defer p.mu.Unlock()
    for _, tab := range p.tabs {
        if tab != nil {
            tab.Close()
        }
    }
}
```

**New CDPClient methods required:**

```go
// ConnectToURL connects directly to a known WebSocket URL (skip discovery)
func (c *CDPClient) ConnectToURL(ctx context.Context, wsURL string) error

// CreateTarget opens a new browser tab, returns targetId
func (c *CDPClient) CreateTarget(ctx context.Context, url string) (string, error)

// Navigate uses Page.navigate to load a URL in the tab
func (c *CDPClient) Navigate(ctx context.Context, url string) error
```

### 2. CachedFetcher (`internal/scraper/cached_fetcher.go`)

Wraps CDPPool with singleflight deduplication and TTL cache.

```go
package scraper

import (
    "context"
    "sync"
    "time"

    "golang.org/x/sync/singleflight"
)

type cacheEntry struct {
    value     string
    expiresAt time.Time
}

type CachedFetcher struct {
    pool  *CDPPool
    group singleflight.Group
    cache sync.Map // map[string]*cacheEntry
    ttl   time.Duration
}

func NewCachedFetcher(pool *CDPPool, ttl time.Duration) *CachedFetcher {
    return &CachedFetcher{pool: pool, ttl: ttl}
}

func (f *CachedFetcher) FetchJSON(ctx context.Context, apiURL string) (string, error) {
    cacheKey, err := apiPathFromURL(apiURL)
    if err != nil {
        return "", err
    }

    // 1. Cache check
    if entry, ok := f.cache.Load(cacheKey); ok {
        ce := entry.(*cacheEntry)
        if time.Now().Before(ce.expiresAt) {
            CacheRequests.WithLabelValues("hit").Inc()
            return ce.value, nil
        }
        f.cache.Delete(cacheKey)
    }
    CacheRequests.WithLabelValues("miss").Inc()

    // 2. Singleflight dedup — concurrent callers for same URL share one fetch
    result, err, _ := f.group.Do(cacheKey, func() (interface{}, error) {
        tab, idx, acquireErr := f.pool.Acquire(ctx)
        if acquireErr != nil {
            return nil, acquireErr
        }
        defer f.pool.Release(idx)

        started := time.Now()
        body, fetchErr := tab.Fetch(ctx, cacheKey)
        CDPFetchDuration.Observe(time.Since(started).Seconds())

        if fetchErr != nil {
            CDPFetchRequests.WithLabelValues("error").Inc()
            return nil, fetchErr
        }

        CDPFetchRequests.WithLabelValues("ok").Inc()

        // 3. Store in cache
        f.cache.Store(cacheKey, &cacheEntry{
            value:     body,
            expiresAt: time.Now().Add(f.ttl),
        })
        return body, nil
    })

    if err != nil {
        return "", err
    }
    return result.(string), nil
}

func (f *CachedFetcher) BatchFetchJSON(
    ctx context.Context,
    apiURLs []string,
) (map[string]string, error) {
    results := make(map[string]string, len(apiURLs))
    pending := make([]string, 0)
    pendingKeys := make([]string, 0)

    // 1. Collect cache hits
    for _, apiURL := range apiURLs {
        key, keyErr := apiPathFromURL(apiURL)
        if keyErr != nil {
            return nil, keyErr
        }

        if entry, ok := f.cache.Load(key); ok {
            ce := entry.(*cacheEntry)
            if time.Now().Before(ce.expiresAt) {
                CacheRequests.WithLabelValues("hit").Inc()
                results[apiURL] = ce.value
                continue
            }
            f.cache.Delete(key)
        }
        CacheRequests.WithLabelValues("miss").Inc()
        pending = append(pending, apiURL)
        pendingKeys = append(pendingKeys, key)
    }

    if len(pending) == 0 {
        return results, nil
    }

    // 2. Acquire one tab, BatchFetch all uncached URLs
    tab, idx, err := f.pool.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    defer f.pool.Release(idx)

    started := time.Now()
    batchResults, err := tab.BatchFetch(ctx, pendingKeys)
    CDPFetchDuration.Observe(time.Since(started).Seconds())

    if err != nil {
        CDPFetchRequests.WithLabelValues("error").Add(float64(len(pending)))
        return nil, err
    }

    // 3. Process results, cache individually
    for i, br := range batchResults {
        if br.Status == "fulfilled" {
            trimmed := strings.TrimSpace(br.Value)
            if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
                CDPFetchRequests.WithLabelValues("ok").Inc()
                UpstreamStatus.WithLabelValues(endpointFromURL(pending[i]), "200").Inc()
                f.cache.Store(pendingKeys[i], &cacheEntry{
                    value:     br.Value,
                    expiresAt: time.Now().Add(f.ttl),
                })
                results[pending[i]] = br.Value
            } else {
                CDPFetchRequests.WithLabelValues("non_json").Inc()
            }
        } else {
            CDPFetchRequests.WithLabelValues("error").Inc()
        }
    }

    return results, nil
}
```

### 3. OptimizedClient (`internal/scraper/optimized_client.go`)

Thin wrapper that handlers interact with. Exposes the same method signatures as existing scraper functions.

```go
package scraper

import (
    "context"
    "encoding/json"
)

type OptimizedClient struct {
    fetcher *CachedFetcher
}

func NewOptimizedClient(fetcher *CachedFetcher) *OptimizedClient {
    return &OptimizedClient{fetcher: fetcher}
}

func (c *OptimizedClient) FetchJSON(ctx context.Context, apiURL string, result any) error {
    body, err := c.fetcher.FetchJSON(ctx, apiURL)
    if err != nil {
        return err
    }
    return parseJSONBody(body, result)
}

func (c *OptimizedClient) BatchFetchJSON(ctx context.Context, apiURLs []string) (map[string]string, error) {
    return c.fetcher.BatchFetchJSON(ctx, apiURLs)
}
```

### 4. v2 Router (`internal/api/router_v2.go`)

Registers `/v2` routes wired to the optimized client. Separate file to keep v1 router untouched.

```go
package api

func registerV2Routes(router *gin.Engine, oc *scraper.OptimizedClient) {
    v2 := router.Group("/v2")
    {
        v2.GET("/worlds", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetWorlds(c, oc)
        }))
        v2.GET("/world/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetWorld(c, getValidator(), oc)
        }))
        v2.GET("/world/:name/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetWorldDetails(c, getValidator(), oc)
        }))
        v2.GET("/world/:name/dashboard", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetWorldDashboard(c, getValidator(), oc)
        }))
        v2.GET("/highscores/:world/:category/:vocation", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetHighscores(c, getValidator(), oc)
        }))
        v2.GET("/killstatistics/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetKillstatistics(c, getValidator(), oc)
        }))
        v2.GET("/deaths/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetDeaths(c, getValidator(), oc)
        }))
        v2.GET("/deaths/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAllDeaths(c, getValidator(), oc)
        }))
        v2.GET("/banishments/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetBanishments(c, getValidator(), oc)
        }))
        v2.GET("/banishments/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAllBanishments(c, getValidator(), oc)
        }))
        v2.GET("/transfers", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetTransfers(c, oc)
        }))
        v2.GET("/transfers/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAllTransfers(c, getValidator(), oc)
        }))
        v2.GET("/character/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetCharacter(c, oc)
        }))
        v2.GET("/guild/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetGuild(c, oc)
        }))
        v2.GET("/guilds/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetGuilds(c, getValidator(), oc)
        }))
        v2.GET("/guilds/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAllGuilds(c, getValidator(), oc)
        }))
        v2.GET("/boosted", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetBoosted(c, oc)
        }))
        v2.GET("/maintenance", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetMaintenance(c, oc)
        }))
        v2.GET("/auctions/current/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetCurrentAuctions(c, oc)
        }))
        v2.GET("/auctions/current/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAllCurrentAuctions(c, oc)
        }))
        v2.GET("/auctions/history/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAuctionHistory(c, oc)
        }))
        v2.GET("/auctions/history/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAllAuctionHistory(c, oc)
        }))
        v2.GET("/auctions/:id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetAuctionDetail(c, oc)
        }))
        v2.GET("/news/id/:news_id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetNewsByID(c, oc)
        }))
        v2.GET("/news/archive", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetNewsArchive(c, oc)
        }))
        v2.GET("/news/latest", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
            return v2GetNewsLatest(c, oc)
        }))
        v2.GET("/outfit", getOutfit)          // reuse v1 — no scraping
        v2.GET("/outfit/:name", getOutfitByCharacterName) // reuse v1
    }
}
```

### 5. v2 Handlers (`internal/api/handlers_v2.go`)

Handler implementations. Three patterns: thin wrapper, fan-out, paginated fan-out.

**Pattern A: Thin wrapper (single resource)**

```go
func v2GetWorld(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
    worldInput := strings.TrimSpace(c.Param("name"))

    if isAllWorldsToken(worldInput) {
        return v2GetAllWorlds(c, validator, oc)
    }

    canonicalWorld, _, ok := validator.WorldExists(worldInput)
    if !ok {
        return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
    }

    sourceURL := fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(resolvedBaseURL, "/"), url.PathEscape(canonicalWorld))
    var payload scraper.WorldAPIResponse
    if err := oc.FetchJSON(c.Request.Context(), sourceURL, &payload); err != nil {
        return endpointResult{Sources: []string{sourceURL}}, err
    }

    return endpointResult{
        PayloadKey: "world",
        Payload:    scraper.MapWorldResponse(payload, canonicalWorld),
        Sources:    []string{sourceURL},
    }, nil
}
```

**Pattern B: Fan-out via BatchFetch (all worlds, no upstream pagination)**

```go
func v2GetAllWorlds(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
    worlds := validator.AllWorlds()
    urls := make([]string, len(worlds))
    for i, w := range worlds {
        urls[i] = fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(resolvedBaseURL, "/"), url.PathEscape(w.Name))
    }

    bodyMap, err := oc.BatchFetchJSON(c.Request.Context(), urls)
    if err != nil {
        return endpointResult{Sources: urls}, err
    }

    results := make([]domain.WorldResult, 0, len(worlds))
    for i, u := range urls {
        body, ok := bodyMap[u]
        if !ok {
            return endpointResult{Sources: urls}, fmt.Errorf("missing result for %s", u)
        }
        var payload scraper.WorldAPIResponse
        if err := json.Unmarshal([]byte(body), &payload); err != nil {
            return endpointResult{Sources: urls}, err
        }
        results = append(results, scraper.MapWorldResponse(payload, worlds[i].Name))
    }

    return endpointResult{
        PayloadKey: "worlds",
        Payload:    results,
        Sources:    urls,
    }, nil
}
```

**Pattern C: Paginated fan-out (deaths/all — upstream paginates at 50/page)**

```go
func v2GetAllDeaths(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
    worldInput := strings.TrimSpace(c.Param("world"))
    canonicalWorld, worldID, ok := validator.WorldExists(worldInput)
    if !ok {
        return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
    }

    // Step 1: Fetch page 1 to discover totalPages
    page1URL := fmt.Sprintf("%s/api/deaths?world=%d&page=1", strings.TrimRight(resolvedBaseURL, "/"), worldID)
    page1Body, err := oc.fetcher.FetchJSON(c.Request.Context(), page1URL)
    if err != nil {
        return endpointResult{Sources: []string{page1URL}}, err
    }

    totalPages, err := scraper.DeathsTotalPagesFromBody(page1Body)
    if err != nil {
        return endpointResult{Sources: []string{page1URL}}, err
    }

    // Step 2: BatchFetch remaining pages
    allBodies := make([]string, totalPages)
    allBodies[0] = page1Body
    sources := make([]string, totalPages)
    sources[0] = page1URL

    if totalPages > 1 {
        remainingURLs := make([]string, 0, totalPages-1)
        for page := 2; page <= totalPages; page++ {
            pageURL := fmt.Sprintf("%s/api/deaths?world=%d&page=%d", strings.TrimRight(resolvedBaseURL, "/"), worldID, page)
            remainingURLs = append(remainingURLs, pageURL)
            sources[page-1] = pageURL
        }

        bodyMap, batchErr := oc.BatchFetchJSON(c.Request.Context(), remainingURLs)
        if batchErr != nil {
            return endpointResult{Sources: sources}, batchErr
        }

        for page := 2; page <= totalPages; page++ {
            pageURL := sources[page-1]
            body, bodyOK := bodyMap[pageURL]
            if !bodyOK {
                return endpointResult{Sources: sources}, fmt.Errorf("missing page %d", page)
            }
            allBodies[page-1] = body
        }
    }

    // Step 3: Parse and aggregate all pages
    result, parseErr := scraper.AggregateDeathsPages(allBodies, canonicalWorld)
    if parseErr != nil {
        return endpointResult{Sources: sources}, parseErr
    }

    return endpointResult{
        PayloadKey: "deaths",
        Payload:    result,
        Sources:    sources,
    }, nil
}
```

## v2 Endpoint Matrix

| v2 Endpoint | Handler Pattern | Upstream Calls | v1 Equivalent |
|---|---|---|---|
| `GET /v2/worlds` | A (thin) | 1 CDP call | `/v1/worlds` |
| `GET /v2/world/:name` | A (thin) | 1 CDP call | `/v1/world/:name` |
| `GET /v2/world/all` | B (fan-out) | 1 BatchFetch (14 URLs) | `/v1/world/all` (sequential loop) |
| `GET /v2/world/:name/details` | A / B | 1 or 14 | `/v1/world/:name/details` |
| `GET /v2/world/:name/dashboard` | A / B | 1 or 14 | `/v1/world/:name/dashboard` |
| `GET /v2/highscores/:world/:cat/:voc` | A (thin) | 1 CDP call, returns all entries | `/v1/highscores/.../all` (paginated response) |
| `GET /v2/killstatistics/:world` | A / B | 1 or 14 | `/v1/killstatistics/:world` |
| `GET /v2/deaths/:world` | A (thin) | 1 CDP call (page 1 only) | `/v1/deaths/:world` |
| `GET /v2/deaths/:world/all` | C (paginated) | 1 + BatchFetch(N-1 pages) | `/v1/deaths/:world/all` (sequential) |
| `GET /v2/banishments/:world` | A (thin) | 1 CDP call | `/v1/banishments/:world` |
| `GET /v2/banishments/:world/all` | C (paginated) | 1 + BatchFetch(N-1 pages) | `/v1/banishments/:world/all` (sequential) |
| `GET /v2/transfers` | A (thin) | 1 CDP call | `/v1/transfers` |
| `GET /v2/transfers/all` | C (paginated) | 1 + BatchFetch(N-1 pages) | `/v1/transfers/all` (sequential) |
| `GET /v2/character/:name` | A (thin) | 1 CDP call | `/v1/character/:name` |
| `GET /v2/guild/:name` | A (thin) | 1 CDP call | `/v1/guild/:name` |
| `GET /v2/guilds/:world` | A (thin) | 1 CDP call | `/v1/guilds/:world` |
| `GET /v2/guilds/:world/all` | C (paginated) | 1 + BatchFetch(N-1 pages) | `/v1/guilds/:world/all` (sequential) |
| `GET /v2/boosted` | A (thin) | 1 CDP call | `/v1/boosted` |
| `GET /v2/maintenance` | A (thin) | 1 CDP call | `/v1/maintenance` |
| `GET /v2/auctions/current/:page` | A (thin) | 1 CDP call | `/v1/auctions/current/:page` |
| `GET /v2/auctions/current/all` | C (paginated) | 1 + BatchFetch(N-1 pages, chunked 20) | `/v1/auctions/current/all` (sequential) |
| `GET /v2/auctions/history/:page` | A (thin) | 1 CDP call | `/v1/auctions/history/:page` |
| `GET /v2/auctions/history/all` | C (paginated) | 1 + BatchFetch(N-1 pages, chunked 20) | `/v1/auctions/history/all` (sequential) |
| `GET /v2/auctions/:id` | A (thin) | 1 CDP call | `/v1/auctions/:id` |
| `GET /v2/news/*` | A (thin) | 1 CDP call | `/v1/news/*` |
| `GET /v2/outfit` | reuse v1 | no scraping | `/v1/outfit` |

## FlareSolverr Configuration Changes

### Environment variables (sidecar container)

```yaml
env:
  - name: PROMETHEUS_ENABLED
    value: "true"
  - name: LOG_LEVEL
    value: info
  - name: HEADLESS
    value: "true"            # already default, explicit
  - name: BROWSER_TIMEOUT
    value: "40000"           # 40s internal timeout (default, explicit)
  - name: DISABLE_MEDIA
    value: "true"            # NEW: skip images/CSS/fonts
```

### Session init changes

FlareSolverr `request.get` calls include:

```json
{
  "cmd": "request.get",
  "url": "https://rubinot.com.br/",
  "session": "rubinot-cdp",
  "maxTimeout": 120000,
  "session_ttl_minutes": 30,
  "disableMedia": true
}
```

## Deployment Changes

### Replica count

```yaml
spec:
  replicas: 5    # was: 30
```

### API container resources

```yaml
containers:
  - name: api
    resources:
      requests:
        cpu: 250m
        memory: 256Mi
      limits:
        cpu: 500m
        memory: 512Mi
```

### New environment variable

```yaml
- name: CDP_POOL_SIZE
  value: "4"
```

## Telemetry

### Existing metrics populated by v2

| Metric | Labels | Source |
|---|---|---|
| `rubinotdata_cdp_fetch_duration_seconds` | — | CachedFetcher on each CDP call |
| `rubinotdata_cdp_fetch_requests_total` | `ok`, `error`, `non_json` | CachedFetcher |
| `rubinotdata_upstream_status_total` | endpoint, status_code | CachedFetcher on successful fetches |

### New metrics for v2

| Metric | Type | Labels | Purpose |
|---|---|---|---|
| `rubinotdata_cache_requests_total` | counter | `hit`, `miss` | Cache effectiveness |
| `rubinotdata_cache_duration_seconds` | histogram | — | Time spent in cache lookup |
| `rubinotdata_singleflight_dedup_total` | counter | — | Requests served by joining in-flight call |
| `rubinotdata_cdp_pool_available` | gauge | — | Number of available tabs in pool |
| `rubinotdata_cdp_pool_rebuilds_total` | counter | — | Tab reconnection count |

Note: `CacheRequests` and `CacheDuration` placeholder metrics already exist in `telemetry.go` — populate them.

## File Layout

```
internal/
├── api/
│   ├── router.go              # UNCHANGED — add registerV2Routes() call only
│   ├── router_v2.go           # NEW — v2 route registration
│   ├── handlers_v2.go         # NEW — v2 handler implementations
│   ├── handler.go             # UNCHANGED — handleEndpoint, endpointResult reused
│   ├── envelope.go            # UNCHANGED — same response format
│   └── middleware.go          # UNCHANGED
├── scraper/
│   ├── client.go              # UNCHANGED — v1 scraper
│   ├── cdp.go                 # MODIFIED — add ConnectToURL, CreateTarget, Navigate methods
│   ├── cdp_pool.go            # NEW — CDPPool with acquire/release
│   ├── cdp_pool_test.go       # NEW — pool lifecycle, acquire/release, rebuild tests
│   ├── cached_fetcher.go      # NEW — singleflight + cache + pool integration
│   ├── cached_fetcher_test.go # NEW — cache hit/miss, singleflight dedup, batch tests
│   ├── optimized_client.go    # NEW — thin wrapper for handlers
│   ├── telemetry.go           # MODIFIED — populate CacheRequests, CacheDuration, add new metrics
│   ├── highscores.go          # UNCHANGED (v1 uses it)
│   ├── deaths.go              # UNCHANGED (v1 uses it)
│   └── ...                    # all other scraper files UNCHANGED
└── domain/                    # UNCHANGED — same domain models
```

## Testing Strategy

### Unit tests

| Test file | Test cases |
|---|---|
| `cdp_pool_test.go` | Pool init with mock CDP, acquire returns tab, release makes it available, acquire blocks when all busy, context cancellation unblocks acquire, rebuild on dead tab, rebuild when no healthy tabs fails |
| `cached_fetcher_test.go` | Cache miss → fetches → stores, cache hit → returns cached (no CDP call), expired entry → refetches, singleflight dedup (2 concurrent callers for same URL → 1 CDP call), singleflight error sharing, BatchFetchJSON partial cache hits, BatchFetchJSON all cached, error not cached |
| `optimized_client_test.go` | FetchJSON delegates to CachedFetcher + parseJSONBody, BatchFetchJSON passes through |
| `handlers_v2_test.go` | Thin wrapper handler (mock OptimizedClient), fan-out handler (14 worlds batched), paginated fan-out (deaths with 3 pages), highscores returns all entries (no pagination in response) |

### Integration tests

| Scenario | What it validates |
|---|---|
| v2 single world → response matches v1 format | Payload compatibility |
| v2 all worlds → all 14 in one call | BatchFetch parallelism |
| v2 deaths/all → paginated fan-out | Page discovery + parallel fetch |
| v2 highscores → full 1000 entries | No client-side pagination |
| v1 endpoints still work after v2 added | No regression |
| CDP tab dies mid-request → recovery | Pool rebuild path |
| Concurrent identical requests → singleflight | Verify only 1 CDP call made |

### Test commands

```bash
make test                    # go test ./... -v
go test ./internal/scraper/... -run TestCDPPool -v
go test ./internal/scraper/... -run TestCachedFetcher -v
go test ./internal/api/... -run TestV2 -v
```

## Implementation Plan — Commit Sequence

### Phase 1: CDP Infrastructure

```
commit 1: feat(scraper): add ConnectToURL, CreateTarget, Navigate to CDPClient
  Phase: 1 — CDP infrastructure
  Layers: backend
  Changes:
    - internal/scraper/cdp.go — add ConnectToURL(), CreateTarget(), Navigate() methods
    - internal/scraper/cdp_test.go — test CreateTarget response parsing, Navigate command
  Tests:
    - TestCDPClient_ConnectToURL — verifies direct WS connection
    - TestCDPClient_CreateTarget — verifies Target.createTarget CDP command
    - TestCDPClient_Navigate — verifies Page.navigate CDP command

commit 2: feat(scraper): add CDPPool with acquire/release and tab recovery
  Phase: 1 — CDP infrastructure
  Layers: backend
  Changes:
    - internal/scraper/cdp_pool.go — CDPPool struct, Init(), Acquire(), Release(), rebuildTab(), Close()
    - internal/scraper/cdp_pool_test.go — full test suite
  Tests:
    - TestCDPPool_Init — pool creates N tabs via CreateTarget
    - TestCDPPool_AcquireRelease — acquire gets tab, release returns it
    - TestCDPPool_AcquireBlocks — blocks when all tabs busy, unblocks on release
    - TestCDPPool_ContextCancel — acquire returns error on cancelled context
    - TestCDPPool_RebuildDeadTab — dead tab is recreated transparently
    - TestCDPPool_Close — all tabs closed

commit 3: feat(scraper): add new CDP pool telemetry metrics
  Phase: 1 — CDP infrastructure
  Layers: backend
  Changes:
    - internal/scraper/telemetry.go — add cdp_pool_available gauge, cdp_pool_rebuilds_total counter, populate CacheRequests/CacheDuration placeholders, add singleflight_dedup_total
  Tests:
    - (telemetry registration verified by existing TestTelemetry if present, or verified in phase 2 integration)
```

### Phase 2: Caching Layer

```
commit 4: feat(scraper): add CachedFetcher with singleflight and TTL cache
  Phase: 2 — Caching layer
  Layers: backend
  Changes:
    - internal/scraper/cached_fetcher.go — CachedFetcher struct, FetchJSON(), BatchFetchJSON()
    - internal/scraper/cached_fetcher_test.go — full test suite
  Tests:
    - TestCachedFetcher_CacheMiss — calls CDP, stores result
    - TestCachedFetcher_CacheHit — returns cached, no CDP call
    - TestCachedFetcher_CacheExpiry — expired entry triggers refetch
    - TestCachedFetcher_SingleflightDedup — 2 goroutines, same URL, 1 CDP call
    - TestCachedFetcher_SingleflightErrorSharing — error propagated to all waiters
    - TestCachedFetcher_ErrorNotCached — next call after error retries
    - TestCachedFetcher_BatchPartialCache — some URLs cached, only uncached fetched
    - TestCachedFetcher_BatchAllCached — no CDP call when all cached

commit 5: feat(scraper): add OptimizedClient wrapper
  Phase: 2 — Caching layer
  Layers: backend
  Changes:
    - internal/scraper/optimized_client.go — OptimizedClient struct, FetchJSON(), BatchFetchJSON()
    - internal/scraper/optimized_client_test.go
  Tests:
    - TestOptimizedClient_FetchJSON — delegates to CachedFetcher + parseJSONBody
    - TestOptimizedClient_BatchFetchJSON — passes through to CachedFetcher
```

### Phase 3: v2 API Handlers

```
commit 6: feat(api): add v2 router with thin wrapper handlers
  Phase: 3 — v2 API
  Layers: backend
  Changes:
    - internal/api/router_v2.go — registerV2Routes(), v2 route group
    - internal/api/handlers_v2.go — thin wrapper handlers (single-resource endpoints)
    - internal/api/router.go — add one line: registerV2Routes(router, optimizedClient) after v1 block
    - internal/api/handlers_v2_test.go — handler tests
  Tests:
    - TestV2GetWorld — single world returns correct envelope
    - TestV2GetCharacter — character lookup via OptimizedClient
    - TestV2GetGuild — guild lookup
    - TestV2GetHighscores — returns all entries, no pagination in response
    - TestV2GetBoosted — simple proxy
    - TestV1Unchanged — v1 endpoints still respond correctly

commit 7: feat(api): add v2 fan-out handlers for all-worlds endpoints
  Phase: 3 — v2 API
  Layers: backend
  Changes:
    - internal/api/handlers_v2.go — add v2GetAllWorlds, v2GetAllWorldDetails, v2GetAllWorldDashboard, v2GetAllKillstatistics
    - internal/api/handlers_v2_test.go — fan-out handler tests
  Tests:
    - TestV2GetAllWorlds — 14 worlds via BatchFetch, correct aggregation
    - TestV2GetAllWorldDetails — fan-out with world details
    - TestV2GetAllKillstatistics — fan-out for killstats

commit 8: feat(api): add v2 paginated fan-out handlers
  Phase: 3 — v2 API
  Layers: backend
  Changes:
    - internal/api/handlers_v2.go — add v2GetAllDeaths, v2GetAllBanishments, v2GetAllTransfers, v2GetAllGuilds, v2GetAllCurrentAuctions, v2GetAllAuctionHistory
    - internal/scraper/deaths.go — export DeathsTotalPagesFromBody, AggregateDeathsPages (or add to a new v2 helper file)
    - internal/scraper/banishments.go — export BanishmentsTotalPagesFromBody
    - internal/scraper/transfers.go — export TransfersTotalPagesFromBody
    - internal/scraper/guilds.go — export GuildsTotalPagesFromBody
    - internal/scraper/auctions.go — export AuctionsTotalPagesFromBody
    - internal/api/handlers_v2_test.go — paginated fan-out tests
  Tests:
    - TestV2GetAllDeaths — page 1 discover, BatchFetch pages 2-6, aggregate
    - TestV2GetAllBanishments — same pattern
    - TestV2GetAllTransfers — same pattern
    - TestV2GetAllGuilds — same pattern
    - TestV2GetAllCurrentAuctions — handles 300+ pages in batches of 20
```

### Phase 4: Initialization & Wiring

```
commit 9: feat(server): wire CDPPool and OptimizedClient into main startup
  Phase: 4 — Wiring
  Layers: backend
  Changes:
    - cmd/server/main.go — initialize CDPPool, CachedFetcher, OptimizedClient; pass to NewRouter
    - internal/api/router.go — modify NewRouter signature to accept *OptimizedClient; call registerV2Routes
    - go.mod — add golang.org/x/sync dependency (for singleflight)
  Tests:
    - TestNewRouter — router initializes with OptimizedClient, v2 routes registered
    - Manual: `make run` → hit /v2/worlds
```

### Phase 5: Configuration Constants

```
commit 10: chore(scraper): bump cdpBatchSize from 6 to 20
  Phase: 5 — Configuration
  Layers: backend
  Changes:
    - internal/scraper/client.go — change cdpBatchSize = 20
  Tests:
    - Existing batch tests still pass with larger batch size

commit 11: feat(scraper): add CDP_POOL_SIZE env var support
  Phase: 5 — Configuration
  Layers: backend
  Changes:
    - internal/scraper/cdp_pool.go — read CDP_POOL_SIZE env var (default 4)
    - cmd/server/main.go — pass pool size from env
  Tests:
    - TestCDPPoolSize_EnvVar — verifies env var parsing
```

### Phase 6: FlareSolverr Optimization

```
commit 12: feat(scraper): add disableMedia and session_ttl_minutes to FlareSolverr requests
  Phase: 6 — FlareSolverr
  Layers: backend
  Changes:
    - internal/scraper/client.go — add DisableMedia and SessionTTLMinutes fields to flareSolverrRequest, set in initFlareSolverrSession and Fetch
  Tests:
    - TestFlareSolverrRequest_DisableMedia — verify request body includes disableMedia: true
    - TestFlareSolverrRequest_SessionTTL — verify session_ttl_minutes: 30
```

### Phase 7: Deployment

```
commit 13: chore(deploy): reduce replicas to 5, add resource limits, add CDP_POOL_SIZE
  Phase: 7 — Deployment
  Layers: infra
  Changes:
    - (GitOps repo: omni-cddlabs-casa) manifests/cddlabs/apps/rubinot/ — update deployment:
      - spec.replicas: 5
      - api container resources (requests: 250m/256Mi, limits: 500m/512Mi)
      - add CDP_POOL_SIZE: "4" env var
      - add DISABLE_MEDIA: "true" to flaresolverr env
  Tests:
    - kubectl rollout status
    - Verify 5 pods running, 2/2 containers
    - Hit /v2/worlds and /v1/worlds — both respond
    - Monitor Grafana metrics for 30 min

commit 14: docs: update CLAUDE.md with v2 API documentation
  Phase: 7 — Documentation
  Layers: docs
  Changes:
    - CLAUDE.md — add v2 section documenting new endpoints and configuration
```

### Phase 8: Review & Cleanup

```
commit 15: review and apply PR feedback
  - Do a PR review of the changes as if you are another engineer reviewing
  - Apply recommended changes and iterate on tests
  - Remove any unnecessary comments in the code
```

## Expected Impact

### Before (current state)

| Metric | Value |
|---|---|
| Replicas | 30 |
| CDP channels total | 30 (1 per pod, mutex-serialized) |
| Upstream calls for /world/all | 14 sequential × 30 pods = 420 potential |
| Memory footprint | ~18Gi |
| CPU footprint | ~16 cores |
| p99 latency | 5s (scraper tail) |

### After (projected)

| Metric | Value |
|---|---|
| Replicas | 5 |
| CDP channels total | 20 (4 per pod × 5 pods) |
| Upstream calls for /v2/world/all | 1 BatchFetch (14 URLs) with singleflight dedup |
| Memory footprint | ~3.5Gi (5 × ~700Mi including extra tabs) |
| CPU footprint | ~2.5 cores |
| p99 latency (projected) | < 1s (cache hits) to ~4s (cold fan-out) |
| Cache hit rate (projected) | 60-80% during normal traffic |

### Key improvements

- **6x fewer replicas** (30 → 5)
- **~80% less memory** (18Gi → 3.5Gi)
- **~85% less CPU** (16 cores → 2.5 cores)
- **Upstream pressure reduced ~95%** through singleflight + cache + fewer pods
- **Fan-out latency ~4x faster** (sequential → parallel BatchFetch)
- **No v1 regression** — frozen code paths

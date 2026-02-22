# Target Architecture (Rubinot Data)

## Architecture goals

1. Stable, versioned JSON contract for consumers
2. Predictable freshness without hammering upstream website
3. Anti-bot resilience by design (not as afterthought)
4. Controlled degradation during upstream disruptions
5. Clear operational model in Kubernetes

## Proposed logical components

```text
[Clients]
   |
   v
[API Gateway / Edge Policies]
   |
   v
[rubinot-data-api]
   |-- reads --> [Redis read-through cache]
   |-- reads --> [PostgreSQL canonical store]
   |-- triggers --> [Job Orchestrator]
                     |-- runs --> [Scraper Workers (Playwright/Puppeteer)]
                     |-- writes --> [PostgreSQL + Redis]

[Assets Generator Pipeline] ---> [rubinot-data-assets JSON]
                                  ^ consumed by API/runtime to enrich responses
```

## Domain boundaries

- **Serving boundary**: only returns normalized/versioned API responses
- **Collection boundary**: scraping & parse jobs only
- **Enrichment boundary**: static/reference datasets (assets)
- **Docs boundary**: generated OpenAPI + changelogs

## API contract strategy

- Start with `/v1` in new repo
- Keep compatibility adapter for existing `rubinot-live` paths where feasible
- Introduce endpoint capability flags:
  - `stable`: contract reliable
  - `beta`: parser still evolving
  - `restricted`: expensive or upstream-sensitive

## Data model direction

- Canonical entities: world, character, guild, highscores, deaths, houses, transfers, news, bans
- Store source metadata per record:
  - `fetched_at`, `source_path`, `parser_version`, `confidence`, `stale_after`
- Support stale-while-revalidate semantics in API responses

## Reliability patterns

1. **Scraper profile rotation** (UA, viewport, pacing)
2. **Adaptive backoff by endpoint and failure class**
3. **Circuit breaker by upstream path** (avoid cascading bans)
4. **Fallback hierarchy**:
   - fresh cache
   - stale cache with explicit stale headers
   - graceful error contract

## Security and abuse control

- API key tiers (optional at launch, required before public growth)
- endpoint-specific rate limits
- bot/user-agent anomaly detection
- per-client quota telemetry

## Observability

- RED metrics per endpoint (rate/errors/duration)
- scrape success ratio per parser
- challenge-detection metric (Cloudflare event counter)
- freshness lag dashboards by entity type

## Deployment model

- Namespace split: API vs workers
- HPA for API, queue-driven autoscaling for workers
- separated low/high-risk jobs to protect core endpoints
- maintenance mode switch for expensive collectors

# TibiaData Reference Analysis

## Repositories reviewed

From `https://github.com/tibiadata`:

- `tibiadata/tibiadata-api-go`
- `tibiadata/tibiadata-api-assets`
- `tibiadata/tibiadata-api-docs`
- `tibiadata/tibiadata-helm-charts`

## Notable maturity patterns

1. **Separation by concern**
   - API runtime in one repo
   - static/near-static assets in another
   - generated/public docs in another
   - deployment charts in dedicated repo

2. **Versioned API strategy**
   - clear major versions (`v4`, historical `v3/v2/v1`)
   - explicit deprecations and endpoint lifecycle

3. **Deployment flexibility**
   - Docker, docker-compose, Helm
   - guidance for edge concerns (cache/rate-limit/auth via gateway)

4. **Operational realism**
   - documented restrictions for heavy endpoints
   - expectation that upstream target limits must be respected

5. **Contract clarity**
   - endpoint catalog and consistency
   - docs generated/distributed from release process

## What to copy (directly)

- Multi-repo split by concern
- Explicit contract versioning
- "restricted mode" model for high-cost endpoints
- Published docs workflow from release artifacts
- Asset-generation pipeline for data that is costly to scrape repeatedly

## What to adapt (not copy blindly)

- Language/runtime choice can remain Node.js initially (migration risk control)
- Endpoint naming should map from current Rubinot users first, then normalize
- Anti-bot handling must be Rubinot-specific (Cloudflare profile differs)
- Traffic profile may differ from TibiaData; SLOs should be realistic from day one

## Initial target model for Rubinot

Recommended repository topology:

1. `rubinot-data` (this repo): architecture, plans, ADRs, governance
2. future `rubinot-data-api`: serving layer + API contract + versioning
3. future `rubinot-data-assets`: static mappings/snapshots (world/town/category metadata)
4. future `rubinot-data-docs`: generated API docs and changelog docs
5. optional `rubinot-data-helm`: deploy packaging if chart lifecycle needs separation

## Strategic takeaway

TibiaData’s biggest advantage is not only scraping capability — it is **productized consistency** (versioning, boundaries, release discipline). Rubinot should optimize for that same consistency to reduce breakage and improve trust for consumers.

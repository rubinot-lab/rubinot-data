# Rubinot-Data Porting Status (TibiaData-style)

_Last updated: 2026-02-22_

## Goal
Port core TibiaData API patterns into `rubinot-data` while preserving Rubinot-specific behavior and deployment model.

## What is already implemented in `rubinot-data`

### API + contracts
- `GET /v1/world/:name`
- `GET /v1/houses/:world/:town` (new)
- Health/system routes: `/`, `/ping`, `/healthz`, `/readyz`, `/versions`
- Response envelope with `information` (`timestamp`, `status`, `sources`) and payload section (`world`/`houses`)

### Scraping runtime
- FlareSolverr-backed fetch path for world and houses endpoints
- Houses endpoint uses TibiaData-like upstream URL shape:
  - `?subtopic=houses&world=<world>&town=<town>&type=houses`
  - `?subtopic=houses&world=<world>&town=<town>&type=guildhalls`
- Initial houses parser extracts:
  - `house_id`, `name`, `size`, `rent`, `status`, `rented`, `auctioned`

### Observability
- Prometheus endpoint added: `GET /metrics`
- App metrics added:
  - `rubinot_scrape_requests_total{endpoint,status}`
  - `rubinot_scrape_duration_seconds{endpoint}`
  - `rubinot_parse_duration_seconds{endpoint}`
- OTel tracing added:
  - Gin middleware instrumentation
  - scraper spans (`FetchWorld`, `FetchHouses`, FlareSolverr request)
  - OTLP exporter config via env vars

## Git references (pushed)
- `rubinot-data` commit: `20795bb`
  - message: `feat(api): add otel tracing, prometheus metrics, and houses endpoint`

## Alignment references used
- `planning/tibiadata/*` (routes, implementation notes)
- `planning/rubinot-data/*` (target endpoint mapping)
- `rubinot-live` source for practical houses fetch path and behavior differences

## Current gap (known)
Code is pushed, but cluster may still run an older pinned image digest until image build/publish + GitOps digest update is performed.

## Next implementation steps
1. Add `GET /v1/house/:world/:house_id` (house details) with TibiaData-compatible fields.
2. Add validation layer (world/town/house existence) before expensive fetches.
3. Expand parser robustness with fixtures/golden tests for houses HTML variants.
4. Add route-level latency/error metrics and cardinality guardrails.
5. Add Grafana dashboards/panels for Mimir + Tempo (API latency, scrape success, parser failures, trace exemplars).

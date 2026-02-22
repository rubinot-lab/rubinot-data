# TibiaData Implementation Details

## Request lifecycle (practical)

1. Gin route handler receives request.
2. Validation package checks params.
3. Handler builds upstream tibia.com URL.
4. `TibiaDataHTMLDataCollector` performs HTTP request.
5. Raw body or extracted HTML box is returned.
6. Parser implementation transforms HTML -> typed response struct.
7. JSON response returned to client.

## Error handling pattern

Central error handler maps internal/validation/upstream errors to:

- HTTP status code
- API-level `error` numeric code
- Human-readable `message`

This gives clients deterministic error contracts.

## Metadata envelope pattern

Responses include:

- API details (`version`, `release`, `commit`)
- timestamp
- source links (`tibia_urls`)
- status object

This is useful for troubleshooting and provenance.

## Proxy and host behavior

Relevant env vars include:

- `TIBIADATA_HOST` (self host metadata)
- `TIBIADATA_PROTOCOL`
- `TIBIADATA_PROXY` (+ protocol) to replace tibia.com domain in collectors
- `TIBIADATA_RESTRICTION_MODE`
- `GIN_MODE`, `GIN_TRUSTED_PROXIES`

## Why this matters for Rubinot

Rubinot faces similar anti-bot constraints. The patterns worth adopting directly:

1. pre-fetch validation to avoid wasteful traffic
2. explicit restricted mode for expensive endpoints
3. consistent error envelope and provenance metadata
4. optional upstream proxy abstraction
5. health/readiness endpoints for K8s and automation

## Known constraints in TibiaData model

- Scraping means parser fragility when HTML changes.
- Upstream throttling can still degrade data freshness.
- Some endpoint capability may need restrictions under high load.

The key is not eliminating those constraints, but making them observable and gracefully handled.

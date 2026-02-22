# TibiaData Comprehensive Reference

This folder documents how TibiaData works in practice, with focus on architecture, endpoint behavior, and operational patterns that can be reused in `rubinot-data`.

## Executive summary

TibiaData is an API facade over Tibia website data. Yes, it is fundamentally **web-scraping based** (HTML collection + parsing), with explicit validation and normalization layers.

Key implementation evidence from source:

- HTTP handlers call Tibia URLs directly (e.g. `https://www.tibia.com/community/?subtopic=characters&name=...`) in `src/webserver.go`.
- The collector function `TibiaDataHTMLDataCollector(...)` fetches HTML with Resty and parses with GoQuery (`src/webserver.go`).
- Endpoint handlers pass fetched HTML into parser implementations (`TibiaCharactersCharacterImpl`, `TibiaGuildsOverviewImpl`, etc).

## Source repositories (official)

- `tibiadata/tibiadata-api-go` — API server runtime (Gin + Resty + GoQuery)
- `tibiadata/tibiadata-api-assets` — static-ish asset generation (worlds/towns/houses/creatures/spells)
- `tibiadata/tibiadata-api-docs` — generated Swagger docs published site
- `tibiadata/tibiadata-helm-charts` — Kubernetes packaging

## How TibiaData works (high-level)

1. Client calls `/v4/...` endpoint.
2. Handler validates path params (world/category/vocation/etc).
3. Handler builds Tibia upstream URL.
4. Collector fetches HTML from tibia.com (with retries, timeout, user-agent, optional proxy).
5. Parser extracts structured data.
6. API returns normalized JSON envelope with metadata/status.

## Runtime and framework choices

- Language: Go
- Web framework: Gin
- HTTP client: Resty
- HTML parser: GoQuery
- API docs: Swagger/OpenAPI
- Compression: gzip middleware
- Liveness/readiness: `/healthz`, `/readyz`

## API versioning model

- `v4` is active.
- `v3` exists as deprecated compatibility shape.
- Contract versioning is explicit and central to API UX.

## Restriction mode

TibiaData has a runtime switch (`TIBIADATA_RESTRICTION_MODE`) to protect expensive endpoints. In highscores specifically, non-`all` vocation is rejected under restriction mode.

This is an important production pattern for anti-abuse/upstream-protection.

## Collector behavior details

From `TibiaDataHTMLDataCollector`:

- timeout: 5s
- retries: 2
- explicit user-agent
- optional proxy replacement via `TIBIADATA_PROXY`
- no redirect policy (used to detect maintenance redirect)
- special handling for:
  - 403 => throttling/rate-limit signal
  - 302 to maintenance => maintenance mode signal
  - other status => upstream unknown error

## Validation model

Validation package blocks invalid requests before scraping:

- character, guild, creature, spell validation
- world existence checks
- highscore category/vocation/page checks
- house/town existence checks

This is crucial for reducing unnecessary upstream load and limiting abuse.

## Assets pipeline purpose

`tibiadata-api-assets` exists because some data is expensive, repetitive, or missing from single-page fetches.

The generator builds `output.json` with assets such as:

- worlds
- towns
- houses (+ house type)
- creatures
- spells

This acts as an enrichment/cache source for runtime usage.

## Why TibiaData has long-term stability

1. Clear repo boundaries by concern
2. Strict versioned API contract
3. Parameter validation before scrape
4. Restriction mode and upstream-aware behavior
5. Dedicated docs and deployment artifacts
6. Asset generation to reduce repeated expensive fetches

## Suggested carryover for rubinot-data

- Keep contract-first versioned API (`/v1`, then `/v2` later)
- Build validation + restriction mode from day 1
- Separate runtime vs assets vs docs concerns
- Add parser fixture tests and challenge detection metrics

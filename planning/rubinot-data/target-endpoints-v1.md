# Target `rubinot-data` Endpoints (TibiaData-inspired)

Proposed base:
- `https://api.rubinot.dev/v1`

Design rules:

1. Path-style resources where possible
2. Validation-first (world/category/vocation/etc)
3. `restricted` mode for expensive paths
4. Consistent metadata + error envelope

## Core routes

- `GET /v1/worlds`
- `GET /v1/world/:name`

- `GET /v1/character/:name`

- `GET /v1/guild/:name`
- `GET /v1/guilds/:world`

- `GET /v1/highscores/:world/:category/:vocation/:page`

- `GET /v1/house/:world/:house_id`
- `GET /v1/houses/:world/:town`

- `GET /v1/killstatistics/:world`

- `GET /v1/news/archive`
- `GET /v1/news/archive/:days`
- `GET /v1/news/id/:news_id`
- `GET /v1/news/latest`

## Rubinot-specific additions (beyond TibiaData baseline)

- `GET /v1/deaths/:world`
- `GET /v1/transfers`
- `GET /v1/banishments/:world`
- `GET /v1/events/schedule`
- `GET /v1/auctions/current/:world/:page`
- `GET /v1/auctions/history/:world/:page`
- `GET /v1/auctions/:id`

## Control/system routes

- `GET /ping`
- `GET /healthz`
- `GET /readyz`
- `GET /versions`

## Contract expectations

- Every response includes `information.api`, `information.timestamp`, and status metadata.
- For stale data fallback, include explicit freshness fields and headers.
- Error object includes both HTTP and domain-specific code.

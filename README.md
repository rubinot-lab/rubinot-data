# rubinot-data

Go API that scrapes `rubinot.com.br` through FlareSolverr and exposes normalized JSON contracts for Rubinot game data.

Live at **https://data.rubinot.dev** — interactive docs at [/docs](https://data.rubinot.dev/docs).

## Status

- Parity target: `rubinot-live` endpoint contracts E1-E18
- Envelope contract: implemented for all API responses
- Validation-first routing: implemented (invalid inputs are rejected before scrape)
- Time normalization: Brazilian upstream timestamps converted to UTC RFC3339 where parser supports date fields

## Data platform roadmap

`rubinot-data` is currently the live scrape layer. Next in the stack:

- `rubinot-api` (new): consume `rubinot-data`, persist historical snapshots into PostgreSQL, process stats and progression, and expose long-term endpoints.
- `rubinot-eve` (new): consume `rubinot-api` and provide visualization + analytics experiences for players/guilds/worlds.

Both repos are scaffolded for Buildah + ArgoCD GitOps workflows.

## Requirements

- Go 1.23+
- Docker (for FlareSolverr and local compose)

## Quick Start

```bash
make docker-up
make run
```

Service defaults:

- API: `http://localhost:8080`
- FlareSolverr: `http://localhost:8191/v1`
- Metrics: `http://localhost:8080/metrics`

Health checks:

```bash
curl -s http://localhost:8080/ping
curl -s http://localhost:8080/readyz
curl -s http://localhost:8080/versions
```

## Response Envelope

All `/v1/*` responses use:

```json
{
  "information": {
    "api": { "version": 1, "release": "v0.2.0", "commit": "abc1234" },
    "timestamp": "2026-02-22T15:04:05Z",
    "status": { "http_code": 200, "message": "ok" },
    "sources": ["https://www.rubinot.com.br/?subtopic=worlds&world=Belaria"]
  },
  "<payload_key>": {}
}
```

On errors, `information.status` includes `error` and `message` and no payload key.

## v2 Endpoints (CDP-optimized)

v2 uses a CDP connection pool (4 tabs/pod), singleflight request deduplication, and a 5-second TTL in-memory cache. Endpoints that accept a world name also accept `all` for fan-out across every world. Paginated endpoints have `/all` variants that fetch every page via parallel `Promise.allSettled()`.

| Endpoint | Payload key | Notes |
|---|---|---|
| `GET /v2/worlds` | `worlds` | All worlds with status and player counts |
| `GET /v2/world/:name` | `world` | World detail with online players; `:name=all` for batch |
| `GET /v2/world/:name/details` | `world` | World + full character info for each online player |
| `GET /v2/world/:name/dashboard` | `dashboard` | World overview: players, recent deaths, killstatistics |
| `GET /v2/highscores/:world/:category/:vocation` | `highscores` | Highscores for a world/category/vocation combo |
| `GET /v2/killstatistics/:world` | `killstatistics` | Kill stats; `:world=all` for batch |
| `GET /v2/deaths/:world` | `deaths` | Recent deaths; query `page`, `level`, `pvp`, `guild` |
| `GET /v2/deaths/:world/all` | `deaths` | All death pages merged; query `level`, `pvp`, `guild` |
| `GET /v2/banishments/:world` | `banishments` | Banishments; query `page` |
| `GET /v2/banishments/:world/all` | `banishments` | All banishment pages merged |
| `GET /v2/transfers` | `transfers` | Transfers; query `world`, `level`, `page` |
| `GET /v2/transfers/all` | `transfers` | All transfer pages merged; query `world`, `level` |
| `GET /v2/character/:name` | `character` | Character lookup |
| `GET /v2/guild/:name` | `guild` | Guild detail |
| `GET /v2/guilds/:world` | `guilds` | Guild list; query `page`; `:world=all` for batch |
| `GET /v2/guilds/:world/all` | `guilds` | All guild pages merged |
| `GET /v2/boosted` | `boosted` | Today's boosted boss and creature |
| `GET /v2/maintenance` | `maintenance` | Server maintenance status |
| `GET /v2/auctions/current/:page` | `auctions` | Current auctions (single page) |
| `GET /v2/auctions/current/all` | `auctions` | All current auctions merged |
| `GET /v2/auctions/history/:page` | `auctions` | Auction history (single page) |
| `GET /v2/auctions/history/all` | `auctions` | All auction history merged |
| `GET /v2/auctions/:id` | `auction` | Auction detail by ID |
| `GET /v2/news/id/:news_id` | `news` | News article or ticker by ID |
| `GET /v2/news/archive` | `newslist` | Archive; query `days` (default 90) |
| `GET /v2/news/latest` | `newslist` | Latest articles |
| `GET /v2/news/newsticker` | `newslist` | Ticker entries |
| `GET /v2/outfit` | image | Outfit image proxy (same params as v1) |
| `GET /v2/outfit/:name` | image | Outfit by character name |

## v1 Endpoints (legacy)

v1 uses the original single-CDP-connection path through FlareSolverr. These routes remain frozen and untouched.

| Endpoint | Payload key | Notes |
|---|---|---|
| `GET /v1/worlds` | `worlds` | Worlds overview |
| `GET /v1/world/:name` | `world` | Canonical world validation |
| `GET /v1/character/:name` | `character` | Character name validation |
| `GET /v1/guild/:name` | `guild` | Guild name validation |
| `GET /v1/guilds/:world` | `guilds` | World name -> world id mapping |
| `GET /v1/highscores/:world/:category/:vocation/:page` | `highscores` | Redirects from shorter routes |
| `GET /v1/killstatistics/:world` | `killstatistics` | World validation |
| `GET /v1/news/id/:news_id` | `news` | `news_id` must be int > 0 |
| `GET /v1/news/archive` | `newslist` | Optional query `days` (default `90`) |
| `GET /v1/news/latest` | `newslist` | Latest list |
| `GET /v1/news/newsticker` | `newslist` | Newsticker list |
| `GET /v1/deaths/:world` | `deaths` | Optional query `guild`, `level`, `pvp` |
| `GET /v1/transfers` | `transfers` | Optional query `world`, `level`, `page` |
| `GET /v1/banishments/:world` | `banishments` | Optional query `page` |
| `GET /v1/events/schedule` | `events` | Optional query `month`, `year` |
| `GET /v1/auctions/current/:page` | `auctions` | `page` must be int >= 1 |
| `GET /v1/auctions/history/:page` | `auctions` | `page` must be int >= 1 |
| `GET /v1/auctions/:id` | `auction` | `id` must be non-empty |

## Error Code Mapping

### Validation (HTTP 400)

- `10001-10007`: character name validation
- `11001-11008`: world/town/house/vocation/category validation
- `14001-14007`: guild name validation
- `30001-30010`: Rubinot filter/page/id validation

### Upstream and not found

| Code | HTTP | Meaning |
|---|---:|---|
| `20001` | `502` | FlareSolverr connection failure |
| `20002` | `502` | FlareSolverr non-200 response |
| `20003` | `502` | Cloudflare challenge still present |
| `20004` | `404` | Upstream entity not found |
| `20005` | `503` | Upstream maintenance mode |
| `20006` | `502` | Upstream forbidden/rate limited |
| `20007` | `502` | Unknown upstream error |
| `20008` | `504` | FlareSolverr timeout |

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `GIN_MODE` | `release` | Gin mode |
| `RUBINOT_BASE_URL` | `https://www.rubinot.com.br` | Upstream base URL |
| `FLARESOLVERR_URL` | `http://flaresolverr.network.svc.cluster.local:8191/v1` | FlareSolverr endpoint |
| `SCRAPE_MAX_TIMEOUT_MS` | `120000` | FlareSolverr `maxTimeout` |
| `SCRAPE_MAX_CONCURRENCY` | `8` | Global scrape concurrency semaphore |
| `APP_VERSION` | `dev` | Service version in envelope/versions/OTel |
| `APP_COMMIT` | `dev` | Commit SHA in envelope/versions |
| `OTEL_SERVICE_NAME` | `rubinot-data` | OTel service name |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `k8s-monitoring-cddlabs-prod-alloy-receiver.observability.svc.cluster.local:4317` | OTel exporter endpoint |

## Observability

Prometheus metrics:

- `rubinotdata_http_requests_total{route,method,status_code}`
- `rubinotdata_http_request_duration_seconds{route,method,status_code}`
- `rubinotdata_scrape_requests_total{endpoint,status}`
- `rubinotdata_scrape_duration_seconds{endpoint}`
- `rubinotdata_parse_duration_seconds{endpoint}`

OpenTelemetry:

- Gin middleware tracing in `cmd/server/main.go`
- Scraper spans around fetch and parse operations

## Fixture Capture

Capture real HTML via FlareSolverr:

```bash
make fixture URL='https://www.rubinot.com.br/?subtopic=worlds' OUT='testdata/worlds/overview.html'
```

Equivalent script usage:

```bash
FLARESOLVERR_URL=http://localhost:8191/v1 \
./scripts/capture-fixture.sh 'https://www.rubinot.com.br/?subtopic=worlds' 'testdata/worlds/overview.html'
```

## Development Commands

```bash
make build
make test
make test-cover
make lint
make run
make docker-up
make docker-down
```

## Kubernetes

Deployment manifest:

- `deploy/k8s/rubinot-data.yaml`

It includes required runtime env vars for FlareSolverr endpoint, scrape timeout/concurrency, and version metadata.

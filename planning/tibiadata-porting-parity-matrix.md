# TibiaData → Rubinot-Data Porting Parity Matrix (Deep Context)

_Last updated: 2026-02-22_

## 1) Purpose

This document is the authoritative parity tracker for porting TibiaData API application patterns into `rubinot-data`.

It captures:
- full endpoint surface (TibiaData v4 + Rubinot target surface),
- current implementation status in `rubinot-data`,
- edge cases and behavior contracts TibiaData handles,
- operational concerns (restriction mode, anti-bot behavior, parser fragility, observability),
- implementation decisions and sequencing.

This is intentionally long and context-heavy so we can make consistent architectural decisions without re-discovering prior analysis.

---

## 2) Sources used for this matrix

### TibiaData application references
- `tibiadata/tibiadata-api-go` (code + README + validation + handlers)
  - Route registrations in `src/webserver.go`
  - Endpoint list + restricted mode notes in README
  - Validation + error code taxonomy in `src/validation/*`
  - Houses handling in `TibiaHousesOverview.go` and `TibiaHousesHouse.go`

### Rubinot references
- `rubinot-data/planning/tibiadata/*`
- `rubinot-data/planning/rubinot-data/*`
- `rubinot-live` implementation (especially houses explorer/parser flow)

### Current `rubinot-data` implementation
- `internal/api/router.go`
- `internal/scraper/world.go`
- `internal/scraper/houses.go`
- `internal/scraper/telemetry.go`
- `internal/observability/otel.go`

---

## 3) TibiaData v4 endpoint inventory (baseline)

TibiaData v4 endpoints (as currently documented in upstream):

### System
- `GET /`
- `GET /ping`
- `GET /healthz`
- `GET /readyz`
- `GET /versions`
- (deprecated compat endpoint: `GET /health`)

### Data endpoints
- `GET /v4/boostablebosses`
- `GET /v4/character/:name`
- `GET /v4/creature/:race`
- `GET /v4/creatures`
- `GET /v4/fansites`
- `GET /v4/guild/:name`
- `GET /v4/guilds/:world`
- `GET /v4/highscores/:world/:category/:vocation/:page`
  - plus redirect aliases for missing segments
- `GET /v4/house/:world/:house_id`
- `GET /v4/houses/:world/:town`
- `GET /v4/killstatistics/:world`
- `GET /v4/news/archive`
- `GET /v4/news/archive/:days`
- `GET /v4/news/id/:news_id`
- `GET /v4/news/latest`
- `GET /v4/news/newsticker`
- `GET /v4/spell/:spell_id`
- `GET /v4/spells`
- `GET /v4/world/:name`
- `GET /v4/worlds`

### TibiaData compatibility behavior
- v1/v2/v3 deprecation behavior with explicit responses
- metadata-rich error envelope with API-level error code taxonomy
- validation-first request guards to reduce costly upstream fetches
- restriction mode behavior on heavy endpoints (notably highscores filtering)

---

## 4) Rubinot target endpoint inventory (strategic target)

From planning (`target-endpoints-v1.md`) and Rubinot product needs:

### Core TibiaData-like target
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

### Rubinot-specific extensions
- `GET /v1/deaths/:world`
- `GET /v1/transfers`
- `GET /v1/banishments/:world`
- `GET /v1/events/schedule`
- `GET /v1/auctions/current/:world/:page`
- `GET /v1/auctions/history/:world/:page`
- `GET /v1/auctions/:id`

### System/control
- `GET /`
- `GET /ping`
- `GET /healthz`
- `GET /readyz`
- `GET /versions`
- `GET /metrics` (Prometheus)

---

## 5) Current parity status (as of now)

## Legend
- ✅ Implemented
- 🟡 Partial / scaffolded
- ⏳ Planned
- ❌ Not started

| Endpoint/Capability | TibiaData baseline | rubinot-data status | Notes |
|---|---|---:|---|
| System endpoints (`/`, `/ping`, `/healthz`, `/readyz`, `/versions`) | Yes | ✅ | Implemented in router |
| `/metrics` Prometheus | Not core v4 concern | ✅ | Added for first-class ops in K8s |
| `/v1/world/:name` | `/v4/world/:name` | ✅ | FlareSolverr-backed + envelope metadata |
| `/v1/houses/:world/:town` | `/v4/houses/:world/:town` | ✅ | Implemented with house + guildhall list fetch |
| `/v1/house/:world/:house_id` | `/v4/house/:world/:house_id` | ❌ | Next priority |
| `/v1/worlds` | `/v4/worlds` | ❌ | Not yet implemented |
| `/v1/character/:name` | `/v4/character/:name` | ❌ | Not yet implemented |
| `/v1/guild/:name` | `/v4/guild/:name` | ❌ | Not yet implemented |
| `/v1/guilds/:world` | `/v4/guilds/:world` | ❌ | Not yet implemented |
| `/v1/highscores/...` | `/v4/highscores/...` | ❌ | Not yet implemented |
| `/v1/killstatistics/:world` | `/v4/killstatistics/:world` | ❌ | Not yet implemented |
| `/v1/news/*` | `/v4/news/*` | ❌ | Not yet implemented |
| `/v1/spells*` | `/v4/spells*` | ❌ | Currently out of immediate Rubinot scope |
| Validation-first guardrail layer | Strongly present | 🟡 | Minimal formatting only today |
| Structured API error codes taxonomy | Strongly present | 🟡 | HTTP envelope present, no full numeric taxonomy |
| Restriction mode semantics | Present | ❌ | Planned |
| OTel traces | N/A in classic TibiaData | ✅ | Added, exported via OTLP |
| Prom scrape + Mimir | N/A in classic TibiaData | ✅ | Added and wired through Alloy annotations |

---

## 6) Edge cases TibiaData handles (and what we should port)

This section explicitly lists edge cases/behaviors we must port or intentionally diverge from.

## 6.1 Input normalization edge cases

### World/town normalization
- Case-insensitive world and town handling.
- Town edge case: `Ab'Dendriel` formatting correctness.
- URL escaping for names with spaces/special chars.

**Rubinot action:**
- Keep world/town canonicalization utility.
- Add explicit handling for apostrophes and multi-word towns.

## 6.2 Validation-first failures (before scrape)

TibiaData validates:
- world exists,
- town exists,
- house id exists (globally and optionally in-town),
- highscores category/vocation/page constraints,
- character/guild/spell/creature lexical validity.

This avoids expensive scrape calls for impossible requests.

**Rubinot action:**
- Build in-memory validation tables (world/town/category/vocation/house metadata).
- Return deterministic validation errors before FlareSolverr fetch.

## 6.3 Restriction mode behavior

TibiaData runs a restricted mode under high load and limits expensive query variants.
Example from docs: highscores vocation filtering may be restricted to `all`.

**Rubinot action:**
- Introduce `RUBINOT_RESTRICTION_MODE` env and endpoint policy map.
- Return explicit domain error (not generic 500) when restricted.

## 6.4 Upstream transport errors and anti-bot status handling

TibiaData error taxonomy differentiates:
- maintenance mode,
- forbidden (403, likely rate-limit/challenge),
- unexpected redirect/found,
- unknown status.

**Rubinot action:**
- Differentiate FlareSolverr transport errors vs target status vs parser errors.
- Preserve meaningful user-safe messages + stable machine code fields.

## 6.5 Houses-specific parsing edge cases

Houses involve variants:
- rented house text,
- auctioned with no bid,
- auctioned with bid + end time,
- auction ended,
- moving out date,
- transfer recipient + transfer price,
- owner sex pronoun extraction,
- house vs guildhall type,
- variable bed counts (including one-bed regex edge cases).

**Rubinot action:**
- Current implementation only captures subset (id/name/size/rent/status flags).
- Must extend parser for full detail parity in `/v1/house/:world/:house_id` and richer list status.

## 6.6 Metadata/provenance envelope

TibiaData consistently adds:
- API version/release/commit details,
- timestamp,
- source URL(s),
- status object.

**Rubinot action:**
- Keep current `information` object and enrich with api details.
- Add parser/build metadata where useful.

---

## 7) Error model parity target

## Current in `rubinot-data`
- HTTP status + message string in `information.status`.

## Target parity
- Stable numeric domain `error_code` space (validation/upstream/parser/restriction).
- Predictable mapping:
  - `4xx` for client validation domain errors,
  - `5xx` for upstream/collector/parser infra class errors,
  - domain-specific code independent of HTTP code.

## Suggested initial code groups
- `10xxx`: name/path validation
- `11xxx`: world/town/house/category existence
- `12xxx`: restrictions/rate policy
- `20xxx`: upstream target state (maintenance/forbidden/redirect)
- `21xxx`: FlareSolverr transport/protocol errors
- `22xxx`: parser/schema errors

---

## 8) Observability parity (Rubinot-first)

While TibiaData’s historical implementation is not deeply OTel-first, Rubinot requires modern observability:

## Implemented
- traces: OTEL exporter + Gin middleware + scraper spans
- metrics endpoint + custom scrape/parse histograms/counters

## Planned expansion
- Route-level request metrics by status/handler
- Upstream status code metrics
- Challenge-page detection counter
- Parser fallback/failure counters by endpoint
- cache hit/miss metrics (when cache layer lands)
- exemplars linking latency histograms to Tempo traces (if enabled in stack)

---

## 9) Endpoint-by-endpoint operations matrix

## 9.1 TibiaData parity endpoints (core)

### Worlds
- `/v1/worlds` (list)
- `/v1/world/:name` (details)

Ops required:
- world name normalization
- world existence validation
- robust table parser for world details / online players

Status:
- world details implemented
- list pending

### Houses
- `/v1/houses/:world/:town` (overview)
- `/v1/house/:world/:house_id` (details)

Ops required:
- town/world validation
- overview fetch for both houses + guildhalls
- details fetch by house id and full status extraction

Status:
- overview implemented (subset fields)
- details pending

### Highscores
- `/v1/highscores/:world/:category/:vocation/:page`

Ops required:
- category/vocation/page validation
- restriction mode support
- pagination handling and source page consistency

Status: pending

### Guilds/Characters/Killstats/News

Ops required:
- route-level validation and parser contracts
- anti-bot resilience and parser fixtures

Status: pending

## 9.2 Rubinot-specific endpoints

- deaths, transfers, banishments, events schedule, auctions

Ops required:
- endpoint-specific parser contracts
- agreement on retained Rubinot-specific shape
- source provenance consistency and cache policy

Status: pending

---

## 10) Deployment + rollout notes

## What is pushed in code
- OTEL + metrics + houses + world endpoints are committed in `rubinot-data`.

## What still controls runtime behavior
- Cluster uses pinned image digest via GitOps manifests.
- New app functionality is only live after:
  1. image build+push,
  2. digest bump in GitOps,
  3. Argo sync rollout.

This is expected in GitOps-first flow.

---

## 11) Next milestones (ordered)

1. **House details endpoint parity**
   - implement `/v1/house/:world/:house_id`
   - include auction/rental/move/transfer fields (TibiaData-like richness)

2. **Validation package**
   - world/town/house/category/vocation validators
   - shared normalization utilities

3. **Error taxonomy**
   - numeric domain codes + consistent envelope

4. **Worlds list endpoint**
   - `/v1/worlds`

5. **Highscores endpoint with restriction mode**
   - include guardrails for heavy mode

6. **Parser fixture tests**
   - golden HTML fixtures per endpoint/edge-case

7. **Observability dashboard pack**
   - Mimir + Tempo panels for endpoint, parser, upstream and trace SLIs

---

## 12) Non-goals (for this stage)

- Full parity with all TibiaData v4 endpoints in one sprint.
- Full compatibility aliases for v1/v2/v3 style legacy routes.
- Shipping direct-to-production without digest-controlled GitOps rollout.

---

## 13) Practical checklist for each new endpoint port

For every endpoint, do all of the following before marking “done”:

1. Route + contract in `internal/api/router.go`
2. Input normalization + validation checks
3. Scraper fetch implementation (FlareSolverr path)
4. Parser implementation + edge-case handling
5. Error mapping (validation/upstream/parser)
6. Metrics + trace spans + relevant attributes
7. Source provenance in response envelope
8. Fixture tests for parser and regressions
9. GitOps/env updates (if needed)
10. Rollout verification (staging or cluster)

This keeps parity and reliability aligned while porting.

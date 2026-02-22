# Migration Plan: rubinot-live + rubinot-api -> rubinot-data model

## Guiding principles

1. No breaking consumer migration without transition window
2. Incremental replacement over big-bang rewrite
3. Keep scraping logic portable and testable
4. Measure each stage with explicit success criteria

## Stage 0 — Discovery & freeze map (1-2 weeks)

- Inventory all externally used endpoints in `rubinot-live` and `rubinot-api`
- Capture request volume and top consumers
- Classify endpoints: stable / flaky / expensive
- Define canonical schemas (entity-level contracts)

Deliverables:
- endpoint usage matrix
- contract baseline files
- migration compatibility policy

## Stage 1 — Contract-first foundation (1-2 weeks)

- Author OpenAPI for new `/v1` (contract-first)
- Define error model, stale model, and metadata fields
- Create schema validation and contract test harness

Deliverables:
- `openapi/v1.yaml` (future repo)
- response envelope standard
- compatibility mapping document

## Stage 2 — Data backbone alignment (2-3 weeks)

- Normalize persistence model for core entities
- Add provenance fields (`parser_version`, `fetched_at`, etc.)
- Introduce freshness policies per endpoint

Deliverables:
- canonical DB schema draft
- data retention policy
- stale and revalidation rules

## Stage 3 — Scraper hardening layer (2-4 weeks)

- Consolidate explorers/parsers into reusable modules
- Add parser golden tests against saved HTML fixtures
- Implement challenge detection and adaptive retries
- Add queue segmentation (critical vs heavy)

Deliverables:
- parser quality gates
- anti-bot strategy matrix
- failure taxonomy and retry matrix

## Stage 4 — Parallel run (2-3 weeks)

- Stand up `rubinot-data-api` in shadow mode
- Mirror selected requests and compare payload diffs
- Track freshness, latency, and mismatch rate

Success criteria:
- >= 95% schema parity on stable endpoints
- p95 latency target met
- scraper success-rate threshold achieved

## Stage 5 — Controlled cutover (1-2 weeks)

- Route a percentage of traffic to new API paths
- keep rollback switch at gateway level
- communicate deprecations for legacy paths

Success criteria:
- no critical regression for top consumers
- rollback tested and documented
- deprecation timeline published

## Stage 6 — Post-cutover cleanup

- retire duplicated collectors
- archive legacy endpoint paths
- lock versioning policy and release cadence

---

## What could go wrong (and mitigation)

1. **Upstream HTML breakage spikes**
   - Mitigation: fixture-based parser tests + canary workers + fallback to stale

2. **Cloudflare challenge escalation**
   - Mitigation: challenge metrics, controlled pacing, emergency restricted mode

3. **Schema drift during migration**
   - Mitigation: contract tests in CI and mirrored-response diffing

4. **Data inconsistency between old/new systems**
   - Mitigation: dual-write/dual-read verification window

5. **Cost explosion from scraping retries**
   - Mitigation: retry budgets and queue circuit breakers

6. **Consumer confusion due to endpoint changes**
   - Mitigation: compatibility aliasing + migration docs + staged deprecation

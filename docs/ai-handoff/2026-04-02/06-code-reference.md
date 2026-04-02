# Code Reference — Key Files Across All Repos

## rubinot-data (Go) — `/Users/gio/git/github/rubinot-lab/rubinot-data`

| File | Purpose |
|---|---|
| `cmd/server/main.go` | HTTP server entrypoint, OTEL init |
| `internal/api/router.go` | V1 routes, bootstrap validator, world discovery |
| `internal/api/router_v2.go` | V2 route registration |
| `internal/api/handlers_v2.go` | V2 handler implementations (all endpoints) |
| `internal/scraper/client.go` | V1 Client (FlareSolverr-based), session init |
| `internal/scraper/optimized_client.go` | V2 OptimizedClient wrapper |
| `internal/scraper/cached_fetcher.go` | Cache + singleflight dedup + CDP pool integration |
| `internal/scraper/cdp_pool.go` | CDP tab pool with auto-recovery |
| `internal/scraper/cdp.go` | CDP WebSocket client, Fetch/BatchFetch/Evaluate |
| `internal/scraper/v2_fetch.go` | V2 fetch functions (all endpoints) |
| `internal/scraper/highscores.go` | V1 highscores fetch + API response type |
| `internal/scraper/api_mappings.go` | World ID/name mappings, vocation mappings |
| `internal/validation/validator.go` | World/town/category/vocation validation |
| `internal/validation/highscores.go` | Highscore categories and vocations |

## rubinot-api (TypeScript) — `/Users/gio/git/github/rubinot-lab/rubinot-api`

### Jobs & Processors
| File | Purpose |
|---|---|
| `src/worker.ts` | BullMQ worker entrypoint, job routing, status recording |
| `src/jobs/scheduler.ts` | Job scheduling, tier config, stale scheduler cleanup |
| `src/jobs/queue.ts` | Queue definitions, `queueNameForJob()`, `JOB_QUEUE_OVERRIDES` |
| `src/jobs/processors/highscores.processor.ts` | Core: fetch → rank → detect changes → persist → rename candidates |
| `src/jobs/processors/highscore-cycle.processor.ts` | Tier-based cycle: processes due categories serially/parallel |
| `src/jobs/processors/guilds.processor.ts` | Guild roster sync, name change detection, membership events |
| `src/jobs/processors/world-online.processor.ts` | Online player tracking, level change detection |
| `src/jobs/processors/world-details.processor.ts` | Character profile enrichment for online players |
| `src/jobs/processors/character-enrichment.processor.ts` | Stale/offline character enrichment, 404 retry logic |
| `src/jobs/processors/name-change-detection.processor.ts` | Async candidate confirmation (fetches API for each candidate) |
| `src/jobs/processors/killstats-global.processor.ts` | Kill statistics change detection |

### Services
| File | Purpose |
|---|---|
| `src/services/rubinot-data-client.ts` | HTTP client to rubinot-data, all fetch methods, retry logic |
| `src/services/name-change-resolution.service.ts` | `applyResolvedNameChange()` — the merge function |
| `src/services/name-change-merge.service.ts` | `confirmAndMergeNameChange()` — async confirmation + merge |
| `src/services/clickhouse-client.ts` | ClickHouse insert functions for all event types |
| `src/services/worker-status-store.ts` | Redis-based job outcome tracking for status page |

### Repositories
| File | Purpose |
|---|---|
| `src/repositories/character.repository.ts` | Character CRUD, `reassignAndDelete`, `hardDeleteCharacter`, `incrementNotFoundCount` |
| `src/repositories/highscore.repository.ts` | Per-category highscore upserts, `rewriteCharacterName` |
| `src/repositories/character-rank.repository.ts` | Per-category rank upserts, guild rank updates |
| `src/repositories/character-analytics.repository.ts` | `buildSeries()` — rolling window XP rate computation |
| `src/repositories/killstats-analytics-read.repository.ts` | ClickHouse queries for creatures page |
| `src/repositories/analytics-read.repository.ts` | ClickHouse queries for character analytics |
| `src/repositories/guild.repository.ts` | Guild CRUD |
| `src/repositories/name-change-candidate.repository.ts` | Candidate queue management |

### Config & Utils
| File | Purpose |
|---|---|
| `src/config/env.ts` | Zod-validated environment schema (all env vars) |
| `src/utils/analytics-window.ts` | Resolution, bucket sizes, window computation |
| `src/constants/highscore-change-policy.ts` | Filter rules: monotonic, skill, rank cutoff |
| `src/constants/vocation-groups.ts` | Vocation affinity, noise thresholds |

### API Routes
| File | Purpose |
|---|---|
| `src/api/routes/killstats.routes.ts` | Creatures analytics, kill statistics |
| `src/api/routes/characters.routes.ts` | Character analytics, outfit/guild history |
| `src/api/routes/highscores.routes.ts` | Highscore listings |
| `src/api/routes/guilds.routes.ts` | Guild listings, details |
| `src/api/routes/status.routes.ts` | Status page API (queues, workers, DB, upstream) |

## rubinot-eve (Next.js) — `/Users/gio/git/github/rubinot-lab/rubinot-eve`

| File | Purpose |
|---|---|
| `src/lib/rubinot-api-client.ts` | API client, all types (CharacterAnalyticsResponse, CreatureAnalyticsResponse, etc.) |
| `src/components/character-analytics-chart.tsx` | Dual-axis XP/h chart with URL sync |
| `src/components/creatures-trend-chart.tsx` | Creatures kills/h chart |
| `src/app/characters/[name]/page.tsx` | Character page |
| `src/app/creatures/creatures-page-content.tsx` | Creatures analytics page |
| `src/app/guilds/guilds-page-content.tsx` | Guilds listing page |
| `src/app/status/page.tsx` | Status dashboard |

## GitOps — `/tmp/platform-gitops` (clone from `rubinot-lab/platform-gitops`)

| File | Purpose |
|---|---|
| `apps/rubinot/manifests/prod/rubinot-data.yaml` | rubinot-data deployment + env vars |
| `apps/rubinot-api/manifests/prod/rubinot-api-worker.yaml` | Heavy worker deployment |
| `apps/rubinot-api/manifests/prod/rubinot-api-worker-baseline.yaml` | Baseline worker |
| `apps/rubinot-api/manifests/prod/rubinot-api-worker-highscore-cycle.yaml` | Cycle worker |
| `apps/rubinot-api/manifests/prod/rubinot-api-scheduler.yaml` | Scheduler |
| `apps/rubinot-api/manifests/prod/postgres.yaml` | CNPG Cluster + Postgres params |

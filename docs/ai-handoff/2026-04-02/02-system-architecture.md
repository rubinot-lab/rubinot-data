# System Architecture

## Repositories

| Repo | Stack | Purpose |
|---|---|---|
| `rubinot-lab/rubinot-data` | Go 1.23, gin, CDP/FlareSolverr | Proxy that fetches Tibia game data from rubinot.com.br via CDP browser automation |
| `rubinot-lab/rubinot-api` | TypeScript, Fastify, BullMQ, Drizzle ORM | Data processing: jobs, change detection, rankings, analytics API |
| `rubinot-lab/rubinot-eve` | Next.js 15, Recharts, Tailwind | Frontend dashboard at eve.rubinot.dev |
| `rubinot-lab/platform-gitops` | K8s YAML | ArgoCD-managed deployment manifests |
| `cddlabs-casa/cluster-gitops` | Helm values | Infrastructure: Tempo, Mimir, Loki, Grafana, k8s-monitoring |

## Data Flow

```
rubinot.com.br (upstream game site)
  ↓ CDP fetch (Chrome DevTools Protocol via FlareSolverr sidecar)
rubinot-data (Go, 12 pods)
  ↓ HTTP API (/v1/*, /v2/*)
rubinot-api (TypeScript workers)
  ├── BullMQ job queues (4 queues)
  ├── PostgreSQL (CNPG, single instance on ceph-block)
  ├── ClickHouse (analytics events)
  └── Redis (job scheduling, worker status, caches)
  ↓ Fastify REST API
rubinot-eve (Next.js, SSR)
  ↓ Browser
eve.rubinot.dev
```

## Kubernetes Namespaces

| Namespace | Components |
|---|---|
| `rubinot` | rubinot-data (12 replicas), flaresolverr (standalone + sidecars) |
| `rubinot-api` | rubinot-api (2), scheduler (1), worker-baseline (1), worker-heavy (2), worker-enrichment (1), worker-highscore-cycle (1), postgres (1), redis (1) |
| `rubinot-eve` | rubinot-eve (2) |
| `rubinot-clickhouse` | ClickHouse cluster + keeper |
| `observability` | Grafana, Tempo, Loki, Mimir, Alloy, OpenCost |

## BullMQ Job Queues

| Queue | Worker | Jobs |
|---|---|---|
| `rubinot-baseline` | worker-baseline | discovery:worlds, discovery:categories, killstats:global, world:online, world-online-snapshot, world:info, highscores-fast:experience, highscores-fast:exp_today |
| `rubinot-heavy` | worker-heavy (×2) | deaths:global, guilds (×14 worlds), maintenance, auctions-current, auctions-enrichment, auction-history, boosted, transfers, news, newsticker, events, banishments:global, name-change-detection, highscore-retention |
| `rubinot-enrichment` | worker-enrichment | character-enrichment, world-details (×14 worlds) |
| `rubinot-highscore-cycle` | worker-highscore-cycle | highscore-cycle (Tier 2-4 categories) |

Job routing is configurable via `JOB_QUEUE_OVERRIDES` env var: `"highscores-fast=rubinot-baseline,world-details=rubinot-enrichment"`.

## Database Schema (PostgreSQL)

### Core Tables
- `character` (763K rows, 417MB) — central entity, 11 FK tables reference it
- `character_profile` (761K, 329MB) — 1:1 enrichment data from upstream API
- `character_name_change` — rename history (old_name → new_name, detection_method)
- `name_change_candidate` — pending/confirmed/rejected rename pairs

### Highscore Tables (per-category, 20 categories × 2 tables = 40 tables)
- `highscore_{slug}` (~90K rows each) — current snapshot per world
- `character_rank_{slug}` (~100-640K rows each) — world rank, global rank, vocation ranks

### Other Tables
- `guild`, `guild_member`, `guild_membership_event`, `guild_membership_interval`
- `death`, `transfer`, `house`, `auction`
- `kill_statistic` (current snapshot), `kill_statistic_change` (empty — changes go to CH)
- `boosted_creature`, `daily_player_snapshot`

### ClickHouse Tables (rubinot_analytics database)
- `highscore_change_events` — XP/skill changes per character, drives character analytics page
- `kill_statistic_change_events` — kill count changes per creature, drives creatures page
- `character_online_status_events` — online/offline transitions
- `character_leveling_events` — level up/down events

## Upstream Data Refresh Rates

| Endpoint | Refresh Rate | Evidence |
|---|---|---|
| `/api/killstats` | ~55 min | All 14 worlds update simultaneously |
| `/api/highscores` | ~10 min per world | Confirmed via `cachedAt` field in response |
| `/api/worlds/{name}` | Real-time | Players online list updates on each fetch |
| `/api/guilds/{name}` | Unknown | Guild member changes detected per fetch |
| `/api/characters/search` | Real-time | Character data + found_by_old_name signal |

## Current Performance

| Metric | Value |
|---|---|
| Experience highscore fetch | 2-5s (v2 batch CDP) |
| Experience highscore persist | 16-31s (P50=35.8s) |
| Experience total cycle | 93% under 60s |
| Guilds per world | ~20s, 0 failures |
| Killstats global | ~5s |
| Character enrichment | ~1s per batch |
| Postgres latency | 3.37ms, 99.9% cache hit |
| WAL write speed (ceph) | ~10.5 MB/s (bottleneck) |

## Key Environment Variables

### rubinot-api Worker Configuration
```
HIGHSCORE_USE_V2=true
HIGHSCORE_CYCLE_ENABLED=true
HIGHSCORE_CYCLE_CONCURRENCY=2
HIGHSCORE_TIER1_CATEGORIES=experience,exp_today
HIGHSCORE_TIER1_INTERVAL_MS=60000
HIGHSCORE_TIER2_INTERVAL_MS=300000 (totalbountypoints)
HIGHSCORE_TIER3_INTERVAL_MS=900000 (charmunlockpoints,linked_tasks,totalweeklytasks)
HIGHSCORE_TIER4_INTERVAL_MS=3600000 (everything else)
HIGHSCORE_ENTER_DROP_RANK_CUTOFF=900
GUILD_RANK_UPDATES_ENABLED=false
DATABASE_SYNC_COMMIT_OFF=true (cycle worker only)
CHARACTER_ENRICHMENT_MIN_LEVEL=50
WORKER_NAME_CHANGE_DETECTION_ENABLED=true
JOB_QUEUE_OVERRIDES=highscores-fast=rubinot-baseline,world-details=rubinot-enrichment
```

### Postgres Tuning
```
shared_buffers=3GB, effective_cache_size=5GB, work_mem=64MB
wal_level=replica, wal_compression=lz4, wal_buffers=64MB
max_wal_size=8GB, min_wal_size=2GB, commit_delay=10000
effective_io_concurrency=200, checkpoint_timeout=15min
```

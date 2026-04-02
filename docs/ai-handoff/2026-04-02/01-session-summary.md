# Session Summary — March 28 to April 2, 2026

## Overview

Multi-day session spanning rubinot-data (Go), rubinot-api (TypeScript/BullMQ), rubinot-eve (Next.js), platform-gitops (K8s manifests), cluster-gitops (infra Helm values), and cicd-templates (GitHub Actions). Started with a broken rubinot-data deployment, ended with a fully operational analytics platform with name change detection, optimized highscore processing, and smooth XP/h charts.

## Releases

| Repo | Version | Key Changes |
|---|---|---|
| rubinot-data | v2.3.7 → v2.3.9 | World ID mapping fix, V2FetchHighscoresBatch, CDP pool recovery |
| rubinot-api | v3.3.5 → v3.3.16 | v2 highscores, cycle processor, name changes, resilience fixes, analytics resolution |
| rubinot-eve | v2.10.2 → v2.10.10 | XP/h rolling rate, dual-axis charts, URL deep linking |
| cicd-templates | — | APP_VERSION/COMMIT auto-update, rubinot-lab sync |
| platform-gitops | — | OTEL fix, dedicated cycle worker, Postgres tuning, job routing |
| cluster-gitops | — | Tempo memcached, metrics generator, Grafana port fix |

## Major Accomplishments

### 1. rubinot-data Fixed (v2.3.7-v2.3.9)
- Root cause: world discovery failed because `/api/worlds` no longer returns `id` field
- Fix: fall back to `worldNameToID` mapping, updated stale world names (SerenianI-IV → Serenian, Etherian, Halorian, Divinian)
- Added `V2FetchHighscoresBatch` for parallel cross-world highscore fetch via CDP batch
- Fixed v2 `world=all` handler (was broken — sequential CDP loop caused broken pipe)
- Added CDP pool self-recovery when all tabs crash

### 2. Highscores v2 Migration (v3.3.5-v3.3.8)
- rubinot-api switched from v1 (paginated, 20 round-trips) to v2 (batch, 1 call) for highscores
- Feature flag `HIGHSCORE_USE_V2` for safe rollback
- ~28x fewer HTTP round-trips, fetch phase dropped from 12s to 2-3s

### 3. Dedicated Cycle Worker + Job Routing (v3.3.8-v3.3.13)
- New `rubinot-highscore-cycle` BullMQ queue with dedicated worker
- `JOB_QUEUE_OVERRIDES` env var for configurable job-to-queue routing
- Tier 1 categories (experience, exp_today) run as standalone fast jobs on baseline worker
- world-details moved to enrichment queue
- Eliminated job scheduling contention

### 4. Postgres Performance Tuning
- `wal_level`: logical → replica (20-30% less WAL)
- `wal_compression`: pglz → lz4
- `commit_delay`: 0 → 10ms (group commit)
- `shared_buffers`: 2GB → 3GB, `max_wal_size`: 4GB → 8GB
- `synchronous_commit=off` on cycle worker
- Guild rank updates disabled (`GUILD_RANK_UPDATES_ENABLED=false`)
- World-details interval: 5min → 30min
- Result: experience persist dropped from 90-540s to 16-31s

### 5. Observability Stack Fixed
- OTEL endpoint corrected across 6 services (was pointing to non-existent DNS)
- Tempo: enabled memcached, fixed ingester replication_factor, fixed Grafana datasource port
- Tempo metrics generator: enabled remote_write to Mimir
- Traces now flowing from rubinot-api and rubinot-data to Tempo

### 6. Name Change Pipeline (v3.3.14)
- Fixed `reassignAndDelete` FK violation (missing character_profile cleanup)
- Changed characterNameChange from delete to reassign (preserves history)
- Truncated 65M stale candidates, enabled name-change-detection job
- Added rank 900-1000 enter/drop filtering
- Added error handling for highscore name rewrite
- Result: 21 name changes confirmed, 0 guilds FK failures

### 7. Analytics Resolution Fixes (v3.3.15-v3.3.16)
- Killstats creatures page: 5-min → hourly buckets (matches upstream ~55 min refresh)
- Character XP: added rolling 1h experiencePerHour/rawExperiencePerHour
- Changed character analytics from 5-min to 10-min buckets (matches upstream ~10 min refresh)
- Skip unnecessary highscore upserts when no changes detected (~85% WAL reduction)

### 8. Frontend Improvements (v2.10.6-v2.10.10)
- XP/h and Raw XP/h rolling rate series (smooth curves)
- Dual-axis chart: bounty/weekly/linked tasks on right Y axis
- URL deep linking: `?chart=rawExperiencePerHour&chart2=bountyDelta`
- Chart series toggle synced with URL params

## Infrastructure Changes
- Node k8s-prod-01: drained, RAM reduced 64GB → 32GB, recovered
- Ceph OSD.3 (node 02): purged due to corrupted NVMe, NVMe needs investigation
- Ceph running on 2/3 OSDs (degraded but functional)
- Redis flushed multiple times during scheduler debugging

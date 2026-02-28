# Data Pipeline v3 — Implementation Handoff

## Prompt for Next Session

```
Read the design doc at docs/plans/2026-02-28-data-pipeline-v3-design.md and this handoff at docs/plans/2026-02-28-data-pipeline-v3-handoff.md, then create a full implementation plan following my planning rules. This is for rubinot-api (TypeScript, BullMQ, Drizzle ORM, PostgreSQL). The plan must include code snippets, decisions with reasoning, file references, and atomic commits.
```

## Context

### Repos
- **rubinot-data** (`/Users/gio/git/github/giovannirco/rubinot-data`) — Go API that scrapes Tibia data. Exposes REST endpoints. Already deployed at `https://data.rubinot.dev`.
- **rubinot-api** (`/Users/gio/git/github/giovannirco/rubinot-api`) — TypeScript worker + API. Branch `gio/new-rubinot-website-migration`, PR #2 open. This is where all implementation happens.

### Tech Stack (rubinot-api)
- Runtime: Node.js ESM, TypeScript
- Job queue: BullMQ (Redis-backed)
- ORM: Drizzle ORM with PostgreSQL
- Schema: `src/db/schema/*.ts` (Drizzle table definitions)
- Repositories: `src/repositories/*.ts` (DB access layer)
- Processors: `src/jobs/processors/*.ts` (job handlers)
- Worker: `src/worker.ts` (job dispatch + unified logging)
- Scheduler: `src/jobs/scheduler.ts` (BullMQ job scheduling)
- Types: `src/jobs/types.ts` (job name union + data types)
- Queue: `src/jobs/queue.ts` (BullMQ defaults + addJob helper)
- Docker: `docker-compose.yml` with postgres, redis, api, worker, schema-push

### Current State of the Highscore Problem
- **13 GB** in PostgreSQL after a few hours, 93% from `highscore` table
- 84,000 rows per snapshot cycle (14 worlds × 6,000 per-vocation entries)
- 20 categories, polling every 5-30 min
- Projected ~100-150 GB/day at current rates
- Indexes alone: 8.3 GB (bigger than data)

### How Highscores Currently Work
- `processHighscoresCrossWorld()` in `src/jobs/processors/highscores.processor.ts` fetches per-vocation from upstream
- Inserts into `highscore` table with `ON CONFLICT DO NOTHING` on `(world, categorySlug, snapshotId, rank)`
- Each poll creates a new `snapshot_id` (UUID), so every poll = 84K new rows
- The `player-snapshot` processor then computes deltas between consecutive snapshots

### Key Files to Read
- `src/jobs/processors/highscores.processor.ts` — current highscore processor (needs refactoring)
- `src/jobs/processors/killstats.processor.ts` — per-world killstats (already has global version)
- `src/jobs/processors/killstats-global.processor.ts` — global killstats processor
- `src/jobs/processors/guilds-global.processor.ts` — shows guild member reconciliation pattern
- `src/jobs/processors/maintenance.processor.ts` — currently log-only, needs DB writes
- `src/db/schema/highscore.ts` — current highscore schema
- `src/db/schema/kill-statistic.ts` + `src/db/schema/kill-statistic-snapshot.ts` — current kill stat schemas
- `src/repositories/highscore.repository.ts` — current highscore repo methods
- `src/repositories/kill-statistic.repository.ts` — kill stat repo
- `src/repositories/kill-statistic-snapshot.repository.ts` — kill stat snapshot repo
- `src/worker.ts` — unified logging pattern (processJob returns Record<string, unknown>)

### Decisions Made (with reasoning)

1. **Current-view + change-log for highscores** (not snapshots)
   - Why: 10-20x storage reduction. Only ~5-10% of entries change per poll.
   - `highscore` becomes UPSERT on `(world, category_slug, character_name_normalized)` — always latest
   - `highscore_change` logs only deltas (old_points, new_points, delta)

2. **Same pattern for kill_statistic**
   - Why: `kill_statistic_snapshot` at 3.2M rows and growing. Change-log enables time-series queries for creature kill graphs (kills/hour, boss kills/day).
   - `kill_statistic` already is current-view. Just add `kill_statistic_change`.

3. **One table per event type** (not unified events table)
   - Why: Better query performance per type, stronger typing, high-volume events (skill changes) don't pollute low-volume events.
   - Existing tables already serve as per-type event tables: `death`, `character_leveling`, `character_outfit_snapshot`, `transfer`, `character_identity`
   - New tables: `highscore_change`, `kill_statistic_change`, `guild_membership_event`, `maintenance_event`

4. **Write-ahead events** (not periodic materialization)
   - Why: Events available immediately for API consumers and rubinot-eve. Real-time feed.

5. **Maintenance as events** (not separate maintenance_window table)
   - Why: Low volume, duration computed at query time. Keeps it simple.

6. **Deprecate player_snapshot** — replaced by `highscore_change` time-series queries
7. **Deprecate kill_statistic_snapshot** — replaced by `kill_statistic_change`
8. **Deprecate boss_kill_snapshot** — replaced by `kill_statistic_change` + `boss_watch` JOIN
9. **Keep daily_player_snapshot** — aggregates online minutes, death counts (not derivable from highscore_change)
10. **Guild membership unified** — one `guild_membership_event` table for both joins and leaves (replaces `former_guild`)

### Unified Events API Design
- Route: `GET /v1/events?world=&character=&type=&since=&limit=`
- Handler builds UNION across relevant per-type tables based on filters
- If `type` filter provided, only query relevant table(s)

### Upstream API Reference (rubinot-data)
- Highscores: `GET /v1/highscores/:world/:category/:vocation/all` — returns `highscore_list[]` with `{rank, name, vocation, vocation_id, level, value}`
- Kill stats: `GET /v1/killstatistics/:world` (or `/all`) — returns entries with `{race, last_day_killed, last_day_players_killed, last_week_killed, last_week_players_killed}`
- Maintenance: `GET /v1/maintenance` — returns `{is_closed: bool, close_message: string}`

### Worker Logging Pattern
All processors return `Record<string, unknown>` which gets spread into the worker's generic info log:
```typescript
const metrics = await processJob(job);
logger.info({ job: job.name, world, durationMs, ...metrics }, `${job.name} completed`);
```
New processors should follow this pattern and return meaningful metrics.

### Data Volumes (from running instance)
- 30,558 characters (15,331 online)
- 14 active worlds
- 20 highscore categories
- 6,000 entries per world per category (per-vocation fetching)
- ~1,400 creature races per world in kill_statistic
- ~2,769 guilds, ~28,665 guild members

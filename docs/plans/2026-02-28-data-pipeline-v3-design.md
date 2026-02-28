# Data Pipeline v3: Change-Log Architecture

Status: approved
Date: 2026-02-28
Repos: rubinot-api, rubinot-data (API routes only)

## Problem

The current snapshot-based highscore storage consumes 13 GB (93% of DB) after a few hours of operation, projected at ~100 GB/day. 84,000 rows are inserted per snapshot cycle (14 worlds * 6,000 per-vocation entries * 20 categories). The pipeline also lacks event detection, maintenance tracking, and granular time-series data for character progression.

## Decisions

1. Highscore table becomes a current-view (UPSERT in place), changes logged to `highscore_change`
2. Kill statistic snapshots get the same treatment with `kill_statistic_change`
3. Events use one table per event type (not a single unified table)
4. Maintenance tracked via state-transition detection, stored as events
5. No Prometheus for per-character metrics (cardinality explosion with 30K+ characters)
6. `player_snapshot` table deprecated (replaced by `highscore_change` queries)
7. `daily_player_snapshot` kept for daily summaries (includes online minutes, death counts)
8. Unified events API queries across per-type tables with UNION

## Architecture

### Highscore Refactor

**Before**: INSERT 84K rows per snapshot with `snapshot_id`. Grows unbounded.
**After**: UPSERT on `(world, category_slug, character_name_normalized)`. Only store deltas.

**`highscore` table** (current view, ~1.2M rows fixed):
- Remove `snapshot_id`, `snapshot_ts` columns
- UPSERT key: `(world, category_slug, character_name_normalized)`
- Always reflects latest state
- Add `updated_at` timestamp

**`highscore_change` table** (append-only change log):
```sql
CREATE TABLE highscore_change (
  id            serial PRIMARY KEY,
  world         text NOT NULL,
  category_slug text NOT NULL,
  character_name text NOT NULL,
  character_name_normalized text NOT NULL,
  vocation      text,
  old_points    bigint,
  new_points    bigint NOT NULL,
  delta         bigint NOT NULL,
  old_rank      int,
  new_rank      int,
  detected_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_hsc_world_cat_time ON highscore_change (world, category_slug, detected_at);
CREATE INDEX idx_hsc_char_cat_time ON highscore_change (character_name_normalized, category_slug, detected_at);
```

**Processor flow**:
1. Fetch highscores from upstream (per-vocation, cross-world)
2. Read current `highscore` rows for that world/category
3. Compare: detect changes in points (delta != 0) or new entries
4. UPSERT `highscore` table (current view stays fresh)
5. INSERT only deltas into `highscore_change`
6. Emit `character_event` entries for skill milestones if needed

**Storage estimate**: ~5-10% of entries change per poll cycle. ~4K-8K change rows per cycle vs 84K snapshot rows. 10-20x reduction.

### Kill Statistic Refactor

Same pattern as highscores.

**`kill_statistic` table** (current view, ~20K rows, already is):
- Already uses UPSERT. No changes needed.

**`kill_statistic_change` table** (append-only):
```sql
CREATE TABLE kill_statistic_change (
  id            serial PRIMARY KEY,
  world         text NOT NULL,
  race          text NOT NULL,
  old_last_day_killed         int,
  new_last_day_killed         int NOT NULL,
  delta_last_day_killed       int NOT NULL,
  old_last_day_players_killed int,
  new_last_day_players_killed int NOT NULL,
  delta_last_day_players_killed int NOT NULL,
  old_last_week_killed        int,
  new_last_week_killed        int NOT NULL,
  delta_last_week_killed      int NOT NULL,
  old_last_week_players_killed int,
  new_last_week_players_killed int NOT NULL,
  delta_last_week_players_killed int NOT NULL,
  detected_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ksc_world_race_time ON kill_statistic_change (world, race, detected_at);
CREATE INDEX idx_ksc_race_time ON kill_statistic_change (race, detected_at);
```

**Boss queries** use JOIN with `boss_watch` at query time (no boss flag in change table):
```sql
SELECT ksc.race, date_trunc('hour', ksc.detected_at) as hour,
       sum(ksc.delta_last_day_killed) as kills
FROM kill_statistic_change ksc
JOIN boss_watch bw ON lower(ksc.race) = lower(bw.name) AND bw.enabled = true
GROUP BY ksc.race, hour ORDER BY hour;
```

### Event Tables (per-type)

**Already existing** (reused as-is):
- `death` — victim, killers, is_pvp, timestamp
- `character_leveling` — character, old_level, new_level, change_type
- `character_outfit_snapshot` — character, outfit fields, detected_at
- `transfer` — character, from_world, to_world, transfer_date
- `character_identity` — name changes, alt links

**New: `guild_membership_event`** (replaces `former_guild` + adds join tracking):
```sql
CREATE TABLE guild_membership_event (
  id              serial PRIMARY KEY,
  character_name  text NOT NULL,
  guild_name      text NOT NULL,
  world           text NOT NULL,
  event_type      text NOT NULL,  -- 'join' | 'leave'
  rank            text,
  detected_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_gme_char ON guild_membership_event (character_name, detected_at);
CREATE INDEX idx_gme_guild ON guild_membership_event (guild_name, world, detected_at);
```

**New: `maintenance_event`**:
```sql
CREATE TABLE maintenance_event (
  id            serial PRIMARY KEY,
  event_type    text NOT NULL,  -- 'start' | 'end'
  message       text,
  detected_at   timestamptz NOT NULL DEFAULT now()
);
```

Maintenance processor: store last known `is_closed` state in memory. On transition `false->true`: insert 'start'. On `true->false`: insert 'end'. Duration computed at query time.

### Unified Events API

`GET /v1/events?world=&character=&type=&since=&limit=`

Handler builds UNION query across relevant tables based on filters:
```sql
(SELECT 'death' as type, timestamp as detected_at, jsonb_build_object(...) as payload FROM death WHERE ...)
UNION ALL
(SELECT 'level_change', detected_at, ... FROM character_leveling WHERE ...)
UNION ALL
(SELECT 'skill_change', detected_at, ... FROM highscore_change WHERE ...)
UNION ALL
(SELECT 'guild_join', detected_at, ... FROM guild_membership_event WHERE event_type='join' AND ...)
UNION ALL
(SELECT 'guild_leave', detected_at, ... FROM guild_membership_event WHERE event_type='leave' AND ...)
ORDER BY detected_at DESC LIMIT $limit;
```

If `type` filter is provided, only query the relevant table(s).

### Time-Series Queries (examples)

**XP/hr for a character**:
```sql
SELECT date_trunc('hour', detected_at) as hour, sum(delta) as xp_gained
FROM highscore_change
WHERE character_name_normalized = 'kaiquerah lowprofile'
  AND category_slug = 'experience'
  AND detected_at >= now() - interval '24 hours'
GROUP BY hour ORDER BY hour;
```

**5-min XP rate (exp_today)**:
```sql
SELECT date_trunc('minute', detected_at) as ts, sum(delta) as xp_5min
FROM highscore_change
WHERE character_name_normalized = $1
  AND category_slug = 'exp_today'
  AND detected_at >= now() - interval '1 hour'
GROUP BY ts ORDER BY ts;
```

**Creature kills through the day**:
```sql
SELECT date_trunc('hour', detected_at) as hour, sum(delta_last_day_killed)
FROM kill_statistic_change
WHERE race = 'Dragon Lord' AND world = 'Serenian I'
  AND detected_at >= current_date
GROUP BY hour ORDER BY hour;
```

**Boss kills per hour (all bosses)**:
```sql
SELECT ksc.race, date_trunc('hour', ksc.detected_at), sum(ksc.delta_last_day_killed)
FROM kill_statistic_change ksc
JOIN boss_watch bw ON lower(ksc.race) = lower(bw.name) AND bw.enabled = true
WHERE ksc.detected_at >= current_date
GROUP BY ksc.race, date_trunc('hour', ksc.detected_at);
```

### Deprecations

- **`player_snapshot`** — replaced by `highscore_change` time-series queries
- **`kill_statistic_snapshot`** — replaced by `kill_statistic_change`
- **`boss_kill_snapshot`** — replaced by `kill_statistic_change` + `boss_watch` join
- **`highscore.snapshot_id` / `highscore.snapshot_ts`** — removed; table becomes current-view
- **`former_guild`** — replaced by `guild_membership_event` (type='leave')

### What Stays

- `daily_player_snapshot` — aggregates online minutes, death counts, level changes (data from multiple sources, not replaceable by highscore_change alone)
- `auction_snapshot` — bid tracking time-series for auctions (different domain)
- `character_online_status` — online/offline state changes (feeds daily snapshot computation)

## Future Features (out of scope)

- **Creature groups**: rubinot-eve feature for grouping creatures by spawn area. Queries `kill_statistic_change` with `WHERE race IN (...)`. No pipeline changes needed.
- **Rank milestones**: detect when a character enters top 10/50/100. Can be added to highscore processor later.
- **Tiered API plans**: rate limiting and API keys. Infrastructure concern, not pipeline.

## Retention Strategy

| Table | Retention | Rationale |
|---|---|---|
| `highscore` (current-view) | forever | Fixed ~1.2M rows |
| `highscore_change` | 90 days | Time-series queries rarely go beyond 90d |
| `kill_statistic` (current-view) | forever | Fixed ~20K rows |
| `kill_statistic_change` | 90 days | Same as highscore_change |
| `guild_membership_event` | forever | Low volume, historically interesting |
| `maintenance_event` | forever | Very low volume |
| `daily_player_snapshot` | forever | One row per character per day, manageable |
| `character_online_status` | 30 days | High volume, only needed for daily snapshot computation |
| `character_leveling` | forever | Low volume |

# Existing Analytics Features

## Character Analytics (eve.rubinot.dev/characters/{name})

### Data Sources
- **ClickHouse**: `highscore_change_events` — XP deltas from experience and exp_today highscore categories
- **ClickHouse**: `character_online_status_events` — online/offline transitions (every 15s scan)
- **ClickHouse**: `character_leveling_events` — level up/down events
- **PostgreSQL**: `character`, `character_profile`, `character_rank_*`

### API Endpoint
`GET /api/v1/characters/{name}/analytics?world={world}&range={range}`

### Series Available
| Series Key | Source Category | Type |
|---|---|---|
| `experiencePerHour` | experience | Rolling 1h sum of positive deltas |
| `rawExperiencePerHour` | exp_today | Rolling 1h sum of positive deltas |
| `experienceDelta` | experience | Raw per-bucket delta |
| `rawExperienceDelta` | exp_today | Raw per-bucket delta |
| `experienceCumulative` | experience | Running total |
| `rawExperienceCumulative` | exp_today | Running total |
| `bountyDelta` | totalbountypoints | Per-bucket delta |
| `weeklyDelta` | totalweeklytasks | Per-bucket delta |
| `linkedTasksDelta` | linked_tasks | Per-bucket delta |
| `bossPointsDelta` | bosstotalpoints | Per-bucket delta |
| `prestigePointsDelta` | prestigepoints | Per-bucket delta |

### Resolution
- Server-save / 24h: 10-minute buckets (was 5-min before v3.3.16)
- 7d / 14d / 30d: 1-hour buckets

### Summary Stats
- XP gained, raw XP gained, avg XP/h
- Online time, hunting time, session count
- Level delta, deaths
- Comparison with previous period (same duration)

### Additional Views
- Progression breakdown by category
- Daily online bar chart
- Online/hunting indicators on timeline

## Creatures Analytics (eve.rubinot.dev/creatures)

### Data Sources
- **ClickHouse**: `kill_statistic_change_events` — kill count deltas per creature per world
- **PostgreSQL**: `kill_statistic` — current snapshot

### API Endpoint
`GET /api/v1/killstats/creatures/analytics?names={names}&range={range}&world={world}&type={type}`

### Features
- Kills/h rolling rate per creature
- Players killed/h
- Cumulative kills/players killed
- World distribution (top 5 worlds per creature)
- Time-of-day heatmap (kills per hour, timezone-adjusted)
- Creature insights: window vs previous, trend (accelerating/decelerating), volatility
- Type filter: all, normal, bosses

### Resolution
- All ranges now use 1-hour buckets (changed in v3.3.15)

## Highscores (eve.rubinot.dev/highscores)

### Data Sources
- **PostgreSQL**: `highscore_{slug}` tables (20 categories)
- **PostgreSQL**: `character_rank_{slug}` tables

### Categories (20 total)
experience, magic, shielding, distance, sword, axe, club, fist, fishing, dromelevel, linked_tasks, exp_today, achievements, battlepass, charmunlockpoints, prestigepoints, totalweeklytasks, totalbountypoints, charmtotalpoints, bosstotalpoints

### Features
- Per-world rankings
- Global cross-world rankings
- Vocation-specific rankings
- Highscore change detection → ClickHouse events

## World Analytics (eve.rubinot.dev/worlds/{world})

### Data Sources
- **ClickHouse**: `character_online_status_events`
- **PostgreSQL**: `character`, `kill_statistic`

### Features
- Online player count over time
- Kill statistics tab
- World info (PvP type, record, creation date)
- Top XP gainers

## Status Page (eve.rubinot.dev/status)

### Data Sources
- **Redis**: Worker status store (job outcomes, durations, errors)
- **PostgreSQL**: pg_stat_database, pg_stat_user_tables
- **BullMQ**: Queue counts (waiting, active, delayed, completed, failed)

### Features
- Service versions (rubinot-eve, rubinot-api, rubinot-data)
- Queue health per queue
- Worker job table with status, interval, duration, success/failure counts, last error
- Database health: latency, cache hit, deadlocks, pool usage, connections
- Table health: top tables by dead rows
- Upstream API route stats

## Frontend Chart Component

File: `src/components/character-analytics-chart.tsx`

- Uses Recharts `ComposedChart` with `Line` components
- Dual Y-axis: left (primary XP series), right (bounty/weekly/etc.)
- URL param sync: `?chart=` for primary, `?chart2=` for secondary
- Online/hunting timeline indicators below chart
- Responsive, dark theme compatible

# Known Issues and Next Steps

## Active Issues

### Infrastructure
- **Ceph OSD.3 (node k8s-prod-02)**: NVMe disk corrupted/empty, unable to create new OSD. Ceph running on 2/3 OSDs (33% degraded). Need to investigate NVMe hardware or replace disk.
- **Postgres on ceph-block**: WAL write speed limited to ~10.5 MB/s. Moving to local NVMe would give 50-100x improvement. User considering dedicated DB machine.
- **GUILD_RANK_UPDATES_ENABLED=false**: Guild rank updates across 20 character_rank_* tables are disabled to reduce WAL contention. Needs re-enabling once Postgres is on faster storage.

### Data Quality
- **Upstream ~10 min cache**: rubinot.com.br caches highscores for ~10 min. Need to confirm during active hours (09:00-13:00 UTC) with live monitoring.
- **Upstream ~55 min killstats cache**: Confirmed. Creatures page now uses hourly buckets to match.
- **Experience and exp_today offset**: Events arrive ~2-3 min apart due to processor stagger. 10-min buckets handle this well.

### Character Enrichment
- **80% never enriched**: 613K of 763K characters never enriched. world-details covers online players; character-enrichment covers offline level 50+. Many low-level characters in highscores (fishing, dromelevel) remain unenriched.
- **Enrichment finds 0 eligible most runs**: world-details marks character_profile fresh, so character-enrichment's `NOT EXISTS character_profile.enriched_at < 7 days` filters them out.

### Intermittent Errors
- **HTTP 503 from rubinot.com.br**: Scattered across killstats, experience, maintenance, world:online. Transient — retries handle it. Not actionable without upstream changes.
- **Highscore cycle occasional timeout**: 900s timeout for full Tier 2-4 cycle. Some runs take longer due to WAL contention.

## Planned Work

### Guilds Feature Expansion
- Research Tibia fansites for guild analytics features
- Guild activity metrics (joins/leaves over time)
- Guild rankings by total level, member count, activity
- Guild wars tracking
- Member contribution analysis

### Postgres Migration to Local NVMe
- Dedicated machine with 1TB local SSD for Postgres
- Expected: persist phase drops from 16-31s to ~3-5s per category
- Re-enable guild rank updates after migration
- CNPG supports `walStorage` on separate volume

### Analytics Enhancements
- Per-character skill progression charts (magic, shielding, etc.)
- Boss kill tracking correlations
- Hunting efficiency metrics (XP per hour adjusted for level)
- Death analytics (death locations, killers, frequency)

### Name Change Pipeline
- Monitor false positive rate for rank 900-1000 filtering
- Consider lowering cutoff if too many legitimate changes are filtered
- Add `former_names` persistence to character table (currently only in upstream API)

## Monitoring Checklist

Check via `eve.rubinot.dev/status`:
- All worker jobs should be "healthy" with 0 failures in last 1h
- Experience should show comp1h > 50 (running every ~60s)
- Guilds should show comp1h = 14 (once per world per hour), 0 failures
- name-change-detection should show comp1h > 0
- Database pool usage should be low (< 50%)
- No deadlocks

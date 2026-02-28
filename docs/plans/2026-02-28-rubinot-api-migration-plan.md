# rubinot-api Migration Plan: Cross-World Endpoints

## Context

rubinot-data now supports cross-world `/all` endpoints. rubinot-api currently runs 14 per-world BullMQ jobs for each data type. This plan consolidates them into single global jobs, reducing upstream HTTP requests by ~93%.

## Architecture (Before → After)

```
BEFORE:
  scheduler → 14 BullMQ jobs (one per world) → worker → processor(world) → client.get*(world) → rubinot-data /v1/X/{world}
                                                                                                  ↓
                                                                                              14 HTTP calls/cycle

AFTER:
  scheduler → 1 BullMQ job (global) → worker → processor() → client.get*All() → rubinot-data /v1/X/all
                                                                                   ↓
                                                                               1 HTTP call/cycle
                                                                                   ↓
                                                                            loop results by world → DB upsert
```

## Request Reduction

| Job | Before | After | Savings |
|---|---|---|---|
| Killstats | 14 calls/min | 1 call/min | -13 req/min |
| Deaths | 14 calls/min | 1 call/min | -13 req/min |
| Banishments | 14 calls/hr | 1 call/hr | -13 req/hr |
| Guilds | 14 calls/hr | 1 call/hr | -13 req/hr |
| Boosted | 1 call/5m (unchanged) | + stores image_url | no change |
| Auctions | already global | no change | already optimal |

---

## Phase 1: Client Methods

### File: `src/services/rubinot-data-client.ts`

Add new methods alongside existing ones (don't remove old ones yet — keep for fallback).

### Decision: Response types

The `/all` endpoints return arrays wrapping the same per-world types. Define new array response types.

```typescript
// New cross-world response types
interface CrossWorldKillstatsResponse {
  killstatistics: KillStatisticsResult[];
  information: ApiInformation;
}

interface CrossWorldDeathsResponse {
  deaths: DeathsResult[];
  information: ApiInformation;
}

interface CrossWorldBanishmentsResponse {
  banishments: BanishmentsResult[];
  information: ApiInformation;
}

interface CrossWorldGuildsResponse {
  guilds: GuildsResult[];
  information: ApiInformation;
}

interface CrossWorldGuildsDetailsResponse {
  guilds: GuildsDetailsResult[];
  information: ApiInformation;
}
```

### New client methods

```typescript
async getKillStatisticsAll(): Promise<CrossWorldKillstatsResponse> {
  return this.get('/v1/killstatistics/all');
}

async getAllDeathsAllWorlds(): Promise<CrossWorldDeathsResponse> {
  return this.get('/v1/deaths/all/all');
}

async getAllBanishmentsAllWorlds(): Promise<CrossWorldBanishmentsResponse> {
  return this.get('/v1/banishments/all/all');
}

async getAllGuildsAllWorlds(): Promise<CrossWorldGuildsResponse> {
  return this.get('/v1/guilds/all/all');
}

async getAllGuildsDetailsAllWorlds(): Promise<CrossWorldGuildsDetailsResponse> {
  return this.get('/v1/guilds/all/all/details');
}
```

### Decision: Keep old per-world methods?

**Yes, keep them.** They serve as fallback and are useful for on-demand single-world refreshes. The global methods are only used by scheduled jobs.

---

## Phase 2: Killstats Global Processor

### Current flow (killstats.processor.ts)

```
receive job { world: "Elysian" }
→ client.getKillStatistics("Elysian")
→ parse response.killstatistics (single KillStatisticsResult)
→ for each entry: upsertKillStatistic(world, race, values)
→ for boss entries: insertBossSnapshot(world, race, values)
```

### New flow

```
receive job {} (no world param)
→ client.getKillStatisticsAll()
→ parse response.killstatistics (KillStatisticsResult[])
→ for each worldResult in array:
    → for each entry: upsertKillStatistic(worldResult.world, race, values)
    → for boss entries: insertBossSnapshot(worldResult.world, race, values)
```

### Code sketch

```typescript
// src/jobs/processors/killstats-global.processor.ts
export async function processKillstatsGlobal(
  client: RubinotDataClient,
  killStatisticRepo: KillStatisticRepository,
  bossKillSnapshotRepo: BossKillSnapshotRepository,
  bossWatchList: Set<string>,
) {
  const response = await client.getKillStatisticsAll();

  for (const worldResult of response.killstatistics) {
    const world = worldResult.world;

    for (const entry of worldResult.entries) {
      await killStatisticRepo.upsert({
        world,
        race: entry.race,
        lastDayPlayersKilled: entry.last_day_players_killed,
        lastDayKilled: entry.last_day_killed,
        lastWeekPlayersKilled: entry.last_week_players_killed,
        lastWeekKilled: entry.last_week_killed,
      });
    }

    // Boss snapshots — same logic, just world comes from the result
    const bossEntries = worldResult.entries.filter(e => bossWatchList.has(e.race));
    for (const boss of bossEntries) {
      await bossKillSnapshotRepo.insert({
        world,
        race: boss.race,
        lastDayKilled: boss.last_day_killed,
        lastDayPlayersKilled: boss.last_day_players_killed,
      });
    }
  }
}
```

### Decision: Transaction boundaries

**Upsert per world, not per entry.** If one world fails, others already committed. This is consistent with current behavior where per-world jobs are independent.

### Decision: Snapshot insert strategy

Currently snapshots are inserted every job run (every 60s). With global job, we still insert every 60s — same frequency, just all worlds in one pass. **No change needed to snapshot intervals.**

---

## Phase 3: Deaths Global Processor

### Current flow (deaths.processor.ts)

```
receive job { world: "Elysian" }
→ client.getAllDeaths("Elysian")
→ parse response.deaths (DeathsResult with entries[])
→ deathRepository.insertBatch(world, entries)
```

### New flow

```typescript
export async function processDeathsGlobal(
  client: RubinotDataClient,
  deathRepo: DeathRepository,
) {
  const response = await client.getAllDeathsAllWorlds();

  for (const worldResult of response.deaths) {
    await deathRepo.insertBatch(worldResult.world, worldResult.entries);
  }
}
```

### Decision: Filters (level, pvp)

The current per-world job does NOT pass level/pvp filters — it fetches all deaths. The cross-world endpoint also fetches all deaths by default. **No filter params needed.**

---

## Phase 4: Banishments Global Processor

### Current flow (banishments.processor.ts)

```
receive job { world: "Elysian" }
→ client.getAllBanishments("Elysian")
→ parse response.banishments (BanishmentsResult with entries[])
→ banishmentRepository.insertBatch(world, entries)
```

### New flow

```typescript
export async function processBanishmentsGlobal(
  client: RubinotDataClient,
  banishmentRepo: BanishmentRepository,
) {
  const response = await client.getAllBanishmentsAllWorlds();

  for (const worldResult of response.banishments) {
    await banishmentRepo.insertBatch(worldResult.world, worldResult.entries);
  }
}
```

---

## Phase 5: Guilds Global Processor

### Current flow (guilds.processor.ts)

```
receive job { world: "Elysian" }
→ client.getAllGuildsDetails("Elysian")
→ parse response.guilds (GuildsDetailsResult with guilds[])
→ for each guild: guildRepository.upsert(world, guild)
→ for each guild.members: guildMemberRepository.upsertBatch(guildId, members)
→ reconcile departed members
```

### New flow

```typescript
export async function processGuildsGlobal(
  client: RubinotDataClient,
  guildRepo: GuildRepository,
  guildMemberRepo: GuildMemberRepository,
) {
  const response = await client.getAllGuildsDetailsAllWorlds();

  for (const worldResult of response.guilds) {
    const world = worldResult.world;

    for (const guild of worldResult.guilds) {
      const guildRecord = await guildRepo.upsert({
        world,
        name: guild.name,
        logoUrl: guild.logo_url,
        description: guild.description,
        founded: guild.founded,
        active: guild.active,
      });

      if (guild.members) {
        await guildMemberRepo.upsertBatch(guildRecord.id, guild.members);
        await guildMemberRepo.reconcileDeparted(guildRecord.id, guild.members);
      }
    }
  }
}
```

### Decision: guilds/all vs guilds/all/details

**Use guilds/all/details** — rubinot-api already uses the details variant to get member lists. The plain /all only returns guild names without members.

---

## Phase 6: Boosted image_url

### Schema migration

```sql
ALTER TABLE boosted_creature
  ADD COLUMN boss_image_url TEXT,
  ADD COLUMN monster_image_url TEXT;
```

### Drizzle schema update (boosted-creature.ts)

```typescript
export const boostedCreature = pgTable('boosted_creature', {
  id: serial('id').primaryKey(),
  date: date('date').notNull(),
  bossId: integer('boss_id').notNull(),
  bossName: text('boss_name').notNull(),
  bossLooktype: integer('boss_looktype').notNull(),
  bossImageUrl: text('boss_image_url'),
  monsterId: integer('monster_id').notNull(),
  monsterName: text('monster_name').notNull(),
  monsterLooktype: integer('monster_looktype').notNull(),
  monsterImageUrl: text('monster_image_url'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
});
```

### Processor update (boosted.processor.ts)

```typescript
const response = await client.getBoosted();
const { boss, monster } = response.boosted;

await boostedRepo.upsertDaily({
  bossId: boss.id,
  bossName: boss.name,
  bossLooktype: boss.looktype,
  bossImageUrl: boss.image_url,       // NEW
  monsterId: monster.id,
  monsterName: monster.name,
  monsterLooktype: monster.looktype,
  monsterImageUrl: monster.image_url,  // NEW
});
```

### Decision: Store relative or absolute URL?

**Store relative path** (`/v1/outfit?type=945`). The consumer constructs the full URL using the rubinot-data base URL. This avoids hardcoding domain names and works across environments.

---

## Phase 7: Scheduler Refactor

### Current scheduler structure

```typescript
// Phase 1: Per-world, high frequency
syncWorldJobs(worlds) {
  for (const world of worlds) {
    addRepeatable('deaths', { world }, INTERVAL_DEATHS_MS)
    addRepeatable('killstats', { world }, INTERVAL_KILLSTATS_MS)
  }
}

// Phase 2: Per-world, low frequency
syncPhase2WorldJobs(worlds) {
  for (const world of worlds) {
    addRepeatable('guilds', { world }, INTERVAL_GUILDS_MS)
    addRepeatable('banishments', { world }, INTERVAL_BANISHMENTS_MS)
  }
}

// Phase 3: Global jobs
syncPhase3Jobs() {
  addRepeatable('auctions-current', {}, INTERVAL_AUCTIONS_MS)
  addRepeatable('boosted', {}, INTERVAL_BOOSTED_MS)
}
```

### New scheduler structure

```typescript
// Phase 1: Global, high frequency (replaces per-world)
syncGlobalHighFrequencyJobs() {
  addRepeatable('deaths:global', {}, INTERVAL_DEATHS_MS)
  addRepeatable('killstats:global', {}, INTERVAL_KILLSTATS_MS)
}

// Phase 2: Global, low frequency (replaces per-world)
syncGlobalLowFrequencyJobs() {
  addRepeatable('guilds:global', {}, INTERVAL_GUILDS_MS)
  addRepeatable('banishments:global', {}, INTERVAL_BANISHMENTS_MS)
}

// Phase 3: Global jobs (unchanged)
syncPhase3Jobs() {
  addRepeatable('auctions-current', {}, INTERVAL_AUCTIONS_MS)
  addRepeatable('boosted', {}, INTERVAL_BOOSTED_MS)
}
```

### Decision: Job naming

Use `:global` suffix to distinguish from per-world jobs during migration. Once stable, remove old per-world jobs.

### Decision: Migration strategy — gradual or big bang?

**Gradual.** Deploy with both per-world and global jobs running. Global jobs have a different name so they don't conflict. Monitor for 24h. Then disable per-world jobs via env flags (set intervals to 0). Then remove old code.

---

## Phase 8: Worker Handler

### Current worker.ts pattern

```typescript
switch (job.name) {
  case 'killstats':
    return processKillstats(job.data.world, ...deps);
  case 'deaths':
    return processDeaths(job.data.world, ...deps);
  // ...
}
```

### New additions

```typescript
switch (job.name) {
  // New global handlers
  case 'killstats:global':
    return processKillstatsGlobal(client, killStatRepo, bossSnapshotRepo, bossWatchList);
  case 'deaths:global':
    return processDeathsGlobal(client, deathRepo);
  case 'banishments:global':
    return processBanishmentsGlobal(client, banishmentRepo);
  case 'guilds:global':
    return processGuildsGlobal(client, guildRepo, guildMemberRepo);

  // Keep old per-world handlers during migration
  case 'killstats':
    return processKillstats(job.data.world, ...deps);
  // ...
}
```

---

## Performance Considerations

### Timeout handling

Cross-world endpoints take longer than per-world (especially deaths/banishments with pagination). Adjust BullMQ job timeouts:

```typescript
addRepeatable('deaths:global', {}, INTERVAL_DEATHS_MS, {
  timeout: 300_000, // 5 min (14 worlds × ~10s each + margin)
})

addRepeatable('guilds:global', {}, INTERVAL_GUILDS_MS, {
  timeout: 600_000, // 10 min (14 worlds × details fetching)
})
```

### Database write batching

Currently each world's data is written independently. With global jobs, consider batching writes:

```typescript
// Instead of 14 separate insertBatch calls:
for (const worldResult of response.deaths) {
  await deathRepo.insertBatch(worldResult.world, worldResult.entries);
}

// Could batch into one transaction (optional optimization):
await db.transaction(async (tx) => {
  for (const worldResult of response.deaths) {
    await deathRepo.insertBatch(worldResult.world, worldResult.entries, tx);
  }
});
```

**Decision:** Start without transaction wrapping (simpler, matches current behavior). Add transactions later if DB write performance becomes an issue.

### Rate limiting

With 93% fewer upstream requests, the `TokenBucket` rate limiter becomes less relevant for these jobs. But keep it — it still protects against burst scenarios and other endpoints.

---

## Commit Plan

| # | Commit | Files |
|---|---|---|
| 1 | `feat(client): add cross-world methods to rubinot-data client` | rubinot-data-client.ts |
| 2 | `feat(jobs): add killstats global processor` | killstats-global.processor.ts, worker.ts |
| 3 | `feat(jobs): add deaths global processor` | deaths-global.processor.ts, worker.ts |
| 4 | `feat(jobs): add banishments global processor` | banishments-global.processor.ts, worker.ts |
| 5 | `feat(jobs): add guilds global processor` | guilds-global.processor.ts, worker.ts |
| 6 | `feat(schema): add image_url to boosted_creature` | boosted-creature.ts, migration |
| 7 | `feat(jobs): store boosted image_url` | boosted.processor.ts, boosted.repository.ts |
| 8 | `feat(scheduler): add global job scheduling` | scheduler.ts |
| 9 | `test: verify global processors match per-world output` | *.test.ts |
| 10 | `chore: disable per-world jobs after validation` | scheduler.ts (set old intervals to 0) |
| 11 | `refactor: remove deprecated per-world processors` | cleanup old code |

### PR review checklist (final step)
- Verify all data is still being stored (compare row counts before/after)
- Verify API responses unchanged (E2E tests)
- Remove unnecessary comments
- Check no data loss for edge cases (world with 0 entries, partial failures)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Cross-world endpoint slower than 14 parallel per-world | Data freshness drops | Benchmark shows per-world parallel is bottlenecked at ~125s anyway; cross-world sequential is similar or faster |
| One world failure fails entire batch | All worlds miss an update cycle | Retry logic already in client; 60s interval means next cycle retries quickly |
| Timeout on large payloads (banishments/guilds) | Job fails, retries | Set generous timeouts (5-10 min); BullMQ auto-retries |
| Migration overlap (both old + new jobs writing) | Duplicate inserts | Repositories use upsert (ON CONFLICT), so duplicates are idempotent |

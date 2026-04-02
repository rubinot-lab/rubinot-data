# Guilds System — Current State and Code Locations

## Overview

The guilds system tracks guild membership across 14 worlds. It detects joins, leaves, and name changes by comparing the current roster with the previous snapshot. It runs on the heavy worker queue, once per hour per world.

## Data Flow

```
rubinot-data /v1/guilds/{world}/all/details
  ↓ HTTP (via rubinot-data-client.ts)
guilds.processor.ts
  ├── ensureSkeletons (create character rows for new members)
  ├── upsertDirectoryEntries (guild listing)
  ├── upsertGuild + upsertGuildMembers (per guild)
  ├── detectGuildNameChangeCandidates (leave+join pairs)
  ├── confirmGuildNameChangeCandidates (batch API check for found_by_old_name)
  │   ├── Confirmed: applyResolvedNameChange() → merge characters
  │   └── Unconfirmed: queue as pending candidate
  ├── guildMembershipEvent (join/leave events)
  ├── guildMembershipInterval (open/close intervals)
  ├── computeGuildRanks (DISABLED via GUILD_RANK_UPDATES_ENABLED=false)
  └── deleteMembersByNames (departed members)
```

## Key Files

### rubinot-api
| File | Purpose |
|---|---|
| `src/jobs/processors/guilds.processor.ts` | Main processor — roster comparison, event detection, name change inline confirmation |
| `src/repositories/guild.repository.ts` | CRUD for guild, guild_member, guild_directory |
| `src/repositories/guild-membership-event.repository.ts` | Join/leave event inserts |
| `src/repositories/guild-membership-interval.repository.ts` | Open/close membership intervals |
| `src/repositories/name-change-candidate.repository.ts` | Pending rename pair queue |
| `src/services/name-change-resolution.service.ts` | `applyResolvedNameChange()` — the merge function |
| `src/api/routes/guilds.routes.ts` | API endpoints for guild data |

### rubinot-eve
| File | Purpose |
|---|---|
| `src/app/guilds/page.tsx` | Guild listing page |
| `src/app/guilds/guilds-page-content.tsx` | Guild page content with search, filtering |

## Database Tables

### PostgreSQL
- `guild` — id, world, name, logo, description, founded, member_count, etc.
- `guild_member` — guild_id FK, character_name, rank, vocation, level, joined_at
- `guild_membership_event` — character_id, guild_id, event_type (join/leave), detected_at
- `guild_membership_interval` — character_id, guild_id, started_at, ended_at (null if current)

### ClickHouse
No guild-specific ClickHouse tables currently. Guild events go to PostgreSQL only.

## Name Change Detection in Guilds

The guilds processor has the most sophisticated inline name change detection:

1. **Detection**: `detectGuildNameChangeCandidates()` matches leave+join pairs by (vocation, level, rank) within the same guild
2. **Batch Confirmation**: `confirmGuildNameChangeCandidates()` fetches old names from upstream API via `client.postCharactersBatch(uniqueOldNames)`
3. **Inline Merge**: If `found_by_old_name=true` and resolved name matches, calls `applyResolvedNameChange()` immediately
4. **Queue Fallback**: Unconfirmed candidates stored in `name_change_candidate` table for async detection job

## Current Issues (as of April 2)

- **Guild rank updates disabled** (`GUILD_RANK_UPDATES_ENABLED=false`) — was causing massive WAL contention by updating all 20 character_rank_* tables per guild member
- **22 daily FK failures eliminated** — was from missing character_profile in reassignAndDelete
- **Name change inline confirmation working** — 21 confirmed in last few hours
- **Batch character fetch sometimes 503s** — falls back to queue-only mode

## Scheduling

- `INTERVAL_GUILDS_MS=3600000` (1 hour)
- 14 world-specific jobs: `guilds:Auroria`, `guilds:Belaria`, etc.
- Each takes ~20s average, up to 300s for large worlds (Elysian with 600+ guilds)
- Runs on `rubinot-heavy` queue

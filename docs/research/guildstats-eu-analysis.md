# GuildStats.eu -- Comprehensive Feature Analysis

**Date:** 2026-04-02
**Purpose:** Catalog every feature, page, data visualization, and analytical capability for Rubinot (rubinot-eve / rubinot-api) feature parity planning.

---

## Executive Summary

GuildStats.eu is the largest Tibia MMORPG statistics fansite, tracking 90 game worlds, ~3,000 guilds, ~215,000 guild members, and ~475,000 total characters. It recently underwent a full visual redesign (March 2026) with a modern dark theme and responsive layout. The site is built on PHP (server-rendered) with AJAX tabs for detail pages, uses Chart.js or similar for data visualizations, and scrapes data from Tibia.com's public APIs/pages.

**Key stats from homepage hero:**
- 15,341 players online (live, with sparkline chart)
- 2,997 guilds / 215,567 people in guilds
- 666B exp gained yesterday (approx level 3,420)

**Supported languages:** Polish, English, Dutch, Spanish, Portuguese

---

## Site Navigation Map

### Top Navigation Bar
- **News** -- `/index` (Tibia news feed + GuildStats announcements)
- **Forum** -- `/forum` (phpBB forum)
- **Contest** -- `/contest` (guild house decoration contest)

### General Section
- Fansite Item Owners -- `/fansite-items`
- Overall Statistics (Census) -- `/census` (per-world or global)
- Players Online Chart -- `/online-counter`
- Worlds Statistics -- `/worlds`
- World Merge History -- `/servers-merge`
- Monsters List -- `/monsters`
- Bosses List -- `/bosses`

#### Rookgaard Sub-section
- Best Characters (Rook) -- `/best-characters?rook=1`
- Time Online (Rook) -- `/time-online?rook=1`
- Oldest Players (Rook) -- `/oldest-players?rook=1`
- Bosses (Rook) -- `/bosses?rook=1`
- Guilds (Rook) -- `/guilds?type=rook`
- Experience (Rook) -- `/ranking-rook`
- Fist Fighting, Fishing, Achievements, Loyalty (filtered by rook vocation)

### Characters Section

#### Highscores (20 categories)
- `/ranking/experience`
- `/ranking/magic-level`
- `/ranking/shielding`
- `/ranking/distance-fighting`
- `/ranking/sword-fighting`
- `/ranking/club-fighting`
- `/ranking/axe-fighting`
- `/ranking/fist-fighting`
- `/ranking/fishing`
- `/ranking/achievements`
- `/ranking/loyalty`
- `/ranking/charm-points`
- `/ranking/goshnars-taint`
- `/ranking/drome-score`
- `/ranking/boss-points`
- `/ranking/bounty-points`
- `/ranking/weekly-tasks`
- `/ranking/titles`
- `/ranking/tibia-completed`
- `/ranking-points` (composite ranking)

#### Character Analytics
- Best Characters -- `/best-characters`
- Time Online -- `/time-online`
- Best on Worlds -- `/world-ranking`
- Oldest Accounts -- `/oldest-players`
- Account Birthday -- `/player-anniversary`
- Top Experience -- `/top-experience`
- Top Deaths -- `/deaths`
- Character Transfers -- `/world-transfer`
- Name Changes -- `/changed-names`
- Traded Characters -- `/traded-characters`
- Drome Statistics -- `/drome-leaderboard`
- Compare Characters -- `/compare-characters` (up to 5)

### Guilds Section
- Top Guilds (per world) -- `/guilds` and `/guilds/{WorldName}`
- Best Guilds (records) -- `/best-guilds`
- Wars -- `/wars`
- Guildhalls -- `/guildhalls`
- Guild Anniversary -- `/guild-anniversary`
- Awarded Guilds -- `/awarded-guilds`
- Compare Guilds -- `/compare-guilds`
- Manage Guild -- `/manage-guild` (authenticated)

### Tools Section (14 tools)
- Online Lists -- `/list`
- Birthday Notification -- `/options/notifications`
- Age Calculator (Tibian Years) -- `/tibian-years`
- Speed Calculator -- `/speed-calculator`
- Hits Calculator -- `/character-hits`
- Level Calculator -- `/character-level`
- Training Calculator -- `/training-calculator`
- Transfer Checker -- `/transferability-tool`
- Stamina Calculator -- `/stamina-calculator`
- Unjustified Kills (Frag Calculator) -- `/frag-calculator`
- Blacklist -- `/blacklist`
- Achievements Checker -- `/ach-checker`
- Forge Simulator -- `/forge-simulator`
- Level Prediction -- `/predict-level`

### Other Section
- Info/About -- `/info`
- Articles -- `/articles`
- Polls -- `/polls`
- Majestic Shield Creation -- `/creation`
- Activity System -- `/activity-system` (gamification for site participation)
- Screen of Week -- `/sow`
- Donations -- `/donation`
- Lottery -- `/lottery`
- Contact -- `/contact`

### Detail Pages (dynamic)
- Guild Detail -- `/guild/{GuildName}` (with tabs: General Info, Members, Time Online, History, Wars, Recruitment)
- Character Detail -- `/character/{CharName}` (with tabs: Character, History, Experience, Time Online, Highscores, Insomnia Entries, Drome, Deaths, Blacklist)
- Boss Detail -- `/bosses/{BossName}`
- Census per World -- `/census/{WorldName}`
- Deaths per World -- `/deaths/{WorldName}`
- Top Experience per World -- `/top-experience/{WorldName}`
- Ranking per World -- `/ranking/{WorldName}`
- Wars per World -- `/wars/{WorldName}`

---

## Guild Features (Primary Focus)

### 1. Guild List Page (`/guilds/{WorldName}`)

**World header shows:**
- World name, PvP type icon, region icon
- Total guilds count
- Total people in guilds
- Average people per guild
- War count (linked)
- Players currently online count (linked)
- BattleEye status

**Guild table columns:**
| Column | Description |
|--------|-------------|
| # | Rank position |
| Guild name | Linked to detail page, with Tibia.com link and "Compare guild" action |
| Logo | Guild logo image from Tibia.com |
| Members (Mbrs) | Total member count |
| Active players | Players active in last 2 months (online >= 15 min), shown as count and percentage |
| Min. exp | Minimum total experience gained by guild |
| Avg. exp | Average experience per player |
| Avg. lvl | Average level of members |
| Tot. lvl | Sum of all member levels |
| Pacc ratio | Premium-to-free account ratio |
| Sex ratio | Female-to-male ratio |
| Ach. points | Total achievement points |
| Avg. Ach. points | Average achievement points per player |
| Wars | Number of wars |
| Power of guild | Composite score: Sp + Tl + (Al x 10) where Sp=ranking points sum, Tl=levels sum, Al=average level |
| Created | Date guild was created |

### 2. Guild Detail Page (`/guild/{GuildName}`)

**Header section:**
- Guild logo (from Tibia.com)
- Guild name with Tibia.com external link
- "Compare guild" action
- "Manage guild" link (authenticated)
- Key stats cards: Members count, Total level, Average level, Power of guild (with formula tooltip)

**Tabbed interface with 6 tabs:**

#### Tab 1: General Info
- Created date + age ("19 years, 2 days")
- World (linked)
- Nationality
- Allied guilds count
- Update timestamp
- **Statistics block:**
  - Average exp per player
  - Minimum total exp
  - Max level / Min level
  - Achievement points (total)
  - Avg achievement points per player
- **Vocation Distribution** -- breakdown by all vocations (Knight, Elite Knight, Paladin, Royal Paladin, Druid, Elder Druid, Sorcerer, Master Sorcerer, Monk, Exalted Monk, No vocation) with counts and percentages
- **Vocations pie chart** (rendered as canvas/chart)
- **Male/Female pie chart**
- **Premium/Free pie chart**
- **Wars section:** Won / No winners / Lost counts
- **Time online stats:**
  - Sum of time online in current month
  - Average time online per player per week
- **Active players gauge:** percentage and count (e.g., "97 / 243 Members")

#### Tab 2: Members
- Full member list table with:
  - Rank within guild
  - Character name (linked to character page)
  - Vocation
  - Level
  - Guild rank/title
  - Account status (Premium/Free)
  - Last login

#### Tab 3: Time Online
- Time online statistics for guild members
- Aggregated charts showing online activity patterns

#### Tab 4: History
- Member join/leave history
- Level changes over time
- Historical membership count

#### Tab 5: Wars
- War history for the guild
- Opponent, score, kill limit, duration, fee, outcome

#### Tab 6: Recruitment
- Guild recruitment settings (managed by guild leader)

### 3. Best Guilds Page (`/best-guilds`)

Cross-world record holders:
| Record | Example |
|--------|---------|
| The biggest guild | Acord Os -- 4,036 members (Quintera) |
| The oldest guild | Soldiers of Justice -- created 18-02-2002 (Antica) |
| Highest avg exp per player | Refugees -- 70,627,230,155 (Celesta) |
| Highest avg level per player | Refugees -- avg lvl 1,497 (Celesta) |
| Highest total level sum | Acord Os -- 1,045,338 (Quintera) |
| Highest total achievement points | Acord Os -- 561,943 (Quintera) |
| Highest avg achievement points | Hambrientos De Menera -- 744 (Menera) |
| Best rookgaard guild | Rookgaard Movement -- avg lvl 37 (Vunira) |
| Best female guild | Artemis -- avg lvl 126 (Celebra) |

### 4. Guild Wars Page (`/wars`)

**Features:**
- "Most wars happened on" -- world ranking by war count (e.g., Vunira 834, Thyria 175)
- **Current wars table:** Guild 1 vs Guild 2, Score, Kill Limit, Duration, Fee, World, Compare guilds link
- **Recently ended wars table:** Same columns plus End date
- Filterable by world (`/wars/{WorldName}`)

### 5. Guild Anniversary (`/guild-anniversary`)
- Lists guilds celebrating their founding anniversary

### 6. Awarded Guilds (`/awarded-guilds`)
- Lists guilds that have received fansite awards/recognition

### 7. Compare Guilds (`/compare-guilds`)
- Side-by-side comparison of two guilds
- Input: two guild names
- Compares all metrics (members, levels, experience, activity, etc.)

### 8. Guildhalls (`/guildhalls`)
- Guildhall ownership tracking
- Shows which guilds own which houses

---

## Character Features

### 1. Character Detail Page (`/character/{CharName}`)

**Header section:**
- Vocation icon and name
- Character name (h1)
- Tibia.com external link
- "Compare character" action
- "Link character" (account linking for management)
- **Level card** with level-up notification (e.g., "Level up! from lvl 3007", "Min exp gained: 127,482,884")
- **Guild affiliation** with join date
- **Sharing exp range** (party level range for exp sharing)

**8 tabs:**

#### Tab 1: Character (Profile)
- Nationality
- Sex (with icon)
- Vocation
- Achievement points
- Titles count (linked to ranking)
- Account created date (with age: "19 years ago")
- World (linked)
- Residence (city)
- Last login timestamp
- Account status (Premium/Free)
- **Tibian age** (game-time age with real-time conversion: "1h real = 1 day in Tibia")
- **Tibia Completed** score (formula: boss points + charm points + (achievements x 10) + (titles x 100)) / max available)

#### Tab 2: History
- Level history over time
- Guild membership changes
- Name changes
- World transfers

#### Tab 3: Experience
- **Best recorded day** highlight with date and amount
- **Stone of Insight** detection (bonus exp item)
- **Current rank** on vocation leaderboard
- **"Experience over time" chart** -- line chart showing exp progression
- **"Daily exp gained vs time online" chart** -- correlation visualization
- **Level prediction** with target level input
- **Monthly experience tables** (3 months shown: current, previous, 2 months ago):
  - Date
  - Exp change (with +/- sign)
  - Vocation rank
  - Level (with level-up indicators like "+1")
  - Total experience
  - Time online that day
  - Average exp/hour
  - **Double XP Event** markers on relevant dates
  - Monthly totals row

#### Tab 4: Time Online
- Summary stats: Last month, Current month, Current week, plus per-day breakdown (Mon-Sun)
- **"Time online - 30 days" bar chart**
- **"Time online - Timeline (30 days)" heatmap/timeline** showing online/offline periods with Double XP event markers
- Timezone selector (30+ timezones: Berlin, London, Warsaw, Sao Paulo, New York, Tokyo, etc.)

#### Tab 5: Highscores
- All skill rankings for the character on their world:
  - Experience, Achievements, Magic Level, Loyalty, Charm Points, Goshnars Taint, Drome Score, Boss Points, Bounty Points, Weekly Tasks
- Each shows: Skill value, 30-day change indicator, World rank position (linked)
- **Ranking points** composite score

#### Tab 6: Insomnia Entries
- Insomnia mini-game participation records

#### Tab 7: Drome
- Drome (arena) participation and scores

#### Tab 8: Deaths
- **Summary stats:**
  - Total number of deaths
  - Total experience lost
  - "Level without dying" calculation
  - Killed by players count
- **Deaths table:**
  - Date/time
  - Killed by (monster or player name, with PvP indicator)
  - Level at time of death
  - Approximate exp lost (calculated with blessings + promotion)

### 2. Rankings/Highscores (`/ranking/{skill}`)

**20 ranking categories** (see navigation map above)

**Features per ranking page:**
- **World filter** dropdown (all worlds or specific)
- **Advanced filters:** Vocation, Gender, City, Account type, PvP Type, Location (region), BattleEye
- **"World distribution in top 500"** -- bubble/tag cloud showing which worlds dominate each ranking
- **Ranking table:**
  - Rank #
  - Character name (with Tibia.com link + Compare character action)
  - Skill value
  - Daily exp gain
  - Vocation abbreviation (ED, MS, EK, RP, etc.)
  - World (with PvP type and BattleEye icons)

### 3. Top Experience (`/top-experience`)
- Daily experience gain leaders across all worlds
- Filterable by world

### 4. Time Online (`/time-online`)
- Players sorted by online time
- Shows current month and weekly statistics

### 5. Best Characters (`/best-characters`)
- Cross-world records (highest level, most experience, etc.)

### 6. Oldest Players (`/oldest-players`)
- Characters sorted by account creation date
- Shows account age

### 7. Deaths Page (`/deaths/{WorldName}`)
- Recent death log per world
- Shows character, level, killer, exp lost

### 8. Compare Characters (`/compare-characters`)
- Up to 5 characters side-by-side
- Input: 5 name fields
- Compares all metrics

### 9. Character Transfers (`/world-transfer`)
- Log of characters that transferred between worlds
- Shows: character name, from world, to world, date

### 10. Name Changes (`/changed-names`)
- Log of character name changes
- Shows: old name, new name, world, date

### 11. Traded Characters (`/traded-characters`)
- Characters sold/bought on the Tibia bazaar
- Tracking of ownership changes

### 12. Player Anniversary (`/player-anniversary`)
- Characters celebrating account creation anniversaries

---

## World/Server Features

### 1. Worlds Statistics (`/worlds`)

**Summary header:** Total worlds (90), Total guilds (2,997), People in guilds (214,965)

**World table columns:**
| Column | Description |
|--------|-------------|
| # | Rank |
| World name | Linked to guild list |
| Updated | Database update timestamp |
| Location | Region icon (Europe/North America/South America) |
| Avg people in guild | Average guild size |
| Guilds | Guild count |
| Achievement points | Total achievement points on world |
| Sex ratio | Female-to-male ratio |
| People in guilds | Total guild members |
| Wars | War count (linked to wars page) |
| Type | PvP type icon (Open PvP, Optional PvP, Hardcore PvP, Retro Open PvP, Retro Hardcore PvP) |
| BattleEye | Protection status (from initial / since date) |
| Record online | All-time record with date |
| Created | World creation date |

### 2. Online Players Counter (`/online-counter`)

**Features:**
- World filter dropdown
- **Timezone selector** (30+ timezones)
- **Past 24 hours** card: Max players (with time), Min players (with time), Average players
- **Past week** card: Same metrics
- **Line chart** showing player count over time
- Supports Double XP Event annotation
- Tracks 15-minute intervals

### 3. Census (`/census` and `/census/{WorldName}`)

**Data tables with percentages:**
- **Cities in Tibia** -- residence distribution (e.g., Thais 56.1%, Venore 9.0%)
- **Vocations in Tibia** -- all vocation types with counts (Knight 22.1%, Elite Knight 13.6%, etc.)
- **Vocations (Free + Premium combined)** -- grouped vocation classes
- **Gender Distribution** -- Male 79.7%, Female 20.3%
- **Premium vs Free Accounts** -- Premium 54.2%, Free 45.8%
- **Guild Nationality** -- BR 37.1%, PL 18.2%, EN 12.9%, etc.
- Per-world breakdowns available

### 4. World Merge History (`/servers-merge`)
- Historical record of merged game worlds

---

## Boss Features (`/bosses`)

### Boss List Page

**Features:**
- World filter dropdown
- Boss name search/filter dropdown (500+ bosses)
- **Boosted Boss** highlight card showing current boosted boss with image and time period
- **Featured bosses** section showing major raid bosses with "last seen" timestamps (e.g., "Ferumbras -- Seen 4 days ago", "Gaz'haragoth -- SEEN YESTERDAY")
- Login prompt to follow bosses (notification feature)

**Filter system:**
- Boss type: Bane, Archfoe, Nemesis (with icons)
- Fight mode: Solo, Team
- Appearance: Raid, Event, Quest
- Min level: 100+, 200+, 300+, 400+, 500+

**Boss table columns:**
| Column | Description |
|--------|-------------|
| # | Rank |
| Boss name | Linked to detail page |
| Watch movie | YouTube link (some bosses) |
| Image | Boss sprite |
| Yesterday killed | Kill count yesterday |
| Yesterday players | Players who killed it yesterday |
| Overall killed | All-time kill count |
| Overall players | All-time player count |
| Last seen | Date last killed |
| Introduced | Date boss was added to game |
| Type | Bane/Archfoe/Nemesis icon |

### Boss Detail Page (`/bosses/{BossName}`)
- Per-world kill statistics
- Kill history chart
- Last seen per world

---

## Drome Features (`/drome-leaderboard`)

- **Rotation filter** (numbered rotations with dates, e.g., "#123 (01 Apr, 2026)")
- **All ranking page filters** (world, vocation, level range, gender, city, account type, PvP type, location, BattleEye)
- **World distribution in top 500** tag cloud
- **Charts:**
  - "Average drome level by vocation"
  - "Average efficiency by vocation"
- **Leaderboard table** with drome scores

---

## Tools & Calculators

### 1. Speed Calculator (`/speed-calculator`)
- Input: Character level
- Equipment selectors with item images:
  - Boots (Zaoan Shoes +10, Draken Boots +30, Boots of Haste +40, etc.)
  - Spells (Haste, Strong Haste, Charge, Swift Foot) -- filtered by vocation
  - Armor (Zaoan Armor +20, Elite Draken Mail +20, Prismatic Armor +30)
  - Other items (Mount +20, Time Ring +60, etc.)
- Output: Calculated speed, equivalent level speed

### 2. Forge Simulator (`/forge-simulator`)
- Classification tier selection (1-4)
- Desired tier target
- Item cost and sliver cost inputs
- Success rate with core modifier (65% base, 50% with core)
- Tier loss rate with core modifier
- Transfer enable option
- **Output: Average results** -- gold tax, estimated costs

### 3. Level Prediction (`/predict-level`)
- Input: Current level, Target level, Average daily experience
- Output: Time needed, Target date, Levels needed, Daily average exp
- Experience chart showing progression curve

### 4. Hits Calculator (`/character-hits`)
- Damage calculation based on character attributes

### 5. Level Calculator (`/character-level`)
- Experience-to-level conversion

### 6. Training Calculator (`/training-calculator`)
- Skill training time estimates

### 7. Transfer Checker (`/transferability-tool`)
- Check if a character can transfer between worlds

### 8. Stamina Calculator (`/stamina-calculator`)
- Stamina regeneration time tracking

### 9. Frag Calculator (`/frag-calculator`)
- Unjustified kill counter and penalty calculator

### 10. Achievements Checker (`/ach-checker`)
- Achievement completion tracking

### 11. Tibian Years Calculator (`/tibian-years`)
- Real-time to Tibian-time conversion

### 12. Blacklist (`/blacklist`)
- Player blacklist/blocklist management

### 13. Online Lists (`/list`)
- Custom online player list tracking

### 14. Signature Generator (`/signature`)
- Dynamic image signature for forum/Discord showing character stats

---

## Data Visualizations & UX Patterns

### Chart Types Used

1. **Line Charts** -- Experience over time (character page), Player online count over time
2. **Bar Charts** -- Time online per day (30-day view), Daily exp gained
3. **Pie/Donut Charts** -- Vocation distribution, Male/Female ratio, Premium/Free ratio
4. **Timeline/Heatmap** -- Time online timeline showing hourly online/offline blocks
5. **Sparkline Charts** -- Mini charts in homepage stat cards (players online, exp yesterday)
6. **Tag Cloud/Bubble** -- World distribution in top 500 (sized by count)
7. **Gauge/Progress** -- Active players percentage in guild
8. **Correlation Charts** -- Daily exp gained vs time online

### UX Patterns

- **Dark theme** (recently redesigned March 2026) -- modern CSS, dark background, good contrast
- **Responsive layout** -- works on mobile, tablet, desktop
- **Tabbed interfaces** -- Guild and character detail pages use AJAX tabs to avoid full page reloads
- **Advanced filter system** -- Consistent filter pattern across ranking pages: World, Vocation, Gender, City, Account type, PvP type, Location, BattleEye
- **Compare feature** -- Characters (up to 5) and guilds (2) can be compared side-by-side
- **Persistent compare widget** -- "Compare characters" and "Compare guilds" floating widgets at page bottom
- **Tooltip-rich tables** -- Column headers have detailed tooltips explaining metrics
- **External links** -- Every character/guild links to official Tibia.com page
- **Event markers** -- Double XP events clearly marked on experience charts/tables
- **Timezone support** -- Extensive timezone selector for online time features (30+ zones)
- **Level-up notifications** -- Banner when character gains level with exp details
- **"Report bug" button** -- Present on every page
- **Search** -- Global search bar on homepage for guilds or characters
- **Breadcrumb navigation** -- Path shown (e.g., /Guild/Antica/Black Dragons)
- **Multi-language** -- 5 languages with URL parameter switching (`?lang=en`)
- **Rashid tracker** -- Shows current NPC Rashid location in header
- **Server save countdown** -- Real-time countdown in header
- **Tibia Drome countdown** -- Next Drome event countdown
- **"Did you know" trivia** -- Random Tibia facts on homepage
- **Screen of the Week** -- Community screenshot feature

---

## Unique/Notable Features

1. **Guild Power Formula** -- `Sp + Tl + (Al x 10)` composite metric for ranking guilds beyond just member count
2. **Active Player Percentage** -- Tracks which guild members are actually active (online >= 15 min in last 2 months)
3. **"Level Without Dying" Calculation** -- Shows what level a character would be if they never died (accounting for all exp lost to deaths)
4. **Stone of Insight Detection** -- Automatically detects when bonus exp items were used on top experience days
5. **Exp/Hour Calculation** -- Correlates time online with exp gained to calculate efficiency
6. **Tibia Completed Score** -- Custom composite score: (boss points + charm points + (achievements x 10) + (titles x 100)) / max
7. **Boss Tracking with "Last Seen"** -- Per-world boss kill tracking showing when each boss was last killed
8. **Drome Leaderboard** -- Arena mini-game statistics with rotation tracking
9. **Activity/Gamification System** -- Users earn points for site participation (posting, articles, donations, etc.) and exchange for in-game rewards
10. **Forge Simulator** -- Monte Carlo-style simulation of Tibia's item upgrade system
11. **Fansite Item Lottery** -- Tracked fansite item distribution
12. **Guild History/Member Tracking** -- Historical join/leave records for guilds
13. **World Distribution in Top Rankings** -- Shows which servers dominate each skill category

---

## URL Patterns

```
/ (homepage)
/guild/{GuildName}           -- guild detail (spaces as +)
/guilds/{WorldName}          -- guild list per world
/character/{CharName}        -- character detail (spaces as +)
/ranking/{skill}             -- highscore by skill type
/ranking/{WorldName}         -- highscore per world
/ranking/{skill}/{WorldName} -- highscore by skill per world
/bosses                      -- all bosses
/bosses/{BossName}           -- boss detail
/bosses?world={World}        -- bosses per world
/bosses?tag={type}           -- bosses by type (bane/archfoe/nemesis/solo/team/raid/event/quest/100/200/300/400/500)
/census/{WorldName}          -- census per world
/deaths/{WorldName}          -- deaths per world
/top-experience/{WorldName}  -- top exp per world
/wars/{WorldName}            -- wars per world
/compare-characters          -- compare up to 5 characters
/compare-guilds              -- compare 2 guilds
/drome-leaderboard           -- drome scores
/drome-leaderboard/{World}   -- drome per world
/sow/{id}                    -- screen of week entry

# AJAX tab endpoints (character detail):
/include/character/tab.php?nick={name}&tab=experience
/include/character/tab.php?nick={name}&tab=deaths
/include/character/tab.php?nick={name}&tab=timeonline
/include/character/tab.php?nick={name}&tab=highscore

# Query parameters:
?lang={pl|en|nl|es|pt}       -- language
?rook=1                      -- rookgaard filter
?voc={0-5}                   -- vocation filter
?world={WorldName}           -- world filter
?type=rook                   -- guild type
?rotation={0|N}              -- drome rotation
```

---

## Data Requirements for Rubinot

To replicate GuildStats.eu features, rubinot-api would need to collect and serve the following data:

### Core Entities

| Entity | Data Points | Update Frequency |
|--------|-------------|-----------------|
| **Character** | name, level, vocation, sex, world, residence, account_created, last_login, account_status, guild_id, guild_rank, achievement_points, titles, experience | Daily (from game API or scraping) |
| **Character Daily Snapshot** | character_id, date, level, experience, exp_change, time_online, ranking positions for each skill | Daily |
| **Character Death** | character_id, date, killed_by, killer_type (pve/pvp), level_at_death, exp_lost | Near real-time |
| **Guild** | name, world, created_date, logo_url, nationality, allied_guilds | Daily |
| **Guild Member** | guild_id, character_id, rank, join_date, leave_date | Daily |
| **Guild Daily Snapshot** | guild_id, date, member_count, total_level, avg_level, total_exp, active_players | Daily |
| **World** | name, region, pvp_type, battleeye_status, created_date, online_record | Static + periodic |
| **World Online Snapshot** | world_id, timestamp, player_count | Every 15 minutes |
| **Boss** | name, image_url, type (bane/archfoe/nemesis), fight_mode, min_level, introduced_date | Static |
| **Boss Kill** | boss_id, world_id, date, kill_count, player_count | Daily |
| **War** | guild1_id, guild2_id, score, kill_limit, duration, fee, start_date, end_date, world | Event-driven |
| **Character Transfer** | character_id, from_world, to_world, date | Event-driven |
| **Character Name Change** | character_id, old_name, new_name, date | Event-driven |
| **Guildhall** | name, world, size, rent, owner_guild | Daily |

### Derived/Calculated Data

- **Guild Power Score:** `ranking_points_sum + total_levels + (avg_level x 10)`
- **Active Player %:** Members online >= 15 min in last 2 months / total members
- **Exp/Hour:** Daily exp change / daily time online
- **Level Without Dying:** Theoretical level if 0 exp was ever lost
- **Tibia Completed:** `(boss_points + charm_points + (achievements x 10) + (titles x 100)) / max_available`
- **Ranking Points:** Composite score from positions across all highscore categories

### Storage Estimates

- ~475,000 characters with daily snapshots = ~170M rows/year for character snapshots
- ~90 worlds with 15-min online counts = ~3.1M rows/year for online tracking
- ~500 bosses x 90 worlds = ~16.4M rows/year for boss kills

---

## Implementation Priority Recommendations

### Phase 1: Core Data Foundation
1. **Character lookup and profiles** -- Basic character page with level, vocation, guild
2. **Guild list and detail pages** -- Guild table with key metrics, detail page with general info tab
3. **World list** -- Server browser with basic stats
4. **Highscores/Rankings** -- Experience ranking with world/vocation filters

### Phase 2: Analytics & Tracking
5. **Experience tracking** -- Daily snapshots, exp charts, level history
6. **Time online tracking** -- Online time recording and heatmaps
7. **Death logging** -- Death feed per world, character death history
8. **Boss tracking** -- Kill counts, last seen, per-world tracking
9. **Online player counter** -- Real-time player count with charts

### Phase 3: Guild Deep Features
10. **Guild member list with details** -- Full member table with levels, vocations
11. **Guild history** -- Join/leave tracking, membership timeline
12. **Guild wars** -- War tracking and war page
13. **Compare guilds/characters** -- Side-by-side comparison views
14. **Guild vocation/gender/premium distribution** -- Pie charts and breakdowns

### Phase 4: Tools & Community
15. **Census statistics** -- World/global population analytics
16. **Character transfers & name changes** -- Event logs
17. **Calculators** -- Speed, level, forge, stamina, frag calculators
18. **Compare feature** -- Up to 5 characters comparison
19. **Drome leaderboard** -- Arena rankings

### Phase 5: Engagement Features
20. **Search** -- Global guild/character search
21. **Multi-language support**
22. **Signature generator** -- Dynamic image signatures
23. **Activity/gamification system** -- Reward site participation
24. **Polls and community features**

### Technology Recommendations for Rubinot

- **Frontend (rubinot-eve):** Next.js App Router, Recharts or Chart.js for data viz, Tailwind CSS with dark theme
- **API (rubinot-api):** TypeScript REST API with caching, scheduled data collection jobs
- **Database:** PostgreSQL with TimescaleDB extension for time-series data (snapshots, online counts)
- **Caching:** Redis for hot data (current online counts, rankings)
- **Job Queue:** Bull or similar for scheduled scraping/data collection tasks

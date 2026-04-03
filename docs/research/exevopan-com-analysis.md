# Exevo Pan (exevopan.com) - Comprehensive Feature Analysis

> Research date: 2026-04-02
> Source: https://www.exevopan.com
> Open source repo: https://github.com/xandjiji/exevo-pan (157 stars, 47 forks)
> Tech stack: Next.js monorepo, TypeScript, Vercel, AWS, Cloudflare
> Creator: xandjiji

---

## Executive Summary

Exevo Pan is a mature, open-source Tibia community tool built as a Next.js monorepo. Its primary feature is a **Char Bazaar aggregator** that scrapes the official Tibia website for auction data, enriches it with computed fields (TC invested, price estimations, tags), and presents it with advanced filtering and search. Secondary features include a **Boss Tracker** with spawn chance predictions, **six calculators**, **bazaar statistics/highscores**, a **premium tier** (Exevo Pro at 250 TC / $8.99 one-time), and a **blog** with changelogs and guides. The site supports 4 languages (EN, PT, ES, PL) and has ~2,486 live auctions at any given time. Data is scraped from tibia.com every ~10 minutes, with history going back to the bazaar's inception (missing ~200k older entries).

---

## Site Navigation Map

```
exevopan.com/
|-- / (Char Bazaar - homepage, auction listings)
|-- /bosses
|   |-- /bosses (Boss Tracker - per-server)
|   |-- /bosses/{ServerName} (Boss Tracker for specific server)
|   |-- /bosses/hunting-groups (Community boss hunting groups)
|   |-- /bosses/hunting-groups/{GroupName} (Specific group page)
|-- /calculators
|   |-- /calculators (Calculator index)
|   |-- /calculators/auction-estimations (Price estimator)
|   |-- /calculators/exercise-weapons (Exercise weapon calculator)
|   |-- /calculators/loot-split (Party loot split)
|   |-- /calculators/stamina (Stamina recovery calculator)
|   |-- /calculators/imbuements-cost (Imbuement cost calculator)
|   |-- /calculators/charm-damage (Charm damage comparison)
|-- /statistics
|   |-- /statistics (Overall bazaar analytics)
|   |-- /statistics/highscores (Top 10 rankings by skill/level/bid)
|-- /highlight-auction (Auction advertising/promotion)
|-- /exevo-pro (Premium tier page)
|-- /blog (Blog index)
|   |-- /blog/about (About the project)
|   |-- /blog/about-our-data (Data sourcing explanation)
|   |-- /blog/{slug} (Individual posts - changelogs, guides)
|-- /login (Authentication)
|-- /dashboard (User dashboard - Pro features)
|   |-- /dashboard/referrals (Referral system)
```

**Header navigation**: Char Bazaar | Bosses | Calculators | Statistics | Advertise | Blog
**Footer links**: Same as header + Exevo Pro, About
**Localization**: /en, /pt, /es, /pl prefixes
**Auth**: Login button in header; Google/Discord OAuth implied

---

## Char Bazaar System (HIGH PRIORITY)

### Overview
The homepage IS the Char Bazaar. It is the core feature of the entire site. It aggregates all current character auctions from the official Tibia Char Bazaar and presents them with significantly more detail and filtering capabilities than the official site.

### Auction Listing Display

Each auction card shows:
- **Character outfit image** (animated GIF from static.tibia.com)
- **Character name** (linked to official Tibia auction page)
- **Level and vocation** (e.g., "Level 680 - Royal Paladin")
- **Server** with region flag icon (BR/NA/EU flags)
- **PvP type** with BattlEye indicator (Green/Yellow BattlEye icons + "Optional"/"Open")
- **Auction end time** (countdown format: "9h 25m, 5:00" or date "4 Apr, 4:00")
- **Price** - either "Minimum Bid" or "Current Bid" with Tibia Coin icon
- **All 8 skills** displayed as a grid:
  - Magic, Club, Fist, Sword, Fishing, Axe, Distance, Shielding
- **Animus Masteries count** (e.g., "Animus Masteries: 43")
- **Charm points** (e.g., "Charm points: 9,179")
- **Imbuements** progress (e.g., "22/23")
- **Boss points** (e.g., "Boss points: 6,760")
- **Quest completion** (e.g., "Quests: 35/42")
- **Store items owned**: Weekly Task Expansion, Charm Expansion, Prey Slot
- **Gems** (e.g., "Gems: 31-1-0" meaning lesser-regular-greater)
- **TC invested** (e.g., "3,745 invested") - **Exevo Pro exclusive** for some auctions
- **Featured items** (up to 4 item icons from the character's inventory)
- **Tags/badges**:
  - "Many charms"
  - "Many quests"
  - "Many mounts"
  - "Rare mounts" (sparkle emoji)
  - "Rare outfits" (diamond emoji)
  - "Many store cosmetics" (shopping bag emoji)
  - "Secondary skill" (sword emoji)
  - "Primal Ordeal available" (dinosaur emoji)

### Pagination
- Shows "1 - 10 of 2,486" format
- 10 auctions per page
- Total count of current auctions displayed

### Filters (from Exevo Pro comparison table)

**Regular filters (free)**:
- Server
- PvP type
- Vocation
- BattlEye type
- Level range
- Skill type and range
- Charm points range
- Bid status (has bid / no bid)
- Auction status

**Premium filters (Exevo Pro)**:
- Minimum greater gems amount
- Supreme gem modifiers (vocation-specific)
- TC invested filter
- Additional undocumented filters mentioned as "Premium filters"

### Auction Detail / Price Estimation

- Each auction has a "Go to character page" link to the official Tibia bazaar
- **Price estimation** feature: Compares a character to similar historical auctions to calculate estimated value
- Algorithm factors: **server type** + **level** + **skills** (intentionally excludes items, mounts, outfits, TC invested, charms as too subjective/noisy)
- Free tier: estimations for auctions up to 10,000 TC value
- Pro tier: unlimited estimation access

### Auction History
- Complete history database of past auctions
- Used for price estimation calculations and statistics
- Missing ~200,000 older entries (pre-corruption)
- History accessible through the Statistics section

### Auction Highlighting (Advertising)
- URL: `/highlight-auction`
- 3-step process: Select > Configure > Checkout
- Search auctions by nickname
- Highlighted auctions appear at the TOP of the listing with special styling
- Each auction card has a "Highlight your auction!" CTA link
- Exevo Pro members get discounts on highlighting
- Revenue model for the site

### Auction Notifications (Pro)
- Track specific auctions and receive notifications when they are bid on
- Notification devices management in user dashboard
- Transaction history in dashboard

---

## Boss Tracking (HIGH PRIORITY)

### Boss Tracker Overview
- URL: `/bosses` and `/bosses/{ServerName}`
- Tracks boss spawn data per server
- Server selector dropdown (defaults to Antica)
- Shows "Updated X hours ago" timestamp
- Sub-navigation: Boss Tracker | Hunting Groups

### Boss Data Displayed

Each boss entry shows:
- **Boss sprite** (animated GIF from assets.service-exevopan.com/public/sprites/bosses/{Name}.gif)
- **Boss name**
- **Spawn chance percentage** (e.g., "44.33%", "20.30%") or status text
- **Status indicators**: Color-coded squares (green = appeared, red = not appeared) for recent days
- **Expected spawn countdown** (e.g., "Expected in: 1 day", "Expected in: 148 days")
- **"Unknown"** status for bosses with no recent data
- **"No chance"** status with expected respawn time

### Boss Categories / Filters
Users can list bosses by:
- **Chance** (spawn probability, sorted highest to lowest)
- **Last seen** (most recently appeared)
- **PoI** (Pits of Inferno bosses)
- **Vampire Lord Tokens** (bosses that drop these)
- **Archdemons** (major demon bosses)
- **Rookgaard** (starter island bosses)
- **Favorites** (user-pinned bosses)

### Recently Appeared Section
- Shows bosses that were recently spotted on the selected server
- Scrollable horizontal list of boss sprites with names

### Exevo Pro Exclusive Bosses
- Premium bosses listed separately with "Exclusive Exevo Pro bosses" label
- Examples: The Pale Count, Shlorg, Man in the Cave, Ocyakao, The Welter, Yeti
- These rare bosses' spawn data is gated behind Pro

### Boss Spawn Prediction Algorithm
- Based on data scraping from official Tibia kill statistics
- Uses historical spawn patterns to calculate percentage chance
- Scripts documented in GitHub repo under `apps/bazaar-scraper`
- Boss chances data stored in `static/bossChances/` directory
- Server-save time tracking (`SS_UTC_HOUR` in codebase)

### Map Integration (tibiamaps.io)
- **NOTE**: The scraped content does NOT show direct tibiamaps.io integration on the boss pages
- The sitemap and scraped data show no embedded map views
- Boss locations are likely shown via static data or linked externally
- tibiamaps.io is referenced in other Tibia community tools but not visibly embedded on exevopan.com's boss pages
- **For Rubinot**: We could improve on this by embedding interactive maps with boss spawn locations

### Hunting Groups
- URL: `/bosses/hunting-groups`
- Community-created boss hunting groups per server
- 671+ total groups
- Each group shows:
  - Group avatar (character sprite)
  - Group name
  - Server name
  - Member count
  - Apply button
  - Optional description text
- Search by: name, server
- Create group functionality (requires login)
- **Private hunting groups** available for Exevo Pro members
- Boss checking system within groups (Pro feature from changelog-11)

---

## Calculators

### 1. Auction Price Estimator
- URL: `/calculators/auction-estimations`
- Inputs:
  - PvP type (Optional, Open, Retro Open, Hardcore, Retro Hardcore)
  - Server location (EU, NA, BR) with flag icons
  - BattlEye type (Green, Yellow)
  - Vocation (Knight, Paladin, Sorcerer, Druid, Monk)
  - Skill type (Fist, Axe/Club/Sword, Distance, Magic level, Shield)
  - Min/Max skill
  - Min/Max level
- Outputs:
  - Number of similar auctions found
  - Estimated price based on historical data
  - Suggested reading blog links
- Free tier: limited to auctions valued up to 10,000 TC
- Pro: unlimited

### 2. Exercise Weapons Calculator
- URL: `/calculators/exercise-weapons`
- Inputs:
  - Vocation (Knight, Paladin, Sorcerer, Druid, Monk) with icons
  - Skill type (Fist, Axe/Club/Sword, Distance, Magic level, Shield)
  - Current skill level
  - Target skill level
  - Percentage left
  - Loyalty bonus (None, 5%-50% in 5% increments)
  - Extra options: Exercise dummy, Double event
  - Weapon charges (Auto)
- Outputs:
  - Money cost in TC and Gold
  - Weapon breakdown (lasting/durable/regular weapon counts)
  - Time required (days, hours, minutes)

### 3. Stamina Calculator
- URL: `/calculators/stamina`
- Inputs:
  - Current stamina (HH:MM format)
  - Desired stamina (HH:MM format)
- Outputs:
  - Current stamina bar visualization
  - Rest time required (days, hours, minutes)
  - Track feature (likely notifications)

### 4. Imbuement Cost Calculator
- URL: `/calculators/imbuements-cost`
- Inputs:
  - Gold Token price
  - Tier selection (Powerful III shown)
  - Imbuement type: Vampirism (life leech), Void (mana leech), Strike (critical)
  - Material quantities and prices (e.g., Vampire Teeth, Bloody Pincers, Piece of Dead Brain)
  - Market price vs Gold Token toggle per material
- Outputs:
  - Total cost in Gold Coins

### 5. Loot Split Calculator
- URL: `/calculators/loot-split`
- Inputs:
  - Paste party hunt session log (text area)
  - Advanced options
- Outputs:
  - Session timestamp
  - Transfer list (who pays whom, with amounts)
  - Total profit per member
  - History of previous sessions (local storage)
  - Save functionality

### 6. Charm Damage Calculator
- URL: `/calculators/charm-damage`
- Inputs:
  - Average damage
  - Critical chance (10% option shown)
  - Creature HP
  - Elemental damage bonus percentage
- Outputs:
  - Low Blow charm final average damage
  - Elemental charm final average damage
  - Side-by-side comparison
  - Link to "best charms" blog article

---

## Statistics & Highscores

### Overall Statistics (`/statistics`)
- Sub-navigation: Overall | Highscores
- Displays:
  - **Total volume**: 3,425,557,915 TC (with % change)
  - **Yesterday's volume**: 2,394,538 TC (with % change)
  - **Cipsoft's total revenue**: 491,947,242 TC (with % change)
  - **Yesterday's revenue**: 321,435 TC (with % change)
  - **Auction success rate**: 62.34%
  - **Vocation distribution** (chart - not fully rendered in scrape)
- Time period toggles: 28 days / 7 days
- Charts visible for volume and revenue trends

### Highscores (`/statistics/highscores`)
- Top 10 rankings for:
  - **Bid** (highest auction prices ever)
  - **Level** (highest level characters sold)
  - **Magic** (highest magic level)
  - **Distance** (highest distance fighting)
  - **Sword** (highest sword fighting)
  - **Axe** (highest axe fighting)
  - **Club** (highest club fighting)
  - **Fist** (highest fist fighting)
  - **Shielding** (highest shielding)
  - **Fishing** (highest fishing)
- Each table shows: Rank (#), Nickname, Value
- Data comes from auction history database (characters that were sold)

---

## Data Visualizations & UX Patterns

### Chart Types
- **Line charts**: Volume over time (28-day and 7-day views in statistics)
- **Line charts**: Revenue over time
- **Pie/donut chart**: Vocation distribution
- **Bar/comparison**: Charm damage calculator
- **Color-coded indicators**: Boss spawn chance (green/red squares for daily tracking)
- **Progress bars/gauges**: Stamina calculator

### UI Patterns
- **Card-based layouts**: Auction cards, calculator cards, blog post cards, hunting group cards
- **Sidebar filters**: Left-side filter panel on Char Bazaar (implied by filter structure)
- **Paginated lists**: 10 items per page for auctions, 20 for hunting groups
- **Icon-heavy design**: Extensive use of game sprites and custom icons
- **Responsive**: Mobile-friendly (implied by modern Next.js)
- **Dark theme**: Primary UI is dark-themed
- **Hover tooltips**: Gem information expands on hover
- **Countdown timers**: Real-time auction end countdowns
- **Tags/badges**: Visual tags for character attributes (charms, quests, etc.)
- **Region flags**: BR, NA, EU flags for server location
- **BattlEye indicators**: Green and Yellow BattlEye shield icons

### External Integrations
- **TibiaTrade**: Featured item listings from tibiatrade.gg embedded on homepage
- **Bestiary Arena**: Promotional iframe embedded in footer areas
- **Tibia Blackjack**: Affiliate banner
- **otPokemon**: Promotional link
- **Edgar TC / Rei dos Coins**: TC selling partners

---

## Exevo Pro (Premium Tier)

- Price: 250 TC or $8.99 (one-time, lifetime access)
- Payment: No subscription, no credit card required
- Features:
  - Access to all premium bosses in Boss Tracker
  - TC invested calculation for any bazaar character
  - Full access to exclusive auction filters
  - Discounts for auction highlighting
  - Track auctions with bid notifications
  - Create private boss hunting groups
  - Unlimited auction price estimations
  - Referral system (earn 25 TC per referral)

---

## Blog & Content

- ~15 blog posts (changelogs, guides, analysis)
- Content types:
  - Changelogs (changelog-1 through changelog-12)
  - Guides ("best charms", "3 mistakes bazaar")
  - Analysis ("store analysis", "battleye servers")
  - Meta ("about", "about our data", "how highlighting works")
- Newsletter signup with email
- Recent posts sidebar
- Breadcrumb navigation
- Table of contents for articles
- Author profiles with Tibia character donation links
- Multi-language support (EN, PT, ES, PL)

---

## URL Patterns

| Pattern | Example | Description |
|---------|---------|-------------|
| `/` | `exevopan.com/` | Char Bazaar main page |
| `/bosses` | `exevopan.com/bosses` | Boss tracker (default server) |
| `/bosses/{Server}` | `exevopan.com/bosses/Refugia` | Boss tracker for specific server |
| `/bosses/hunting-groups` | `exevopan.com/bosses/hunting-groups` | Hunting group index |
| `/bosses/hunting-groups/{Name}` | `exevopan.com/bosses/hunting-groups/BossBusters` | Specific hunting group |
| `/calculators` | `exevopan.com/calculators` | Calculator index |
| `/calculators/{type}` | `exevopan.com/calculators/exercise-weapons` | Specific calculator |
| `/statistics` | `exevopan.com/statistics` | Overall stats |
| `/statistics/highscores` | `exevopan.com/statistics/highscores` | Rankings |
| `/highlight-auction` | `exevopan.com/highlight-auction` | Auction promotion |
| `/exevo-pro` | `exevopan.com/exevo-pro` | Premium tier |
| `/blog` | `exevopan.com/blog` | Blog index |
| `/blog/{slug}` | `exevopan.com/blog/about-our-data` | Blog post |
| `/{lang}/...` | `exevopan.com/pt/blog` | Localized version |
| `/login` | `exevopan.com/login` | Auth page |
| `/dashboard` | `exevopan.com/dashboard` | User dashboard |
| `/dashboard/referrals` | `exevopan.com/dashboard/referrals` | Referral system |

---

## Technical Architecture (from GitHub repo)

```
Monorepo structure:
├── automations/           (cron jobs, scripts)
├── apps/
│   ├── bazaar-scraper/    (data scraping from tibia.com)
│   ├── blog-worker/       (blog content processing)
│   ├── current-auctions-lambda/  (AWS Lambda for live auctions)
│   ├── exevo-pan/         (Next.js frontend app)
│   └── history-server/    (auction history API)
├── packages/
│   ├── auction-queries/   (query logic for auctions)
│   ├── config/            (shared configuration)
│   ├── data-dictionary/   (game data constants)
│   ├── db/                (database layer - Kysely)
│   ├── logging/           (logging utilities)
│   ├── mock-maker/        (test data generation)
│   ├── shared-utils/      (common utilities)
│   ├── tsconfig/          (shared TS config)
│   └── @types/            (shared type definitions)
├── scripts/               (build/deploy scripts)
├── static/                (static assets, boss chances data)
├── vercel.json            (Vercel deployment + cron config)
└── yarn.lock              (Yarn workspaces)
```

**Key technologies**:
- Next.js (frontend + API routes)
- TypeScript (entire stack)
- Kysely (database queries)
- Vercel (hosting + cron)
- AWS Lambda (real-time auction data)
- Cloudflare (CDN/protection)
- Husky (git hooks)
- ESLint + Prettier

**Data pipeline**:
1. `bazaar-scraper` makes HTTP requests to tibia.com (no official API exists)
2. Data scraped every ~10 minutes for current auctions
3. History server maintains complete auction archive
4. `current-auctions-lambda` serves live data
5. Boss data scraped from kill statistics pages
6. Static boss spawn chance data in `static/bossChances/`

---

## Data Requirements for Rubinot

### Character/Auction Data Model
```
Auction {
  id: number
  nickname: string
  level: number
  vocation: "Knight" | "Paladin" | "Sorcerer" | "Druid" | "Monk"
  server: string
  serverLocation: "EU" | "NA" | "BR"
  pvpType: "Optional" | "Open" | "Retro Open" | "Hardcore" | "Retro Hardcore"
  battleEye: "Green" | "Yellow"
  outfitId: string
  auctionEnd: DateTime
  currentBid: number | null
  minimumBid: number
  hasBeenBidded: boolean
  
  skills: {
    magic: number
    club: number
    fist: number
    sword: number
    fishing: number
    axe: number
    distance: number
    shielding: number
  }
  
  animusMasteries: number
  charmPoints: number
  imbuements: { current: number, max: number }
  bossPoints: number
  quests: { completed: number, total: number }
  gems: { lesser: number, regular: number, greater: number }
  tcInvested: number | null
  
  storeItems: string[]  // e.g., ["Weekly Task Expansion", "Charm Expansion", "Prey Slot"]
  featuredItems: { id: string, imageUrl: string, count?: number }[]
  tags: string[]  // e.g., ["Many charms", "Rare mounts", "Primal Ordeal available"]
  
  isHighlighted: boolean
}
```

### Boss Data Model
```
Boss {
  name: string
  spriteUrl: string
  category: "PoI" | "VampireLordTokens" | "Archdemons" | "Rookgaard" | "Regular" | "Premium"
  isPremium: boolean
}

BossSpawnData {
  bossName: string
  serverName: string
  spawnChance: number | null  // percentage, null = "Unknown"
  expectedIn: number | null  // days until possible spawn
  lastSeen: DateTime | null
  recentHistory: ("appeared" | "not_appeared")[]  // last N days
  status: "chance" | "no_chance" | "unknown"
  updatedAt: DateTime
}
```

### Statistics Data Model
```
BazaarStatistics {
  totalVolume: number  // TC
  yesterdayVolume: number
  totalRevenue: number  // Cipsoft cut
  yesterdayRevenue: number
  successRate: number  // percentage
  vocationDistribution: Record<string, number>
  volumeHistory: { date: Date, volume: number }[]
  revenueHistory: { date: Date, revenue: number }[]
}

Highscore {
  category: "bid" | "level" | "magic" | "distance" | "sword" | "axe" | "club" | "fist" | "shielding" | "fishing"
  entries: { rank: number, nickname: string, value: number }[]
}
```

### Hunting Group Data Model
```
HuntingGroup {
  name: string
  server: string
  memberCount: number
  avatarUrl: string
  description: string | null
  isPrivate: boolean
  members: string[]
  bossChecks: Record<string, boolean>  // boss checking system
}
```

---

## Implementation Priority Recommendations

### Phase 1: Core (Must Have)
1. **Char Bazaar Listings** - Auction card display with all data fields
2. **Auction Filters** - Server, vocation, PvP, level range, skill range, BattlEye
3. **Boss Tracker** - Per-server boss list with spawn chances and status
4. **Basic Statistics** - Volume, revenue, success rate
5. **Highscores** - Top 10 tables for all skill categories

### Phase 2: Calculators & Tools
6. **Exercise Weapons Calculator** - Most useful calculator for players
7. **Stamina Calculator** - Simple but high-traffic tool
8. **Loot Split Calculator** - Essential party hunting tool
9. **Imbuement Cost Calculator** - Market-dependent pricing tool
10. **Charm Damage Calculator** - Theory-crafting tool

### Phase 3: Advanced Features
11. **Auction Price Estimator** - Requires historical auction database
12. **Boss Hunting Groups** - Community feature with create/join/check mechanics
13. **Auction Highlighting** - Revenue feature, requires payment integration
14. **TC Invested Calculation** - Store purchase reverse engineering

### Phase 4: Premium & Social
15. **Premium Tier** - Gated features (premium bosses, extra filters, notifications)
16. **Auction Notifications** - Push/webhook when tracked auctions are bid on
17. **Referral System** - Growth mechanism
18. **Blog/CMS** - Content marketing and SEO

### Key Differentiators Rubinot Could Add
- **Embedded interactive maps** for boss locations (exevopan does NOT have this - they show no map integration despite tibiamaps.io existing)
- **Real-time auction websockets** instead of 10-minute polling
- **Character comparison tool** (side-by-side auction comparison)
- **Price history charts per character** (show how bids evolved over time)
- **Boss alert push notifications** (when a boss becomes huntable)
- **API-first approach** since we control the game server (no scraping needed)
- **Mobile app** (PWA or native)
- **Guild-level statistics** and analytics
- **Market price tracking** for items (not just characters)

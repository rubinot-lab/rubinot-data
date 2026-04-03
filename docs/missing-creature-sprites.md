# Missing Creature Sprites

48 creatures from killstats that have no sprite available from TibiaWiki or hunts.rubinot.app.
Generated 2026-04-03 during Plan 1 image serving verification.

## Guild War Objects (6)

Non-creature entities from guild wars. No sprites exist on any fansite.

- `alliance_crystal`
- `alliance_pillar`
- `alliance_tower`
- `horde_crystal`
- `horde_pillar`
- `horde_tower`

## Rubinot-Custom Creatures (15)

Creatures unique to rubinot that don't exist in Tibia Global. No fansite has sprites for these.

- `blamana`
- `blastberry`
- `bombat`
- `fallen_moohtah_master_ghar`
- `fledgling_sprout`
- `gazharagoth`
- `minion_of_gazharagoth`
- `moohtah_warrior`
- `mythical_rat`
- `nightmare_of_gazharagoth`
- `rootthing_cocoon`
- `rootthing_conservator`
- `timira_the_many-headed`
- `undergrowth`
- `painapple`

## Event/Internal Entities (11)

Special game entities, test creatures, or internal mechanics that don't have public sprites.

- `elemental_forces` — parenthesized name `(elemental forces)` in killstats
- `end_of_days` — ambiguous with boss "The End of Days" (which was downloaded)
- `goshnars_greed_feeding` — phase-specific boss variant
- `overload` — internal entity
- `pest_bug` — event creature
- `players` — meta entry in killstats (not a creature)
- `powergenerator` — quest object
- `somewhat_beatable` — event variant
- `special_demon` — internal test creature
- `special_warlock` — internal test creature
- `minor_timedisplaced_anomaly`

## Naming Mismatches (10)

Creatures where the rubinot killstats name doesn't match TibiaWiki's naming convention.
These may be fixable with manual name mapping.

- `memory_of_a_amazon` — TibiaWiki likely uses "Memory of an Amazon" (grammar: a→an)
- `memory_of_a_elf` — TibiaWiki likely uses "Memory of an Elf"
- `memory_of_a_insectoid` — TibiaWiki likely uses "Memory of an Insectoid"
- `memory_of_a_ogre` — TibiaWiki likely uses "Memory of an Ogre"
- `fairy_tail_rabbit` — possible typo, TibiaWiki might use "Fairy Tale Rabbit"
- `thorn_lilly` — TibiaWiki uses "Thorn Lily" (single L)
- `condensed_sins` — naturally ends in S, may need exact TibiaWiki lookup
- `bookworm` — ambiguous name on TibiaWiki (item vs creature)
- `avid_reader` — may exist under different name
- `old_giant_spider` — TibiaWiki may use "Old Giant Spider" but file not found

## Other (6)

Rare or removed creatures, quest-only spawns.

- `blocking_stalagmites` — environmental hazard
- `blooming_tower` — quest object
- `lovely_deer` — event creature
- `lovely_scorpion` — event creature
- `pirate_artillerist` — possibly removed or renamed
- `pirate_catapult` — quest object
- `minion_of_ghazbaran` — may use different TibiaWiki name

## Resolution Options

1. **Rubinot-custom creatures** — need sprites created manually or extracted from the game client
2. **Naming mismatches** — try "an" instead of "a", fix typos, search TibiaWiki manually
3. **Guild war objects / internal entities** — low priority, rarely displayed
4. **TibiaWiki fallback** — the handler in `assets.go` will attempt TibiaWiki on every request for a missing creature, so if the naming is fixed upstream in killstats, they'll auto-resolve

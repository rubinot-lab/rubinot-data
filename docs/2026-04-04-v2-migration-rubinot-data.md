# rubinot-data V2 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 8 new V2 data endpoints, 2 upstream debug endpoints, full auction detail expansion (20+ upstream sections), and schema drift detection to rubinot-data — enabling rubinot-api to fully migrate off V1.

**Architecture:** All new endpoints route through CDPPool + CachedFetcher (V2 path). POST batch endpoints acquire a single CDPPool tab and evaluate `Promise.allSettled()` in Chrome. Schema drift detection compares upstream JSON field names against a hardcoded registry and exposes Prometheus metrics.

**Tech Stack:** Go 1.23, Gin HTTP framework, CDPPool (Chrome DevTools Protocol), Prometheus metrics, OpenTelemetry tracing

**Design spec:** `rubinot-research/docs/superpowers/specs/2026-04-04-v2-migration-design.md`

**Repo:** `/Users/gio/git/github/rubinot-lab/rubinot-data`

**Git identity:** `gh auth switch --hostname github.com --user unwashed-and-dazed`

**Build/test commands:**
```bash
make build          # go build ./...
make test           # go test ./... -v
make lint           # go vet ./...
```

---

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `internal/domain/auctions_v2.go` | CREATE | V2AuctionDetail + 15 sub-types |
| `internal/domain/events.go` | MODIFY | Expand Event type to 15 upstream fields |
| `internal/scraper/auctions_v2.go` | CREATE | v2AuctionDetailAPIResponse struct + mapping |
| `internal/scraper/auctions_v2_test.go` | CREATE | Mapping tests with fixture data |
| `internal/scraper/schema_registry.go` | CREATE | Expected upstream field schemas |
| `internal/scraper/schema_diff.go` | CREATE | Schema comparison logic |
| `internal/scraper/schema_diff_test.go` | CREATE | Schema diff unit tests |
| `internal/scraper/v2_fetch.go` | MODIFY | Add batch fetch functions (characters, guilds, killstats, guild details, events, categories, auction details) |
| `internal/scraper/telemetry.go` | MODIFY | Add UpstreamSchemaDrift + UpstreamSchemaNewFields metrics |
| `internal/api/handlers_v2.go` | MODIFY | Add 10 new handler functions |
| `internal/api/handlers_v2_test.go` | MODIFY | Integration tests for new endpoints |
| `internal/api/router_v2.go` | MODIFY | Register 10 new routes |

---

### Task 1: V2 Auction Detail Domain Types

**Files:**
- Create: `internal/domain/auctions_v2.go`

- [ ] **Step 1: Create V2 auction detail domain types**

```go
// internal/domain/auctions_v2.go
package domain

type V2AuctionDetail struct {
	AuctionID     int    `json:"auction_id"`
	State         int    `json:"state"`
	StateName     string `json:"state_name"`
	HasWinner     bool   `json:"has_winner"`
	StartingValue int    `json:"starting_value"`
	CurrentValue  int    `json:"current_value"`
	WinningBid    int    `json:"winning_bid"`
	AuctionStart  string `json:"auction_start"`
	AuctionEnd    string `json:"auction_end"`
	Status        string `json:"status"`

	CharacterName string              `json:"character_name"`
	Level         int                 `json:"level"`
	Vocation      string              `json:"vocation"`
	VocationID    int                 `json:"vocation_id"`
	Sex           string              `json:"sex"`
	World         string              `json:"world"`
	Outfit        V2AuctionOutfitLook `json:"outfit"`

	General V2AuctionGeneral `json:"general"`

	Items             []V2AuctionItem           `json:"items"`
	ItemsTotal        int                       `json:"items_total"`
	StoreItems        []V2AuctionStoreItem      `json:"store_items"`
	StoreItemsTotal   int                       `json:"store_items_total"`
	Outfits           []V2AuctionOutfit         `json:"outfits"`
	Mounts            []V2AuctionMount          `json:"mounts"`
	Familiars         []V2AuctionFamiliar       `json:"familiars"`
	Charms            []V2AuctionCharm          `json:"charms"`
	Blessings         []V2AuctionBlessing       `json:"blessings"`
	Titles            []int                     `json:"titles"`
	Gems              []V2AuctionGem            `json:"gems"`
	Bosstiaries       []V2AuctionBosstiary      `json:"bosstiaries"`
	BosstiariosTotal  int                       `json:"bosstiarios_total"`
	WeaponProficiency []V2AuctionWeaponProf     `json:"weapon_proficiency"`
	BattlepassSeasons []V2AuctionBattlepass     `json:"battlepass_seasons"`
	Achievements      []V2AuctionAchievement    `json:"achievements"`
	BountyTalismans   []V2AuctionBountyTalisman `json:"bounty_talismans"`
	BountyPoints      int                       `json:"bounty_points"`
	TotalBountyPoints int                       `json:"total_bounty_points"`
	BountyRerolls     int                       `json:"bounty_rerolls"`
	Auras             []V2AuctionAura           `json:"auras"`
	HirelingSkills    []V2AuctionHirelingSkill  `json:"hireling_skills"`
	HirelingWardrobe  []V2AuctionHirelingItem   `json:"hireling_wardrobe"`

	HighlightItems    []AuctionItem    `json:"highlight_items"`
	HighlightAugments []AuctionAugment `json:"highlight_augments"`
}

type V2AuctionGeneral struct {
	Health               int           `json:"health"`
	HealthMax            int           `json:"health_max"`
	Mana                 int           `json:"mana"`
	ManaMax              int           `json:"mana_max"`
	ManaSpent            int64         `json:"mana_spent"`
	Cap                  int           `json:"cap"`
	Stamina              int           `json:"stamina"`
	Soul                 int           `json:"soul"`
	Experience           int64         `json:"experience"`
	MagLevel             int           `json:"mag_level"`
	Skills               AuctionSkills `json:"skills"`
	MountsCount          int           `json:"mounts_count"`
	OutfitsCount         int           `json:"outfits_count"`
	TitlesCount          int           `json:"titles_count"`
	LinkedTasks          int           `json:"linked_tasks"`
	CreateDate           string        `json:"create_date"`
	Balance              int64         `json:"balance"`
	TotalMoney           int64         `json:"total_money"`
	AchievementPoints    int           `json:"achievement_points"`
	CharmPoints          int           `json:"charm_points"`
	SpentCharmPoints     int           `json:"spent_charm_points"`
	AvailableCharmPoints int           `json:"available_charm_points"`
	SpentMinorEchoes     int           `json:"spent_minor_echoes"`
	AvailableMinorEchoes int           `json:"available_minor_echoes"`
	CharmExpansion       bool          `json:"charm_expansion"`
	StreakDays           int           `json:"streak_days"`
	HuntingTaskPoints    int           `json:"hunting_task_points"`
	ThirdPrey            bool          `json:"third_prey"`
	ThirdHunting         bool          `json:"third_hunting"`
	PermanentWeeklySlot  bool          `json:"permanent_weekly_slot"`
	PreyWildcards        int           `json:"prey_wildcards"`
	HirelingCount        int           `json:"hireling_count"`
	HirelingJobs         int           `json:"hireling_jobs"`
	HirelingOutfits      int           `json:"hireling_outfits"`
	Dust                 int           `json:"dust"`
	DustMax              int           `json:"dust_max"`
	BossPoints           int           `json:"boss_points"`
	WheelPoints          int           `json:"wheel_points"`
	MaxWheelPoints       int           `json:"max_wheel_points"`
	GpActive             bool          `json:"gp_active"`
	GpPoints             int           `json:"gp_points"`
}

type V2AuctionOutfitLook struct {
	LookType   int `json:"looktype"`
	LookHead   int `json:"lookhead,omitempty"`
	LookBody   int `json:"lookbody,omitempty"`
	LookLegs   int `json:"looklegs,omitempty"`
	LookFeet   int `json:"lookfeet,omitempty"`
	LookAddons int `json:"lookaddons,omitempty"`
	LookMount  int `json:"lookmount,omitempty"`
	Direction  int `json:"direction,omitempty"`
}

type V2AuctionItem struct {
	Name        string `json:"name"`
	SlotID      int    `json:"slot_id"`
	ClientID    int    `json:"client_id"`
	ItemID      int    `json:"item_id"`
	Count       int    `json:"count"`
	Tier        int    `json:"tier"`
	Description string `json:"description,omitempty"`
}

type V2AuctionStoreItem struct {
	Name     string `json:"name"`
	SlotID   int    `json:"slot_id"`
	ClientID int    `json:"client_id"`
	ItemID   int    `json:"item_id"`
	Count    int    `json:"count"`
}

type V2AuctionOutfit struct {
	ID     int          `json:"id"`
	Addons int          `json:"addons"`
	Info   V2OutfitInfo `json:"info"`
}

type V2OutfitInfo struct {
	LookType int    `json:"looktype"`
	Name     string `json:"name"`
}

type V2AuctionMount struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ClientID int    `json:"client_id"`
}

type V2AuctionFamiliar struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ClientID int    `json:"client_id"`
}

type V2AuctionCharm struct {
	ID     int `json:"id"`
	Tier   int `json:"tier"`
	RaceID int `json:"race_id"`
	Type   int `json:"type"`
}

type V2AuctionBlessing struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type V2AuctionGem struct {
	SlotID   int `json:"slot_id"`
	ClientID int `json:"client_id"`
	ItemID   int `json:"item_id"`
}

type V2AuctionBosstiary struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Kills   int    `json:"kills"`
	Gained1 int    `json:"gained1"`
	Gained2 int    `json:"gained2"`
	Gained3 int    `json:"gained3"`
}

type V2AuctionWeaponProf struct {
	ItemID          int  `json:"item_id"`
	Experience      int  `json:"experience"`
	WeaponLevel     int  `json:"weapon_level"`
	MasteryAchieved bool `json:"mastery_achieved"`
	ActivePerks     int  `json:"active_perks"`
}

type V2AuctionBattlepass struct {
	Season     int  `json:"season"`
	Points     int  `json:"points"`
	Active     bool `json:"active"`
	ShopPoints int  `json:"shoppoints"`
	Steps      int  `json:"steps"`
}

type V2AuctionAchievement struct {
	ID         int   `json:"id"`
	UnlockedAt int64 `json:"unlocked_at"`
}

type V2AuctionBountyTalisman struct {
	Type        int `json:"type"`
	Level       int `json:"level"`
	EffectValue int `json:"effect_value"`
}

type V2AuctionAura struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type V2AuctionHirelingSkill struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type V2AuctionHirelingItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type V2AuctionsDetailsResult struct {
	Type         string             `json:"type"`
	Page         int                `json:"page"`
	TotalResults int                `json:"total_results"`
	TotalPages   int                `json:"total_pages"`
	Entries      []V2AuctionDetail  `json:"entries"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 3: Commit**

```bash
git add internal/domain/auctions_v2.go
git commit -m "feat(domain): add V2 auction detail types with all 20+ upstream sections"
```

---

### Task 2: V2 Auction Detail API Response Struct + Mapping

**Files:**
- Create: `internal/scraper/auctions_v2.go`
- Create: `internal/scraper/auctions_v2_test.go`

This struct maps the raw upstream JSON from `/api/bazaar/{id}`. Verified via Playwright on 2026-04-04 — the upstream returns 31 top-level keys. The current `auctionDetailAPIResponse` only captures 5 sections. This new struct captures all of them.

- [ ] **Step 1: Write failing test for V2 auction detail mapping**

```go
// internal/scraper/auctions_v2_test.go
package scraper

import (
	"encoding/json"
	"testing"
)

func TestMapV2AuctionDetailResponse(t *testing.T) {
	raw := `{
		"auction": {"id": 187649, "state": 1, "stateName": "Active", "owner": 1, "startingValue": 1000, "currentValue": 1001, "winningBid": 1001, "highestBidderId": 0, "hasWinner": false, "auctionStart": 1743634689, "auctionEnd": 1743778800},
		"player": {"name": "Test Knight", "level": 500, "vocation": 1, "vocationName": "Elite Knight", "sex": 1, "worldName": "Auroria", "lookType": 129, "lookHead": 78, "lookBody": 69, "lookLegs": 58, "lookFeet": 76, "lookAddons": 3, "direction": 3, "lookMount": 0},
		"general": {"health": 4000, "healthMax": 4000, "mana": 2000, "manaMax": 2000, "manaSpent": 500000, "cap": 1520, "stamina": 2520, "soul": 200, "experience": 14000000000, "magLevel": 15, "skills": {"axe": 120, "club": 30, "sword": 30, "distance": 30, "shielding": 115, "fishing": 20, "fist": 15, "magic": 12}, "mountsCount": 25, "outfitsCount": 35, "titlesCount": 10, "linkedTasks": 5, "createDate": 1700000000, "balance": 0, "totalMoney": 50000000, "achievementPoints": 420, "charmPoints": 6500, "spentCharmPoints": 4000, "availableCharmPoints": 2500, "spentMinorEchoes": 0, "availableMinorEchoes": 0, "charmExpansion": true, "streakDays": 30, "huntingTaskPoints": 100, "thirdPrey": false, "thirdHunting": false, "permanentWeeklyTaskSlot": false, "preyWildcards": 5, "hirelingCount": 2, "hirelingJobs": 3, "hirelingOutfits": 1, "dust": 50, "dustMax": 100, "bossPoints": 8500, "wheelPoints": 1000, "maxWheelPoints": 2000, "gpActive": false, "gpPoints": 0},
		"items": [{"name": "Magic Plate Armor", "slotId": 4, "clientId": 3366, "itemId": 3366, "count": 1, "tier": 3, "description": "Arm:17"}],
		"itemsTotal": 1,
		"storeItems": [],
		"storeItemsTotal": 0,
		"outfits": [{"id": 131, "addons": 3, "info": {"lookType": 131, "name": "Citizen"}}],
		"mounts": [{"id": 1, "name": "Widow Queen", "clientId": 368}],
		"familiars": [],
		"charms": [{"id": 1, "tier": 0, "raceId": 100, "type": 1}],
		"blessings": [{"name": "Twist of Fate", "count": 1}],
		"titles": [1, 5, 10],
		"gems": [],
		"bosstiaries": [{"id": 1, "name": "Orshabaal", "kills": 5, "gained1": 1, "gained2": 0, "gained3": 0}],
		"bosstiariosTotal": 1,
		"weaponProficiency": [{"itemId": 1, "experience": 5000, "weaponLevel": 3, "masteryAchieved": false, "activePerks": 1}],
		"battlepassSeasons": [{"season": 1, "points": 500, "active": true, "shoppoints": 100, "steps": 10}],
		"achievements": [{"id": 1, "unlockedAt": 1700000000}],
		"bountyTalismans": [{"type": 1, "level": 2, "effectValue": 10}],
		"bountyPoints": 100,
		"totalBountyPoints": 500,
		"bountyRerolls": 3,
		"auras": [],
		"hirelingSkills": [],
		"hirelingWardrobe": [],
		"highlightItems": [{"itemId": 3366, "clientId": 3366, "tier": 3, "count": 1, "name": "Magic Plate Armor"}],
		"highlightAugments": [{"text": "120 axe fighting", "argType": 0}],
		"storages": [[1000, 1], [1001, 5]],
		"isAdmin": false,
		"adminInfo": null
	}`

	var payload v2AuctionDetailAPIResponse
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result := mapV2AuctionDetailResponse(payload)

	if result.AuctionID != 187649 {
		t.Errorf("AuctionID = %d, want 187649", result.AuctionID)
	}
	if result.CharacterName != "Test Knight" {
		t.Errorf("CharacterName = %q, want %q", result.CharacterName, "Test Knight")
	}
	if result.Level != 500 {
		t.Errorf("Level = %d, want 500", result.Level)
	}
	if result.General.Health != 4000 {
		t.Errorf("General.Health = %d, want 4000", result.General.Health)
	}
	if result.General.Experience != 14000000000 {
		t.Errorf("General.Experience = %d, want 14000000000", result.General.Experience)
	}
	if result.General.BossPoints != 8500 {
		t.Errorf("General.BossPoints = %d, want 8500", result.General.BossPoints)
	}
	if result.General.Skills.Axe != 120 {
		t.Errorf("General.Skills.Axe = %d, want 120", result.General.Skills.Axe)
	}
	if len(result.Items) != 1 {
		t.Fatalf("Items len = %d, want 1", len(result.Items))
	}
	if result.Items[0].Name != "Magic Plate Armor" {
		t.Errorf("Items[0].Name = %q, want %q", result.Items[0].Name, "Magic Plate Armor")
	}
	if result.Items[0].Tier != 3 {
		t.Errorf("Items[0].Tier = %d, want 3", result.Items[0].Tier)
	}
	if len(result.Outfits) != 1 {
		t.Fatalf("Outfits len = %d, want 1", len(result.Outfits))
	}
	if result.Outfits[0].Info.Name != "Citizen" {
		t.Errorf("Outfits[0].Info.Name = %q, want %q", result.Outfits[0].Info.Name, "Citizen")
	}
	if len(result.Mounts) != 1 || result.Mounts[0].Name != "Widow Queen" {
		t.Errorf("Mounts = %+v", result.Mounts)
	}
	if len(result.Charms) != 1 || result.Charms[0].RaceID != 100 {
		t.Errorf("Charms = %+v", result.Charms)
	}
	if len(result.Blessings) != 1 || result.Blessings[0].Name != "Twist of Fate" {
		t.Errorf("Blessings = %+v", result.Blessings)
	}
	if len(result.Titles) != 3 {
		t.Errorf("Titles len = %d, want 3", len(result.Titles))
	}
	if len(result.Bosstiaries) != 1 || result.Bosstiaries[0].Name != "Orshabaal" {
		t.Errorf("Bosstiaries = %+v", result.Bosstiaries)
	}
	if len(result.WeaponProficiency) != 1 || result.WeaponProficiency[0].WeaponLevel != 3 {
		t.Errorf("WeaponProficiency = %+v", result.WeaponProficiency)
	}
	if len(result.BattlepassSeasons) != 1 || result.BattlepassSeasons[0].Points != 500 {
		t.Errorf("BattlepassSeasons = %+v", result.BattlepassSeasons)
	}
	if len(result.Achievements) != 1 || result.Achievements[0].ID != 1 {
		t.Errorf("Achievements = %+v", result.Achievements)
	}
	if len(result.BountyTalismans) != 1 || result.BountyTalismans[0].EffectValue != 10 {
		t.Errorf("BountyTalismans = %+v", result.BountyTalismans)
	}
	if result.BountyPoints != 100 || result.TotalBountyPoints != 500 || result.BountyRerolls != 3 {
		t.Errorf("BountyPoints=%d TotalBountyPoints=%d BountyRerolls=%d", result.BountyPoints, result.TotalBountyPoints, result.BountyRerolls)
	}
	if result.Outfit.LookType != 129 || result.Outfit.LookAddons != 3 {
		t.Errorf("Outfit = %+v", result.Outfit)
	}
	if result.Sex != "Male" {
		t.Errorf("Sex = %q, want Male", result.Sex)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scraper/ -run TestMapV2AuctionDetailResponse -v`
Expected: FAIL — `v2AuctionDetailAPIResponse` and `mapV2AuctionDetailResponse` not defined

- [ ] **Step 3: Create V2 API response struct + mapping function**

```go
// internal/scraper/auctions_v2.go
package scraper

import (
	"strings"

	"github.com/giovannirco/rubinot-data/internal/domain"
)

type v2AuctionDetailAPIResponse struct {
	Auction struct {
		ID              int    `json:"id"`
		State           int    `json:"state"`
		StateName       string `json:"stateName"`
		Owner           int    `json:"owner"`
		StartingValue   int    `json:"startingValue"`
		CurrentValue    int    `json:"currentValue"`
		WinningBid      int    `json:"winningBid"`
		HighestBidderID int    `json:"highestBidderId"`
		HasWinner       bool   `json:"hasWinner"`
		AuctionStart    int64  `json:"auctionStart"`
		AuctionEnd      int64  `json:"auctionEnd"`
	} `json:"auction"`
	Player struct {
		Name       string `json:"name"`
		Level      int    `json:"level"`
		Vocation   int    `json:"vocation"`
		VocName    string `json:"vocationName"`
		Sex        int    `json:"sex"`
		WorldName  string `json:"worldName"`
		LookType   int    `json:"lookType"`
		LookHead   int    `json:"lookHead"`
		LookBody   int    `json:"lookBody"`
		LookLegs   int    `json:"lookLegs"`
		LookFeet   int    `json:"lookFeet"`
		LookAddons int    `json:"lookAddons"`
		Direction  int    `json:"direction"`
		LookMount  int    `json:"lookMount"`
	} `json:"player"`
	General struct {
		Health               int   `json:"health"`
		HealthMax            int   `json:"healthMax"`
		Mana                 int   `json:"mana"`
		ManaMax              int   `json:"manaMax"`
		ManaSpent            int64 `json:"manaSpent"`
		Cap                  int   `json:"cap"`
		Stamina              int   `json:"stamina"`
		Soul                 int   `json:"soul"`
		Experience           int64 `json:"experience"`
		MagLevel             int   `json:"magLevel"`
		Skills               struct {
			Axe       int `json:"axe"`
			Club      int `json:"club"`
			Sword     int `json:"sword"`
			Distance  int `json:"distance"`
			Dist      int `json:"dist"`
			Shielding int `json:"shielding"`
			Fishing   int `json:"fishing"`
			Fist      int `json:"fist"`
			Magic     int `json:"magic"`
		} `json:"skills"`
		MountsCount          int   `json:"mountsCount"`
		OutfitsCount         int   `json:"outfitsCount"`
		TitlesCount          int   `json:"titlesCount"`
		LinkedTasks          int   `json:"linkedTasks"`
		CreateDate           int64 `json:"createDate"`
		Balance              int64 `json:"balance"`
		TotalMoney           int64 `json:"totalMoney"`
		AchievementPoints    int   `json:"achievementPoints"`
		CharmPoints          int   `json:"charmPoints"`
		SpentCharmPoints     int   `json:"spentCharmPoints"`
		AvailableCharmPoints int   `json:"availableCharmPoints"`
		SpentMinorEchoes     int   `json:"spentMinorEchoes"`
		AvailableMinorEchoes int   `json:"availableMinorEchoes"`
		CharmExpansion       bool  `json:"charmExpansion"`
		StreakDays           int   `json:"streakDays"`
		HuntingTaskPoints    int   `json:"huntingTaskPoints"`
		ThirdPrey            bool  `json:"thirdPrey"`
		ThirdHunting         bool  `json:"thirdHunting"`
		PermanentWeeklySlot  bool  `json:"permanentWeeklyTaskSlot"`
		PreyWildcards        int   `json:"preyWildcards"`
		HirelingCount        int   `json:"hirelingCount"`
		HirelingJobs         int   `json:"hirelingJobs"`
		HirelingOutfits      int   `json:"hirelingOutfits"`
		Dust                 int   `json:"dust"`
		DustMax              int   `json:"dustMax"`
		BossPoints           int   `json:"bossPoints"`
		WheelPoints          int   `json:"wheelPoints"`
		MaxWheelPoints       int   `json:"maxWheelPoints"`
		GpActive             bool  `json:"gpActive"`
		GpPoints             int   `json:"gpPoints"`
	} `json:"general"`
	Items []struct {
		Name        string `json:"name"`
		SlotID      int    `json:"slotId"`
		ClientID    int    `json:"clientId"`
		ItemID      int    `json:"itemId"`
		Count       int    `json:"count"`
		Tier        int    `json:"tier"`
		Description string `json:"description"`
	} `json:"items"`
	ItemsTotal int `json:"itemsTotal"`
	StoreItems []struct {
		Name     string `json:"name"`
		SlotID   int    `json:"slotId"`
		ClientID int    `json:"clientId"`
		ItemID   int    `json:"itemId"`
		Count    int    `json:"count"`
	} `json:"storeItems"`
	StoreItemsTotal int `json:"storeItemsTotal"`
	Outfits []struct {
		ID     int `json:"id"`
		Addons int `json:"addons"`
		Info   struct {
			LookType int    `json:"lookType"`
			Name     string `json:"name"`
		} `json:"info"`
	} `json:"outfits"`
	Mounts []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		ClientID int    `json:"clientId"`
	} `json:"mounts"`
	Familiars []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		ClientID int    `json:"clientId"`
	} `json:"familiars"`
	Charms []struct {
		ID     int `json:"id"`
		Tier   int `json:"tier"`
		RaceID int `json:"raceId"`
		Type   int `json:"type"`
	} `json:"charms"`
	Blessings []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"blessings"`
	Titles []int `json:"titles"`
	Gems   []struct {
		SlotID   int `json:"slotId"`
		ClientID int `json:"clientId"`
		ItemID   int `json:"itemId"`
	} `json:"gems"`
	Bosstiaries []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Kills   int    `json:"kills"`
		Gained1 int    `json:"gained1"`
		Gained2 int    `json:"gained2"`
		Gained3 int    `json:"gained3"`
	} `json:"bosstiaries"`
	BosstiariosTotal int `json:"bosstiariosTotal"`
	WeaponProficiency []struct {
		ItemID          int  `json:"itemId"`
		Experience      int  `json:"experience"`
		WeaponLevel     int  `json:"weaponLevel"`
		MasteryAchieved bool `json:"masteryAchieved"`
		ActivePerks     int  `json:"activePerks"`
	} `json:"weaponProficiency"`
	BattlepassSeasons []struct {
		Season     int  `json:"season"`
		Points     int  `json:"points"`
		Active     bool `json:"active"`
		ShopPoints int  `json:"shoppoints"`
		Steps      int  `json:"steps"`
	} `json:"battlepassSeasons"`
	Achievements []struct {
		ID         int   `json:"id"`
		UnlockedAt int64 `json:"unlockedAt"`
	} `json:"achievements"`
	BountyTalismans []struct {
		Type        int `json:"type"`
		Level       int `json:"level"`
		EffectValue int `json:"effectValue"`
	} `json:"bountyTalismans"`
	BountyPoints      int `json:"bountyPoints"`
	TotalBountyPoints int `json:"totalBountyPoints"`
	BountyRerolls     int `json:"bountyRerolls"`
	Auras []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"auras"`
	HirelingSkills []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"hirelingSkills"`
	HirelingWardrobe []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"hirelingWardrobe"`
	HighlightItems []struct {
		ItemID   int    `json:"itemId"`
		ClientID int    `json:"clientId"`
		Tier     int    `json:"tier"`
		Count    int    `json:"count"`
		Name     string `json:"name"`
	} `json:"highlightItems"`
	HighlightAugments []struct {
		Text    string `json:"text"`
		ArgType int    `json:"argType"`
	} `json:"highlightAugments"`
}

func mapV2AuctionDetailResponse(p v2AuctionDetailAPIResponse) domain.V2AuctionDetail {
	items := make([]domain.V2AuctionItem, 0, len(p.Items))
	for _, i := range p.Items {
		items = append(items, domain.V2AuctionItem{
			Name: i.Name, SlotID: i.SlotID, ClientID: i.ClientID,
			ItemID: i.ItemID, Count: i.Count, Tier: i.Tier, Description: i.Description,
		})
	}

	storeItems := make([]domain.V2AuctionStoreItem, 0, len(p.StoreItems))
	for _, i := range p.StoreItems {
		storeItems = append(storeItems, domain.V2AuctionStoreItem{
			Name: i.Name, SlotID: i.SlotID, ClientID: i.ClientID, ItemID: i.ItemID, Count: i.Count,
		})
	}

	outfits := make([]domain.V2AuctionOutfit, 0, len(p.Outfits))
	for _, o := range p.Outfits {
		outfits = append(outfits, domain.V2AuctionOutfit{
			ID: o.ID, Addons: o.Addons,
			Info: domain.V2OutfitInfo{LookType: o.Info.LookType, Name: o.Info.Name},
		})
	}

	mounts := make([]domain.V2AuctionMount, 0, len(p.Mounts))
	for _, m := range p.Mounts {
		mounts = append(mounts, domain.V2AuctionMount{ID: m.ID, Name: m.Name, ClientID: m.ClientID})
	}

	familiars := make([]domain.V2AuctionFamiliar, 0, len(p.Familiars))
	for _, f := range p.Familiars {
		familiars = append(familiars, domain.V2AuctionFamiliar{ID: f.ID, Name: f.Name, ClientID: f.ClientID})
	}

	charms := make([]domain.V2AuctionCharm, 0, len(p.Charms))
	for _, ch := range p.Charms {
		charms = append(charms, domain.V2AuctionCharm{ID: ch.ID, Tier: ch.Tier, RaceID: ch.RaceID, Type: ch.Type})
	}

	blessings := make([]domain.V2AuctionBlessing, 0, len(p.Blessings))
	for _, b := range p.Blessings {
		blessings = append(blessings, domain.V2AuctionBlessing{Name: b.Name, Count: b.Count})
	}

	gems := make([]domain.V2AuctionGem, 0, len(p.Gems))
	for _, g := range p.Gems {
		gems = append(gems, domain.V2AuctionGem{SlotID: g.SlotID, ClientID: g.ClientID, ItemID: g.ItemID})
	}

	bosstiaries := make([]domain.V2AuctionBosstiary, 0, len(p.Bosstiaries))
	for _, b := range p.Bosstiaries {
		bosstiaries = append(bosstiaries, domain.V2AuctionBosstiary{
			ID: b.ID, Name: b.Name, Kills: b.Kills, Gained1: b.Gained1, Gained2: b.Gained2, Gained3: b.Gained3,
		})
	}

	weaponProf := make([]domain.V2AuctionWeaponProf, 0, len(p.WeaponProficiency))
	for _, w := range p.WeaponProficiency {
		weaponProf = append(weaponProf, domain.V2AuctionWeaponProf{
			ItemID: w.ItemID, Experience: w.Experience, WeaponLevel: w.WeaponLevel,
			MasteryAchieved: w.MasteryAchieved, ActivePerks: w.ActivePerks,
		})
	}

	battlepass := make([]domain.V2AuctionBattlepass, 0, len(p.BattlepassSeasons))
	for _, b := range p.BattlepassSeasons {
		battlepass = append(battlepass, domain.V2AuctionBattlepass{
			Season: b.Season, Points: b.Points, Active: b.Active, ShopPoints: b.ShopPoints, Steps: b.Steps,
		})
	}

	achievements := make([]domain.V2AuctionAchievement, 0, len(p.Achievements))
	for _, a := range p.Achievements {
		achievements = append(achievements, domain.V2AuctionAchievement{ID: a.ID, UnlockedAt: a.UnlockedAt})
	}

	bountyTalismans := make([]domain.V2AuctionBountyTalisman, 0, len(p.BountyTalismans))
	for _, b := range p.BountyTalismans {
		bountyTalismans = append(bountyTalismans, domain.V2AuctionBountyTalisman{
			Type: b.Type, Level: b.Level, EffectValue: b.EffectValue,
		})
	}

	auras := make([]domain.V2AuctionAura, 0, len(p.Auras))
	for _, a := range p.Auras {
		auras = append(auras, domain.V2AuctionAura{ID: a.ID, Name: a.Name})
	}

	hirelingSkills := make([]domain.V2AuctionHirelingSkill, 0, len(p.HirelingSkills))
	for _, h := range p.HirelingSkills {
		hirelingSkills = append(hirelingSkills, domain.V2AuctionHirelingSkill{ID: h.ID, Name: h.Name})
	}

	hirelingWardrobe := make([]domain.V2AuctionHirelingItem, 0, len(p.HirelingWardrobe))
	for _, h := range p.HirelingWardrobe {
		hirelingWardrobe = append(hirelingWardrobe, domain.V2AuctionHirelingItem{ID: h.ID, Name: h.Name})
	}

	highlightItems := make([]domain.AuctionItem, 0, len(p.HighlightItems))
	for _, h := range p.HighlightItems {
		id := h.ItemID
		if id == 0 {
			id = h.ClientID
		}
		highlightItems = append(highlightItems, domain.AuctionItem{ID: id, Name: h.Name})
	}

	highlightAugments := make([]domain.AuctionAugment, 0, len(p.HighlightAugments))
	for _, h := range p.HighlightAugments {
		highlightAugments = append(highlightAugments, domain.AuctionAugment{
			ID: h.ArgType, Name: strings.TrimSpace(h.Text),
		})
	}

	distSkill := p.General.Skills.Distance
	if distSkill == 0 {
		distSkill = p.General.Skills.Dist
	}

	return domain.V2AuctionDetail{
		AuctionID:     p.Auction.ID,
		State:         p.Auction.State,
		StateName:     strings.TrimSpace(p.Auction.StateName),
		HasWinner:     p.Auction.HasWinner,
		StartingValue: p.Auction.StartingValue,
		CurrentValue:  p.Auction.CurrentValue,
		WinningBid:    p.Auction.WinningBid,
		AuctionStart:  unixSecondsToRFC3339(p.Auction.AuctionStart),
		AuctionEnd:    unixSecondsToRFC3339(p.Auction.AuctionEnd),
		Status:        strings.TrimSpace(p.Auction.StateName),

		CharacterName: strings.TrimSpace(p.Player.Name),
		Level:         p.Player.Level,
		Vocation:      strings.TrimSpace(p.Player.VocName),
		VocationID:    p.Player.Vocation,
		Sex:           sexName(p.Player.Sex),
		World:         strings.TrimSpace(p.Player.WorldName),
		Outfit: domain.V2AuctionOutfitLook{
			LookType: p.Player.LookType, LookHead: p.Player.LookHead,
			LookBody: p.Player.LookBody, LookLegs: p.Player.LookLegs,
			LookFeet: p.Player.LookFeet, LookAddons: p.Player.LookAddons,
			LookMount: p.Player.LookMount, Direction: p.Player.Direction,
		},

		General: domain.V2AuctionGeneral{
			Health: p.General.Health, HealthMax: p.General.HealthMax,
			Mana: p.General.Mana, ManaMax: p.General.ManaMax,
			ManaSpent: p.General.ManaSpent, Cap: p.General.Cap,
			Stamina: p.General.Stamina, Soul: p.General.Soul,
			Experience: p.General.Experience, MagLevel: p.General.MagLevel,
			Skills: domain.AuctionSkills{
				Axe: p.General.Skills.Axe, Club: p.General.Skills.Club,
				Sword: p.General.Skills.Sword, Distance: distSkill,
				Shielding: p.General.Skills.Shielding, Fishing: p.General.Skills.Fishing,
				Fist: p.General.Skills.Fist,
			},
			MountsCount: p.General.MountsCount, OutfitsCount: p.General.OutfitsCount,
			TitlesCount: p.General.TitlesCount, LinkedTasks: p.General.LinkedTasks,
			CreateDate: unixSecondsToRFC3339(p.General.CreateDate),
			Balance: p.General.Balance, TotalMoney: p.General.TotalMoney,
			AchievementPoints: p.General.AchievementPoints,
			CharmPoints: p.General.CharmPoints, SpentCharmPoints: p.General.SpentCharmPoints,
			AvailableCharmPoints: p.General.AvailableCharmPoints,
			SpentMinorEchoes: p.General.SpentMinorEchoes, AvailableMinorEchoes: p.General.AvailableMinorEchoes,
			CharmExpansion: p.General.CharmExpansion, StreakDays: p.General.StreakDays,
			HuntingTaskPoints: p.General.HuntingTaskPoints,
			ThirdPrey: p.General.ThirdPrey, ThirdHunting: p.General.ThirdHunting,
			PermanentWeeklySlot: p.General.PermanentWeeklySlot,
			PreyWildcards: p.General.PreyWildcards,
			HirelingCount: p.General.HirelingCount, HirelingJobs: p.General.HirelingJobs,
			HirelingOutfits: p.General.HirelingOutfits,
			Dust: p.General.Dust, DustMax: p.General.DustMax,
			BossPoints: p.General.BossPoints,
			WheelPoints: p.General.WheelPoints, MaxWheelPoints: p.General.MaxWheelPoints,
			GpActive: p.General.GpActive, GpPoints: p.General.GpPoints,
		},

		Items: items, ItemsTotal: p.ItemsTotal,
		StoreItems: storeItems, StoreItemsTotal: p.StoreItemsTotal,
		Outfits: outfits, Mounts: mounts, Familiars: familiars,
		Charms: charms, Blessings: blessings, Titles: p.Titles,
		Gems: gems, Bosstiaries: bosstiaries, BosstiariosTotal: p.BosstiariosTotal,
		WeaponProficiency: weaponProf, BattlepassSeasons: battlepass,
		Achievements: achievements, BountyTalismans: bountyTalismans,
		BountyPoints: p.BountyPoints, TotalBountyPoints: p.TotalBountyPoints,
		BountyRerolls: p.BountyRerolls,
		Auras: auras, HirelingSkills: hirelingSkills, HirelingWardrobe: hirelingWardrobe,
		HighlightItems: highlightItems, HighlightAugments: highlightAugments,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/scraper/ -run TestMapV2AuctionDetailResponse -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scraper/auctions_v2.go internal/scraper/auctions_v2_test.go
git commit -m "feat(scraper): add V2 auction detail API response struct and mapping

Maps all 20+ upstream sections from /api/bazaar/{id}: general (40 fields),
items, outfits, mounts, charms, blessings, titles, gems, bosstiaries,
weapon proficiency, battlepass, achievements, bounty talismans, auras,
hirelings. Verified against live upstream via Playwright 2026-04-04."
```

---

### Task 3: Schema Registry + Diff Logic

**Files:**
- Create: `internal/scraper/schema_registry.go`
- Create: `internal/scraper/schema_diff.go`
- Create: `internal/scraper/schema_diff_test.go`

- [ ] **Step 1: Write failing test for schema diff**

```go
// internal/scraper/schema_diff_test.go
package scraper

import (
	"testing"
)

func TestCompareSchemaDriftDetected(t *testing.T) {
	rawJSON := []byte(`{"worlds": [], "totalOnline": 100, "overallRecord": 200, "overallRecordTime": 300, "newField": "surprise"}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "drift" {
		t.Errorf("Status = %q, want drift", diff.Status)
	}
	if len(diff.NewFields) != 1 || diff.NewFields[0] != "newField" {
		t.Errorf("NewFields = %v, want [newField]", diff.NewFields)
	}
	if len(diff.MissingFields) != 0 {
		t.Errorf("MissingFields = %v, want []", diff.MissingFields)
	}
}

func TestCompareSchemaMatch(t *testing.T) {
	rawJSON := []byte(`{"worlds": [], "totalOnline": 100, "overallRecord": 200, "overallRecordTime": 300}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "match" {
		t.Errorf("Status = %q, want match", diff.Status)
	}
}

func TestCompareSchemaMissingField(t *testing.T) {
	rawJSON := []byte(`{"worlds": [], "totalOnline": 100}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "drift" {
		t.Errorf("Status = %q, want drift", diff.Status)
	}
	if len(diff.MissingFields) != 2 {
		t.Errorf("MissingFields = %v, want 2 items", diff.MissingFields)
	}
}

func TestCompareSchemaNestedDrift(t *testing.T) {
	rawJSON := []byte(`{"worlds": [{"name": "Auroria", "pvpType": "Open PvP", "pvpTypeLabel": "Open PvP", "worldType": "yellow", "locked": false, "playersOnline": 500, "newWorldField": true}], "totalOnline": 100, "overallRecord": 200, "overallRecordTime": 300}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "drift" {
		t.Errorf("Status = %q, want drift", diff.Status)
	}
	nd, ok := diff.NestedDiffs["worlds[0]"]
	if !ok {
		t.Fatal("expected nested diff for worlds[0]")
	}
	if len(nd.NewFields) != 1 || nd.NewFields[0] != "newWorldField" {
		t.Errorf("NestedDiffs[worlds[0]].NewFields = %v, want [newWorldField]", nd.NewFields)
	}
}

func TestCompareSchemaUnknownEndpoint(t *testing.T) {
	diff, err := CompareSchema("/api/unknown", []byte(`{"foo": 1}`))
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "unknown" {
		t.Errorf("Status = %q, want unknown", diff.Status)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scraper/ -run TestCompareSchema -v`
Expected: FAIL — functions not defined

- [ ] **Step 3: Create schema registry**

```go
// internal/scraper/schema_registry.go
package scraper

type SchemaExpectation struct {
	TopLevel []string
	Nested   map[string][]string
}

var UpstreamSchemas = map[string]SchemaExpectation{
	"/api/worlds": {
		TopLevel: []string{"worlds", "totalOnline", "overallRecord", "overallRecordTime"},
		Nested: map[string][]string{
			"worlds[0]": {"name", "pvpType", "pvpTypeLabel", "worldType", "locked", "playersOnline"},
		},
	},
	"/api/highscores": {
		TopLevel: []string{"players", "totalCount", "cachedAt"},
		Nested: map[string][]string{
			"players[0]": {"rank", "name", "level", "vocation", "worldName", "value"},
		},
	},
	"/api/deaths": {
		TopLevel: []string{"deaths", "pagination", "canSeeDeathDetails"},
		Nested: map[string][]string{
			"deaths[0]": {"time", "level", "killed_by", "is_player", "mostdamage_by", "mostdamage_is_player", "victim", "worldName"},
		},
	},
	"/api/guilds": {
		TopLevel: []string{"guilds", "totalCount", "totalPages", "currentPage"},
		Nested: map[string][]string{
			"guilds[0]": {"name", "description", "worldName", "logo_name"},
		},
	},
	"/api/transfers": {
		TopLevel: []string{"transfers", "totalResults", "totalPages", "currentPage"},
		Nested: map[string][]string{
			"transfers[0]": {"player_name", "player_level", "from_world", "to_world", "transferred_at"},
		},
	},
	"/api/boosted": {
		TopLevel: []string{"boss", "monster"},
		Nested: map[string][]string{
			"boss":    {"id", "name", "looktype", "addons", "head", "body", "legs", "feet"},
			"monster": {"id", "name", "looktype"},
		},
	},
	"/api/maintenance": {
		TopLevel: []string{"isClosed", "closeMessage"},
	},
	"/api/news": {
		TopLevel: []string{"tickers", "articles"},
		Nested: map[string][]string{
			"tickers[0]":  {"id", "message", "category_id", "category", "author", "created_at"},
			"articles[0]": {"id", "title", "slug", "summary", "content", "cover_image", "author", "category", "published_at"},
		},
	},
	"/api/events/calendar": {
		TopLevel: []string{"events"},
		Nested: map[string][]string{
			"events[0]": {"id", "name", "description", "colorDark", "colorLight", "displayPriority", "specialEffect", "startDate", "endDate", "isRecurring", "recurringWeekdays", "recurringMonthDays", "recurringStart", "recurringEnd", "tags"},
		},
	},
	"/api/bazaar": {
		TopLevel: []string{"auctions", "pagination"},
		Nested: map[string][]string{
			"auctions[0]": {"id", "state", "stateName", "owner", "startingValue", "currentValue", "auctionStart", "auctionEnd", "name", "level", "vocation", "vocationName", "sex", "worldName", "lookType", "lookHead", "lookBody", "lookLegs", "lookFeet", "lookAddons", "direction", "charmPoints", "achievementPoints", "magLevel", "skills", "highlightItems", "highlightAugments"},
		},
	},
	"/api/bazaar/{id}": {
		TopLevel: []string{"auction", "player", "general", "items", "itemsTotal", "storeItems", "storeItemsTotal", "outfits", "mounts", "familiars", "charms", "blessings", "titles", "gems", "bosstiaries", "bosstiariosTotal", "weaponProficiency", "battlepassSeasons", "achievements", "bountyTalismans", "bountyPoints", "totalBountyPoints", "bountyRerolls", "auras", "hirelingSkills", "hirelingWardrobe", "highlightItems", "highlightAugments", "storages"},
		Nested: map[string][]string{
			"auction": {"id", "state", "stateName", "owner", "startingValue", "currentValue", "winningBid", "highestBidderId", "hasWinner", "auctionStart", "auctionEnd"},
			"general": {"health", "healthMax", "mana", "manaMax", "manaSpent", "cap", "stamina", "soul", "experience", "magLevel", "skills", "mountsCount", "outfitsCount", "titlesCount", "linkedTasks", "createDate", "balance", "totalMoney", "achievementPoints", "charmPoints", "spentCharmPoints", "availableCharmPoints", "spentMinorEchoes", "availableMinorEchoes", "charmExpansion", "streakDays", "huntingTaskPoints", "thirdPrey", "thirdHunting", "permanentWeeklyTaskSlot", "preyWildcards", "hirelingCount", "hirelingJobs", "hirelingOutfits", "dust", "dustMax", "bossPoints", "wheelPoints", "maxWheelPoints", "gpActive", "gpPoints"},
		},
	},
}
```

- [ ] **Step 4: Create schema diff logic**

```go
// internal/scraper/schema_diff.go
package scraper

import (
	"encoding/json"
	"sort"
	"strings"
)

type SchemaDiff struct {
	Endpoint      string              `json:"endpoint"`
	Status        string              `json:"status"`
	NewFields     []string            `json:"new_fields,omitempty"`
	MissingFields []string            `json:"missing_fields,omitempty"`
	NestedDiffs   map[string]FieldDiff `json:"nested_diffs,omitempty"`
}

type FieldDiff struct {
	NewFields     []string `json:"new_fields,omitempty"`
	MissingFields []string `json:"missing_fields,omitempty"`
}

func CompareSchema(endpoint string, rawJSON []byte) (*SchemaDiff, error) {
	expected, ok := UpstreamSchemas[endpoint]
	if !ok {
		return &SchemaDiff{Endpoint: endpoint, Status: "unknown"}, nil
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &parsed); err != nil {
		return nil, err
	}

	actualKeys := make([]string, 0, len(parsed))
	for k := range parsed {
		actualKeys = append(actualKeys, k)
	}

	diff := &SchemaDiff{
		Endpoint:    endpoint,
		Status:      "match",
		NestedDiffs: make(map[string]FieldDiff),
	}

	diff.NewFields = setDiff(actualKeys, expected.TopLevel)
	diff.MissingFields = setDiff(expected.TopLevel, actualKeys)

	for field, expectedNested := range expected.Nested {
		actualField := strings.TrimSuffix(field, "[0]")
		isArray := strings.HasSuffix(field, "[0]")

		raw, exists := parsed[actualField]
		if !exists {
			continue
		}

		var nestedObj map[string]json.RawMessage
		if isArray {
			var arr []json.RawMessage
			if json.Unmarshal(raw, &arr) != nil || len(arr) == 0 {
				continue
			}
			if json.Unmarshal(arr[0], &nestedObj) != nil {
				continue
			}
		} else {
			if json.Unmarshal(raw, &nestedObj) != nil {
				continue
			}
		}

		nestedKeys := make([]string, 0, len(nestedObj))
		for k := range nestedObj {
			nestedKeys = append(nestedKeys, k)
		}

		nd := FieldDiff{
			NewFields:     setDiff(nestedKeys, expectedNested),
			MissingFields: setDiff(expectedNested, nestedKeys),
		}
		if len(nd.NewFields) > 0 || len(nd.MissingFields) > 0 {
			diff.NestedDiffs[field] = nd
		}
	}

	if len(diff.NewFields) > 0 || len(diff.MissingFields) > 0 || len(diff.NestedDiffs) > 0 {
		diff.Status = "drift"
	}

	return diff, nil
}

func setDiff(a, b []string) []string {
	bSet := make(map[string]bool, len(b))
	for _, v := range b {
		bSet[v] = true
	}
	var result []string
	for _, v := range a {
		if !bSet[v] {
			result = append(result, v)
		}
	}
	sort.Strings(result)
	return result
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/scraper/ -run TestCompareSchema -v`
Expected: All 5 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scraper/schema_registry.go internal/scraper/schema_diff.go internal/scraper/schema_diff_test.go
git commit -m "feat(scraper): add upstream schema registry and drift detection

Hardcoded expected field schemas for all upstream endpoints (verified
via Playwright 2026-04-04). CompareSchema() diffs actual vs expected
at top-level and one nested level, returns new/missing fields."
```

---

### Task 4: Upstream Proxy + Schema Test Handler

**Files:**
- Modify: `internal/scraper/telemetry.go`
- Modify: `internal/api/handlers_v2.go`
- Modify: `internal/api/router_v2.go`

- [ ] **Step 1: Add schema drift Prometheus metrics**

Add to `internal/scraper/telemetry.go` — add these metrics after the existing declarations and register them in the `init()` function:

```go
var UpstreamSchemaDrift = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "rubinotdata_upstream_schema_drift",
		Help: "0=match, 1=new fields detected, -1=fields missing",
	},
	[]string{"endpoint"},
)

var UpstreamSchemaNewFieldsCount = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "rubinotdata_upstream_schema_new_fields_count",
		Help: "Number of new fields detected in upstream response",
	},
	[]string{"endpoint"},
)
```

Register both in the `init()` `prometheus.MustRegister()` call.

- [ ] **Step 2: Add upstream proxy handler to handlers_v2.go**

Add to `internal/api/handlers_v2.go`:

```go
func v2GetUpstreamRaw(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	apiPath := strings.TrimSpace(c.Param("path"))
	if apiPath == "" {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidPayload, "path required", nil)
	}

	isTest := c.Query("test") == "true"

	sourceURL := resolvedBaseURL + "/api" + apiPath
	if queryString := c.Request.URL.RawQuery; queryString != "" {
		cleaned := removeQueryParam(queryString, "test")
		if cleaned != "" {
			sourceURL = sourceURL + "?" + cleaned
		}
	}

	rawBody, err := oc.Fetcher.FetchJSON(c.Request.Context(), sourceURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	if isTest {
		schemaKey := normalizeSchemaKey(apiPath)
		diff, diffErr := scraper.CompareSchema(schemaKey, []byte(rawBody))
		if diffErr != nil {
			return endpointResult{Sources: []string{sourceURL}}, diffErr
		}

		switch diff.Status {
		case "match":
			scraper.UpstreamSchemaDrift.WithLabelValues(schemaKey).Set(0)
			scraper.UpstreamSchemaNewFieldsCount.WithLabelValues(schemaKey).Set(0)
		case "drift":
			if len(diff.MissingFields) > 0 {
				scraper.UpstreamSchemaDrift.WithLabelValues(schemaKey).Set(-1)
			} else {
				scraper.UpstreamSchemaDrift.WithLabelValues(schemaKey).Set(1)
			}
			total := len(diff.NewFields)
			for _, nd := range diff.NestedDiffs {
				total += len(nd.NewFields)
			}
			scraper.UpstreamSchemaNewFieldsCount.WithLabelValues(schemaKey).Set(float64(total))
		}

		return endpointResult{PayloadKey: "schema_test", Payload: diff, Sources: []string{sourceURL}}, nil
	}

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(rawBody), &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{PayloadKey: "upstream", Payload: raw, Sources: []string{sourceURL}}, nil
}

func normalizeSchemaKey(apiPath string) string {
	parts := strings.Split(strings.Trim(apiPath, "/"), "/")
	if len(parts) >= 1 && parts[0] == "bazaar" && len(parts) == 2 {
		return "/api/bazaar/{id}"
	}
	return "/api/" + strings.Join(parts, "/")
}

func removeQueryParam(rawQuery, param string) string {
	pairs := strings.Split(rawQuery, "&")
	var kept []string
	for _, p := range pairs {
		if !strings.HasPrefix(p, param+"=") {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "&")
}
```

- [ ] **Step 3: Register route in router_v2.go**

Add to the V2 route group in `internal/api/router_v2.go`:

```go
v2.GET("/upstream/*path", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2GetUpstreamRaw(c, oc)
}))
```

- [ ] **Step 4: Verify compilation and run tests**

Run: `go build ./... && go test ./... -v -count=1 -timeout 120s`
Expected: clean build, all tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/scraper/telemetry.go internal/api/handlers_v2.go internal/api/router_v2.go
git commit -m "feat(api): add upstream proxy and schema drift detection endpoints

GET /v2/upstream/*path — raw proxy returns unmodified upstream JSON
GET /v2/upstream/*path?test=true — compares upstream fields against
schema registry, updates Prometheus drift metrics"
```

---

### Task 5: New V2 Fetch Functions for Missing Endpoints

**Files:**
- Modify: `internal/scraper/v2_fetch.go`

This task adds V2 fetch functions for the 8 new endpoints. Each function follows the established pattern: build URL, call `oc.FetchJSON()` or `oc.BatchFetchJSON()`, map response to domain type.

- [ ] **Step 1: Add V2FetchCharactersBatch**

Add to `internal/scraper/v2_fetch.go`:

```go
func V2FetchCharactersBatch(ctx context.Context, oc *OptimizedClient, baseURL string, names []string) ([]domain.CharacterResult, []string, error) {
	paths := make([]string, len(names))
	sources := make([]string, len(names))
	for i, name := range names {
		path := fmt.Sprintf("/api/character?name=%s", url.QueryEscape(name))
		paths[i] = path
		sources[i] = fmt.Sprintf("%s%s", strings.TrimRight(baseURL, "/"), path)
	}

	tab, idx, err := oc.Fetcher.pool.Acquire(ctx)
	if err != nil {
		return nil, sources, err
	}
	defer oc.Fetcher.pool.Release(idx)

	batchResults, err := tab.BatchFetch(ctx, paths)
	if err != nil {
		return nil, sources, err
	}

	characters := make([]domain.CharacterResult, 0, len(batchResults))
	for i, br := range batchResults {
		if br.Status != "fulfilled" {
			characters = append(characters, domain.CharacterResult{Name: names[i], Error: br.Value})
			continue
		}
		trimmed := strings.TrimSpace(br.Value)
		if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
			characters = append(characters, domain.CharacterResult{Name: names[i], Error: "non-JSON response"})
			continue
		}
		var payload characterAPIResponse
		if err := parseJSONBody(trimmed, &payload); err != nil {
			characters = append(characters, domain.CharacterResult{Name: names[i], Error: err.Error()})
			continue
		}
		characters = append(characters, mapCharacterResponse(payload))
	}
	return characters, sources, nil
}
```

- [ ] **Step 2: Add V2FetchAllGuildsDetails**

```go
func V2FetchAllGuildsDetails(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int) (domain.GuildsDetailResult, []string, error) {
	guildList, listSources, err := V2FetchAllGuilds(ctx, oc, baseURL, worldName, worldID)
	if err != nil {
		return domain.GuildsDetailResult{}, listSources, err
	}

	if len(guildList.Guilds) == 0 {
		return domain.GuildsDetailResult{World: worldName, Guilds: []domain.GuildResult{}}, listSources, nil
	}

	guildURLs := make([]string, len(guildList.Guilds))
	for i, g := range guildList.Guilds {
		guildURLs[i] = fmt.Sprintf("%s/api/guild?name=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(g.Name))
	}

	batchResults, err := oc.BatchFetchJSON(ctx, guildURLs)
	if err != nil {
		return domain.GuildsDetailResult{}, listSources, err
	}

	guilds := make([]domain.GuildResult, 0, len(guildList.Guilds))
	allSources := make([]string, 0, len(listSources)+len(guildURLs))
	allSources = append(allSources, listSources...)

	for _, g := range guildList.Guilds {
		guildURL := fmt.Sprintf("%s/api/guild?name=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(g.Name))
		raw, ok := batchResults[guildURL]
		if !ok {
			continue
		}
		var payload guildAPIResponse
		if err := parseJSONBody(raw, &payload); err != nil {
			continue
		}
		guilds = append(guilds, mapGuildResponse(payload))
		allSources = append(allSources, guildURL)
	}

	return domain.GuildsDetailResult{World: worldName, Guilds: guilds}, allSources, nil
}
```

- [ ] **Step 3: Add V2FetchEventsCalendar**

```go
func V2FetchEventsCalendar(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.EventCalendarResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/events/calendar", strings.TrimRight(baseURL, "/"))
	var payload eventsCalendarAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.EventCalendarResult{}, sourceURL, err
	}
	return mapEventsCalendarResponse(payload), sourceURL, nil
}
```

Note: Requires `eventsCalendarAPIResponse` struct and `mapEventsCalendarResponse` function. Add to `internal/scraper/events.go` (or create if it doesn't exist):

```go
type eventsCalendarAPIResponse struct {
	Events []struct {
		ID                int      `json:"id"`
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		ColorDark         string   `json:"colorDark"`
		ColorLight        string   `json:"colorLight"`
		DisplayPriority   int      `json:"displayPriority"`
		SpecialEffect     string   `json:"specialEffect"`
		StartDate         string   `json:"startDate"`
		EndDate           string   `json:"endDate"`
		IsRecurring       bool     `json:"isRecurring"`
		RecurringWeekdays []int    `json:"recurringWeekdays"`
		RecurringMonthDays []int   `json:"recurringMonthDays"`
		RecurringStart    string   `json:"recurringStart"`
		RecurringEnd      string   `json:"recurringEnd"`
		Tags              []string `json:"tags"`
	} `json:"events"`
}
```

- [ ] **Step 4: Add V2FetchHighscoreCategories**

```go
func V2FetchHighscoreCategories(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.HighscoreCategoriesResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/highscores/categories", strings.TrimRight(baseURL, "/"))
	var payload highscoreCategoriesAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.HighscoreCategoriesResult{}, sourceURL, err
	}
	return mapHighscoreCategoriesResponse(payload), sourceURL, nil
}
```

- [ ] **Step 5: Add V2FetchV2AuctionDetail (full detail)**

```go
func V2FetchV2AuctionDetail(ctx context.Context, oc *OptimizedClient, baseURL string, auctionID int) (domain.V2AuctionDetail, string, error) {
	sourceURL := fmt.Sprintf("%s/api/bazaar/%d", strings.TrimRight(baseURL, "/"), auctionID)
	var payload v2AuctionDetailAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.V2AuctionDetail{}, sourceURL, err
	}
	return mapV2AuctionDetailResponse(payload), sourceURL, nil
}
```

- [ ] **Step 6: Add V2FetchCurrentAuctionDetails and V2FetchAuctionHistoryDetails**

These fetch a page of auctions, then batch-fetch full details for each:

```go
func V2FetchCurrentAuctionDetails(ctx context.Context, oc *OptimizedClient, baseURL string, page int) (domain.V2AuctionsDetailsResult, string, error) {
	return v2FetchAuctionDetailsPage(ctx, oc, baseURL, "current", page)
}

func V2FetchAuctionHistoryDetails(ctx context.Context, oc *OptimizedClient, baseURL string, page int) (domain.V2AuctionsDetailsResult, string, error) {
	return v2FetchAuctionDetailsPage(ctx, oc, baseURL, "history", page)
}

func v2FetchAuctionDetailsPage(ctx context.Context, oc *OptimizedClient, baseURL, auctionType string, page int) (domain.V2AuctionsDetailsResult, string, error) {
	listURL := buildAuctionListURL(baseURL, auctionType, page)
	var listPayload auctionListAPIResponse
	if err := oc.FetchJSON(ctx, listURL, &listPayload); err != nil {
		return domain.V2AuctionsDetailsResult{}, listURL, err
	}

	if len(listPayload.Auctions) == 0 {
		return domain.V2AuctionsDetailsResult{
			Type: auctionType, Page: page,
			TotalResults: listPayload.Pagination.Total,
			TotalPages:   listPayload.Pagination.TotalPages,
		}, listURL, nil
	}

	detailURLs := make([]string, len(listPayload.Auctions))
	for i, a := range listPayload.Auctions {
		detailURLs[i] = fmt.Sprintf("%s/api/bazaar/%d", strings.TrimRight(baseURL, "/"), a.ID)
	}

	batchResults, err := oc.BatchFetchJSON(ctx, detailURLs)
	if err != nil {
		return domain.V2AuctionsDetailsResult{}, listURL, err
	}

	entries := make([]domain.V2AuctionDetail, 0, len(listPayload.Auctions))
	for _, a := range listPayload.Auctions {
		detailURL := fmt.Sprintf("%s/api/bazaar/%d", strings.TrimRight(baseURL, "/"), a.ID)
		raw, ok := batchResults[detailURL]
		if !ok {
			continue
		}
		var detailPayload v2AuctionDetailAPIResponse
		if err := parseJSONBody(raw, &detailPayload); err != nil {
			continue
		}
		entries = append(entries, mapV2AuctionDetailResponse(detailPayload))
	}

	return domain.V2AuctionsDetailsResult{
		Type:         auctionType,
		Page:         page,
		TotalResults: listPayload.Pagination.Total,
		TotalPages:   listPayload.Pagination.TotalPages,
		Entries:      entries,
	}, listURL, nil
}
```

- [ ] **Step 7: Add V2FetchGuildsBatch and V2FetchKillstatisticsBatch**

```go
func V2FetchGuildsBatch(ctx context.Context, oc *OptimizedClient, baseURL string, names []string) ([]domain.GuildResult, []string, error) {
	paths := make([]string, len(names))
	sources := make([]string, len(names))
	for i, name := range names {
		path := fmt.Sprintf("/api/guild?name=%s", url.QueryEscape(name))
		paths[i] = path
		sources[i] = strings.TrimRight(baseURL, "/") + path
	}

	tab, idx, err := oc.Fetcher.pool.Acquire(ctx)
	if err != nil {
		return nil, sources, err
	}
	defer oc.Fetcher.pool.Release(idx)

	batchResults, err := tab.BatchFetch(ctx, paths)
	if err != nil {
		return nil, sources, err
	}

	guilds := make([]domain.GuildResult, 0, len(batchResults))
	for i, br := range batchResults {
		if br.Status != "fulfilled" {
			continue
		}
		var payload guildAPIResponse
		if err := parseJSONBody(strings.TrimSpace(br.Value), &payload); err != nil {
			continue
		}
		guilds = append(guilds, mapGuildResponse(payload))
		_ = i
	}
	return guilds, sources, nil
}

func V2FetchKillstatisticsBatch(ctx context.Context, oc *OptimizedClient, baseURL string, worldIDs []int) ([]domain.KillstatisticsResult, []string, error) {
	paths := make([]string, len(worldIDs))
	sources := make([]string, len(worldIDs))
	for i, wid := range worldIDs {
		path := fmt.Sprintf("/api/killstatistics?world=%d", wid)
		paths[i] = path
		sources[i] = strings.TrimRight(baseURL, "/") + path
	}

	tab, idx, err := oc.Fetcher.pool.Acquire(ctx)
	if err != nil {
		return nil, sources, err
	}
	defer oc.Fetcher.pool.Release(idx)

	batchResults, err := tab.BatchFetch(ctx, paths)
	if err != nil {
		return nil, sources, err
	}

	results := make([]domain.KillstatisticsResult, 0, len(batchResults))
	for _, br := range batchResults {
		if br.Status != "fulfilled" {
			continue
		}
		var payload killstatisticsAPIResponse
		if err := parseJSONBody(strings.TrimSpace(br.Value), &payload); err != nil {
			continue
		}
		results = append(results, mapKillstatisticsResponse(payload))
	}
	return results, sources, nil
}
```

- [ ] **Step 8: Verify compilation**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 9: Commit**

```bash
git add internal/scraper/v2_fetch.go
git commit -m "feat(scraper): add V2 fetch functions for all missing endpoints

Characters batch, guilds batch, killstats batch (via CDPPool tab),
guild details (list + batch detail), events calendar, highscore
categories, auction detail pages (list + batch enrich)."
```

---

### Task 6: V2 Handlers + Route Registration

**Files:**
- Modify: `internal/api/handlers_v2.go`
- Modify: `internal/api/router_v2.go`

- [ ] **Step 1: Add handler functions to handlers_v2.go**

Add each handler following the existing pattern (validate input → call scraper → return endpointResult):

```go
func v2PostCharactersBatch(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	var req struct {
		Names []string `json:"names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidPayload, "invalid batch request", err)
	}
	if len(req.Names) == 0 || len(req.Names) > 500 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidPayload, "names must be 1-500", nil)
	}
	results, sources, err := scraper.V2FetchCharactersBatch(c.Request.Context(), oc, resolvedBaseURL, req.Names)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{PayloadKey: "characters", Payload: results, Sources: sources}, nil
}

func v2GetAllGuildsDetails(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}
	result, sources, err := scraper.V2FetchAllGuildsDetails(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{PayloadKey: "guilds", Payload: result, Sources: sources}, nil
}

func v2GetEventsCalendar(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	result, sourceURL, err := scraper.V2FetchEventsCalendar(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{PayloadKey: "events", Payload: result, Sources: []string{sourceURL}}, nil
}

func v2GetHighscoreCategories(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	result, sourceURL, err := scraper.V2FetchHighscoreCategories(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{PayloadKey: "categories", Payload: result, Sources: []string{sourceURL}}, nil
}

func v2GetCurrentAuctionDetails(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	page, err := strconv.Atoi(c.Param("page"))
	if err != nil || page < 1 {
		page = 1
	}
	result, sourceURL, fetchErr := scraper.V2FetchCurrentAuctionDetails(c.Request.Context(), oc, resolvedBaseURL, page)
	if fetchErr != nil {
		return endpointResult{Sources: []string{sourceURL}}, fetchErr
	}
	return endpointResult{PayloadKey: "auctions", Payload: result, Sources: []string{sourceURL}}, nil
}

func v2GetAuctionHistoryDetails(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	page, err := strconv.Atoi(c.Param("page"))
	if err != nil || page < 1 {
		page = 1
	}
	result, sourceURL, fetchErr := scraper.V2FetchAuctionHistoryDetails(c.Request.Context(), oc, resolvedBaseURL, page)
	if fetchErr != nil {
		return endpointResult{Sources: []string{sourceURL}}, fetchErr
	}
	return endpointResult{PayloadKey: "auctions", Payload: result, Sources: []string{sourceURL}}, nil
}

func v2PostGuildsBatch(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	var req struct {
		Names []string `json:"names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidPayload, "invalid batch request", err)
	}
	if len(req.Names) == 0 || len(req.Names) > 200 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidPayload, "names must be 1-200", nil)
	}
	results, sources, err := scraper.V2FetchGuildsBatch(c.Request.Context(), oc, resolvedBaseURL, req.Names)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{PayloadKey: "guilds", Payload: results, Sources: sources}, nil
}

func v2PostKillstatisticsBatch(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	var req struct {
		WorldIDs []int `json:"world_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidPayload, "invalid batch request", err)
	}
	if len(req.WorldIDs) == 0 || len(req.WorldIDs) > 50 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidPayload, "world_ids must be 1-50", nil)
	}
	results, sources, err := scraper.V2FetchKillstatisticsBatch(c.Request.Context(), oc, resolvedBaseURL, req.WorldIDs)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{PayloadKey: "killstatistics", Payload: results, Sources: sources}, nil
}

func v2GetV2AuctionDetail(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	auctionID, err := validation.ParseAuctionID(c.Param("id"))
	if err != nil {
		return endpointResult{}, err
	}
	result, sourceURL, fetchErr := scraper.V2FetchV2AuctionDetail(c.Request.Context(), oc, resolvedBaseURL, auctionID)
	if fetchErr != nil {
		return endpointResult{Sources: []string{sourceURL}}, fetchErr
	}
	return endpointResult{PayloadKey: "auction", Payload: result, Sources: []string{sourceURL}}, nil
}
```

- [ ] **Step 2: Register routes in router_v2.go**

Add to the V2 route group:

```go
v2.POST("/characters/batch", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2PostCharactersBatch(c, oc)
}))
v2.GET("/guilds/:world/all/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2GetAllGuildsDetails(c, getValidator(), oc)
}))
v2.GET("/events/calendar", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2GetEventsCalendar(c, oc)
}))
v2.GET("/highscores/categories", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2GetHighscoreCategories(c, oc)
}))
v2.GET("/auctions/current/:page/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2GetCurrentAuctionDetails(c, oc)
}))
v2.GET("/auctions/history/:page/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2GetAuctionHistoryDetails(c, oc)
}))
v2.GET("/auctions/:id/full", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2GetV2AuctionDetail(c, oc)
}))
v2.POST("/guilds/batch", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2PostGuildsBatch(c, oc)
}))
v2.POST("/killstatistics/batch", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
	return v2PostKillstatisticsBatch(c, oc)
}))
```

Note: The full V2 auction detail is at `/v2/auctions/:id/full` to avoid conflicting with the existing `/v2/auctions/:id` which returns the V1-compatible detail. rubinot-api's enrichment processor will call `/full`.

- [ ] **Step 3: Build and run all tests**

Run: `go build ./... && go test ./... -v -count=1 -timeout 120s`
Expected: clean build, all tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/api/handlers_v2.go internal/api/router_v2.go
git commit -m "feat(api): register 10 new V2 endpoints

POST /v2/characters/batch, GET /v2/guilds/:world/all/details,
GET /v2/events/calendar, GET /v2/highscores/categories,
GET /v2/auctions/current/:page/details, GET /v2/auctions/history/:page/details,
GET /v2/auctions/:id/full, POST /v2/guilds/batch,
POST /v2/killstatistics/batch, GET /v2/upstream/*path"
```

---

### Task 7: Expand Events Domain Type

**Files:**
- Modify: `internal/domain/events.go`

- [ ] **Step 1: Add EventCalendarResult and Event types**

Update `internal/domain/events.go` to include the full upstream event schema:

```go
type Event struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	ColorDark         string   `json:"color_dark"`
	ColorLight        string   `json:"color_light"`
	DisplayPriority   int      `json:"display_priority"`
	SpecialEffect     string   `json:"special_effect"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
	IsRecurring       bool     `json:"is_recurring"`
	RecurringWeekdays  []int   `json:"recurring_weekdays,omitempty"`
	RecurringMonthDays []int   `json:"recurring_month_days,omitempty"`
	RecurringStart     string  `json:"recurring_start,omitempty"`
	RecurringEnd       string  `json:"recurring_end,omitempty"`
	Tags               []string `json:"tags,omitempty"`
}

type EventCalendarResult struct {
	Events []Event `json:"events"`
}
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add internal/domain/events.go
git commit -m "feat(domain): expand Event type to all 15 upstream fields"
```

---

### Task 8: Tag and Deploy

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v -count=1 -timeout 120s`
Expected: all tests pass

- [ ] **Step 2: Run linter**

Run: `go vet ./...`
Expected: clean

- [ ] **Step 3: Tag and push**

```bash
git tag v2.6.0
git push origin main
git push origin v2.6.0
```

- [ ] **Step 4: Monitor CI build**

Run: `gh run list --repo rubinot-lab/rubinot-data --limit 1`
Wait for build to complete and ArgoCD to sync.

- [ ] **Step 5: Verify new endpoints via port-forward**

```bash
kubectl -n rubinot port-forward deployment/rubinot-data 18085:8080
curl -s http://localhost:18085/v2/highscores/categories | head -c 200
curl -s http://localhost:18085/v2/events/calendar | head -c 200
curl -s http://localhost:18085/v2/upstream/worlds?test=true | python3 -m json.tool
curl -s http://localhost:18085/versions
```

Expected: v2.6.0, all endpoints return data, schema test returns "match" status.

---

### Task 9: PR Review + Cleanup

- [ ] **Step 1: Review all changes in the branch**

```bash
git diff v2.5.4..HEAD --stat
git diff v2.5.4..HEAD
```

Review as another engineer would: check types match across files, mapping is complete, no dropped fields, tests cover the new code.

- [ ] **Step 2: Apply any recommended fixes**

Fix issues found in review, re-run tests.

- [ ] **Step 3: Remove unnecessary code comments**

Scan all changed files for comments that are self-explanatory by the code.

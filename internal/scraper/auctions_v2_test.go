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
		"weaponProficiency": [{"itemId": 1, "experience": 5000, "weaponLevel": 3, "masteryAchieved": false, "activePerks": [{"lane": 0, "index": 1}, {"lane": 1, "index": 0}]}],
		"battlepassSeasons": [{"season": 1, "points": 500, "active": 1, "shoppoints": 100, "steps": []}],
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
	if result.State != 1 {
		t.Errorf("State = %d, want 1", result.State)
	}
	if result.StateName != "Active" {
		t.Errorf("StateName = %q, want %q", result.StateName, "Active")
	}
	if result.HasWinner != false {
		t.Errorf("HasWinner = %v, want false", result.HasWinner)
	}
	if result.StartingValue != 1000 {
		t.Errorf("StartingValue = %d, want 1000", result.StartingValue)
	}
	if result.CurrentValue != 1001 {
		t.Errorf("CurrentValue = %d, want 1001", result.CurrentValue)
	}
	if result.WinningBid != 1001 {
		t.Errorf("WinningBid = %d, want 1001", result.WinningBid)
	}
	if result.AuctionStart == "" {
		t.Error("AuctionStart is empty")
	}
	if result.AuctionEnd == "" {
		t.Error("AuctionEnd is empty")
	}
	if result.Status != "Active" {
		t.Errorf("Status = %q, want %q", result.Status, "Active")
	}

	if result.CharacterName != "Test Knight" {
		t.Errorf("CharacterName = %q, want %q", result.CharacterName, "Test Knight")
	}
	if result.Level != 500 {
		t.Errorf("Level = %d, want 500", result.Level)
	}
	if result.Vocation != "Elite Knight" {
		t.Errorf("Vocation = %q, want %q", result.Vocation, "Elite Knight")
	}
	if result.VocationID != 1 {
		t.Errorf("VocationID = %d, want 1", result.VocationID)
	}
	if result.Sex != "Male" {
		t.Errorf("Sex = %q, want Male", result.Sex)
	}
	if result.World != "Auroria" {
		t.Errorf("World = %q, want %q", result.World, "Auroria")
	}

	if result.Outfit.LookType != 129 {
		t.Errorf("Outfit.LookType = %d, want 129", result.Outfit.LookType)
	}
	if result.Outfit.LookHead != 78 {
		t.Errorf("Outfit.LookHead = %d, want 78", result.Outfit.LookHead)
	}
	if result.Outfit.LookAddons != 3 {
		t.Errorf("Outfit.LookAddons = %d, want 3", result.Outfit.LookAddons)
	}
	if result.Outfit.Direction != 3 {
		t.Errorf("Outfit.Direction = %d, want 3", result.Outfit.Direction)
	}

	if result.General.Health != 4000 {
		t.Errorf("General.Health = %d, want 4000", result.General.Health)
	}
	if result.General.HealthMax != 4000 {
		t.Errorf("General.HealthMax = %d, want 4000", result.General.HealthMax)
	}
	if result.General.Mana != 2000 {
		t.Errorf("General.Mana = %d, want 2000", result.General.Mana)
	}
	if result.General.ManaSpent != 500000 {
		t.Errorf("General.ManaSpent = %d, want 500000", result.General.ManaSpent)
	}
	if result.General.Cap != 1520 {
		t.Errorf("General.Cap = %d, want 1520", result.General.Cap)
	}
	if result.General.Stamina != 2520 {
		t.Errorf("General.Stamina = %d, want 2520", result.General.Stamina)
	}
	if result.General.Soul != 200 {
		t.Errorf("General.Soul = %d, want 200", result.General.Soul)
	}
	if result.General.Experience != 14000000000 {
		t.Errorf("General.Experience = %d, want 14000000000", result.General.Experience)
	}
	if result.General.MagLevel != 15 {
		t.Errorf("General.MagLevel = %d, want 15", result.General.MagLevel)
	}
	if result.General.Skills.Axe != 120 {
		t.Errorf("General.Skills.Axe = %d, want 120", result.General.Skills.Axe)
	}
	if result.General.Skills.Club != 30 {
		t.Errorf("General.Skills.Club = %d, want 30", result.General.Skills.Club)
	}
	if result.General.Skills.Sword != 30 {
		t.Errorf("General.Skills.Sword = %d, want 30", result.General.Skills.Sword)
	}
	if result.General.Skills.Distance != 30 {
		t.Errorf("General.Skills.Distance = %d, want 30", result.General.Skills.Distance)
	}
	if result.General.Skills.Shielding != 115 {
		t.Errorf("General.Skills.Shielding = %d, want 115", result.General.Skills.Shielding)
	}
	if result.General.Skills.Fishing != 20 {
		t.Errorf("General.Skills.Fishing = %d, want 20", result.General.Skills.Fishing)
	}
	if result.General.Skills.Fist != 15 {
		t.Errorf("General.Skills.Fist = %d, want 15", result.General.Skills.Fist)
	}
	if result.General.MountsCount != 25 {
		t.Errorf("General.MountsCount = %d, want 25", result.General.MountsCount)
	}
	if result.General.OutfitsCount != 35 {
		t.Errorf("General.OutfitsCount = %d, want 35", result.General.OutfitsCount)
	}
	if result.General.TitlesCount != 10 {
		t.Errorf("General.TitlesCount = %d, want 10", result.General.TitlesCount)
	}
	if result.General.LinkedTasks != 5 {
		t.Errorf("General.LinkedTasks = %d, want 5", result.General.LinkedTasks)
	}
	if result.General.CreateDate == "" {
		t.Error("General.CreateDate is empty")
	}
	if result.General.TotalMoney != 50000000 {
		t.Errorf("General.TotalMoney = %d, want 50000000", result.General.TotalMoney)
	}
	if result.General.AchievementPoints != 420 {
		t.Errorf("General.AchievementPoints = %d, want 420", result.General.AchievementPoints)
	}
	if result.General.CharmPoints != 6500 {
		t.Errorf("General.CharmPoints = %d, want 6500", result.General.CharmPoints)
	}
	if result.General.SpentCharmPoints != 4000 {
		t.Errorf("General.SpentCharmPoints = %d, want 4000", result.General.SpentCharmPoints)
	}
	if result.General.AvailableCharmPoints != 2500 {
		t.Errorf("General.AvailableCharmPoints = %d, want 2500", result.General.AvailableCharmPoints)
	}
	if result.General.CharmExpansion != true {
		t.Errorf("General.CharmExpansion = %v, want true", result.General.CharmExpansion)
	}
	if result.General.StreakDays != 30 {
		t.Errorf("General.StreakDays = %d, want 30", result.General.StreakDays)
	}
	if result.General.HuntingTaskPoints != 100 {
		t.Errorf("General.HuntingTaskPoints = %d, want 100", result.General.HuntingTaskPoints)
	}
	if result.General.PreyWildcards != 5 {
		t.Errorf("General.PreyWildcards = %d, want 5", result.General.PreyWildcards)
	}
	if result.General.HirelingCount != 2 {
		t.Errorf("General.HirelingCount = %d, want 2", result.General.HirelingCount)
	}
	if result.General.HirelingJobs != 3 {
		t.Errorf("General.HirelingJobs = %d, want 3", result.General.HirelingJobs)
	}
	if result.General.HirelingOutfits != 1 {
		t.Errorf("General.HirelingOutfits = %d, want 1", result.General.HirelingOutfits)
	}
	if result.General.Dust != 50 {
		t.Errorf("General.Dust = %d, want 50", result.General.Dust)
	}
	if result.General.DustMax != 100 {
		t.Errorf("General.DustMax = %d, want 100", result.General.DustMax)
	}
	if result.General.BossPoints != 8500 {
		t.Errorf("General.BossPoints = %d, want 8500", result.General.BossPoints)
	}
	if result.General.WheelPoints != 1000 {
		t.Errorf("General.WheelPoints = %d, want 1000", result.General.WheelPoints)
	}
	if result.General.MaxWheelPoints != 2000 {
		t.Errorf("General.MaxWheelPoints = %d, want 2000", result.General.MaxWheelPoints)
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
	if result.Items[0].SlotID != 4 {
		t.Errorf("Items[0].SlotID = %d, want 4", result.Items[0].SlotID)
	}
	if result.Items[0].ClientID != 3366 {
		t.Errorf("Items[0].ClientID = %d, want 3366", result.Items[0].ClientID)
	}
	if result.Items[0].Description != "Arm:17" {
		t.Errorf("Items[0].Description = %q, want %q", result.Items[0].Description, "Arm:17")
	}
	if result.ItemsTotal != 1 {
		t.Errorf("ItemsTotal = %d, want 1", result.ItemsTotal)
	}

	if len(result.StoreItems) != 0 {
		t.Errorf("StoreItems len = %d, want 0", len(result.StoreItems))
	}
	if result.StoreItemsTotal != 0 {
		t.Errorf("StoreItemsTotal = %d, want 0", result.StoreItemsTotal)
	}

	if len(result.Outfits) != 1 {
		t.Fatalf("Outfits len = %d, want 1", len(result.Outfits))
	}
	if result.Outfits[0].ID != 131 {
		t.Errorf("Outfits[0].ID = %d, want 131", result.Outfits[0].ID)
	}
	if result.Outfits[0].Addons != 3 {
		t.Errorf("Outfits[0].Addons = %d, want 3", result.Outfits[0].Addons)
	}
	if result.Outfits[0].Info.LookType != 131 {
		t.Errorf("Outfits[0].Info.LookType = %d, want 131", result.Outfits[0].Info.LookType)
	}
	if result.Outfits[0].Info.Name != "Citizen" {
		t.Errorf("Outfits[0].Info.Name = %q, want %q", result.Outfits[0].Info.Name, "Citizen")
	}

	if len(result.Mounts) != 1 {
		t.Fatalf("Mounts len = %d, want 1", len(result.Mounts))
	}
	if result.Mounts[0].ID != 1 {
		t.Errorf("Mounts[0].ID = %d, want 1", result.Mounts[0].ID)
	}
	if result.Mounts[0].Name != "Widow Queen" {
		t.Errorf("Mounts[0].Name = %q, want %q", result.Mounts[0].Name, "Widow Queen")
	}
	if result.Mounts[0].ClientID != 368 {
		t.Errorf("Mounts[0].ClientID = %d, want 368", result.Mounts[0].ClientID)
	}

	if len(result.Familiars) != 0 {
		t.Errorf("Familiars len = %d, want 0", len(result.Familiars))
	}

	if len(result.Charms) != 1 {
		t.Fatalf("Charms len = %d, want 1", len(result.Charms))
	}
	if result.Charms[0].ID != 1 {
		t.Errorf("Charms[0].ID = %d, want 1", result.Charms[0].ID)
	}
	if result.Charms[0].RaceID != 100 {
		t.Errorf("Charms[0].RaceID = %d, want 100", result.Charms[0].RaceID)
	}
	if result.Charms[0].Type != 1 {
		t.Errorf("Charms[0].Type = %d, want 1", result.Charms[0].Type)
	}

	if len(result.Blessings) != 1 {
		t.Fatalf("Blessings len = %d, want 1", len(result.Blessings))
	}
	if result.Blessings[0].Name != "Twist of Fate" {
		t.Errorf("Blessings[0].Name = %q, want %q", result.Blessings[0].Name, "Twist of Fate")
	}
	if result.Blessings[0].Count != 1 {
		t.Errorf("Blessings[0].Count = %d, want 1", result.Blessings[0].Count)
	}

	if len(result.Titles) != 3 {
		t.Fatalf("Titles len = %d, want 3", len(result.Titles))
	}
	if result.Titles[0] != 1 || result.Titles[1] != 5 || result.Titles[2] != 10 {
		t.Errorf("Titles = %v, want [1 5 10]", result.Titles)
	}

	if len(result.Gems) != 0 {
		t.Errorf("Gems len = %d, want 0", len(result.Gems))
	}

	if len(result.Bosstiaries) != 1 {
		t.Fatalf("Bosstiaries len = %d, want 1", len(result.Bosstiaries))
	}
	if result.Bosstiaries[0].ID != 1 {
		t.Errorf("Bosstiaries[0].ID = %d, want 1", result.Bosstiaries[0].ID)
	}
	if result.Bosstiaries[0].Name != "Orshabaal" {
		t.Errorf("Bosstiaries[0].Name = %q, want %q", result.Bosstiaries[0].Name, "Orshabaal")
	}
	if result.Bosstiaries[0].Kills != 5 {
		t.Errorf("Bosstiaries[0].Kills = %d, want 5", result.Bosstiaries[0].Kills)
	}
	if result.Bosstiaries[0].Gained1 != 1 {
		t.Errorf("Bosstiaries[0].Gained1 = %d, want 1", result.Bosstiaries[0].Gained1)
	}
	if result.BosstiariosTotal != 1 {
		t.Errorf("BosstiariosTotal = %d, want 1", result.BosstiariosTotal)
	}

	if len(result.WeaponProficiency) != 1 {
		t.Fatalf("WeaponProficiency len = %d, want 1", len(result.WeaponProficiency))
	}
	if result.WeaponProficiency[0].ItemID != 1 {
		t.Errorf("WeaponProficiency[0].ItemID = %d, want 1", result.WeaponProficiency[0].ItemID)
	}
	if result.WeaponProficiency[0].Experience != 5000 {
		t.Errorf("WeaponProficiency[0].Experience = %d, want 5000", result.WeaponProficiency[0].Experience)
	}
	if result.WeaponProficiency[0].WeaponLevel != 3 {
		t.Errorf("WeaponProficiency[0].WeaponLevel = %d, want 3", result.WeaponProficiency[0].WeaponLevel)
	}
	if result.WeaponProficiency[0].MasteryAchieved != false {
		t.Errorf("WeaponProficiency[0].MasteryAchieved = %v, want false", result.WeaponProficiency[0].MasteryAchieved)
	}
	if len(result.WeaponProficiency[0].ActivePerks) != 2 {
		t.Errorf("WeaponProficiency[0].ActivePerks len = %d, want 2", len(result.WeaponProficiency[0].ActivePerks))
	}
	if result.WeaponProficiency[0].ActivePerks[0].Lane != 0 || result.WeaponProficiency[0].ActivePerks[0].Index != 1 {
		t.Errorf("WeaponProficiency[0].ActivePerks[0] = %+v", result.WeaponProficiency[0].ActivePerks[0])
	}

	if len(result.BattlepassSeasons) != 1 {
		t.Fatalf("BattlepassSeasons len = %d, want 1", len(result.BattlepassSeasons))
	}
	if result.BattlepassSeasons[0].Season != 1 {
		t.Errorf("BattlepassSeasons[0].Season = %d, want 1", result.BattlepassSeasons[0].Season)
	}
	if result.BattlepassSeasons[0].Points != 500 {
		t.Errorf("BattlepassSeasons[0].Points = %d, want 500", result.BattlepassSeasons[0].Points)
	}
	if result.BattlepassSeasons[0].Active != 1 {
		t.Errorf("BattlepassSeasons[0].Active = %d, want 1", result.BattlepassSeasons[0].Active)
	}
	if result.BattlepassSeasons[0].ShopPoints != 100 {
		t.Errorf("BattlepassSeasons[0].ShopPoints = %d, want 100", result.BattlepassSeasons[0].ShopPoints)
	}
	if len(result.BattlepassSeasons[0].Steps) != 0 {
		t.Errorf("BattlepassSeasons[0].Steps len = %d, want 0", len(result.BattlepassSeasons[0].Steps))
	}

	if len(result.Achievements) != 1 {
		t.Fatalf("Achievements len = %d, want 1", len(result.Achievements))
	}
	if result.Achievements[0].ID != 1 {
		t.Errorf("Achievements[0].ID = %d, want 1", result.Achievements[0].ID)
	}
	if result.Achievements[0].UnlockedAt != 1700000000 {
		t.Errorf("Achievements[0].UnlockedAt = %d, want 1700000000", result.Achievements[0].UnlockedAt)
	}

	if len(result.BountyTalismans) != 1 {
		t.Fatalf("BountyTalismans len = %d, want 1", len(result.BountyTalismans))
	}
	if result.BountyTalismans[0].Type != 1 {
		t.Errorf("BountyTalismans[0].Type = %d, want 1", result.BountyTalismans[0].Type)
	}
	if result.BountyTalismans[0].Level != 2 {
		t.Errorf("BountyTalismans[0].Level = %d, want 2", result.BountyTalismans[0].Level)
	}
	if result.BountyTalismans[0].EffectValue != 10 {
		t.Errorf("BountyTalismans[0].EffectValue = %d, want 10", result.BountyTalismans[0].EffectValue)
	}

	if result.BountyPoints != 100 {
		t.Errorf("BountyPoints = %d, want 100", result.BountyPoints)
	}
	if result.TotalBountyPoints != 500 {
		t.Errorf("TotalBountyPoints = %d, want 500", result.TotalBountyPoints)
	}
	if result.BountyRerolls != 3 {
		t.Errorf("BountyRerolls = %d, want 3", result.BountyRerolls)
	}

	if len(result.Auras) != 0 {
		t.Errorf("Auras len = %d, want 0", len(result.Auras))
	}
	if len(result.HirelingSkills) != 0 {
		t.Errorf("HirelingSkills len = %d, want 0", len(result.HirelingSkills))
	}
	if len(result.HirelingWardrobe) != 0 {
		t.Errorf("HirelingWardrobe len = %d, want 0", len(result.HirelingWardrobe))
	}

	if len(result.HighlightItems) != 1 {
		t.Fatalf("HighlightItems len = %d, want 1", len(result.HighlightItems))
	}
	if result.HighlightItems[0].ID != 3366 {
		t.Errorf("HighlightItems[0].ID = %d, want 3366", result.HighlightItems[0].ID)
	}
	if result.HighlightItems[0].Name != "Magic Plate Armor" {
		t.Errorf("HighlightItems[0].Name = %q, want %q", result.HighlightItems[0].Name, "Magic Plate Armor")
	}

	if len(result.HighlightAugments) != 1 {
		t.Fatalf("HighlightAugments len = %d, want 1", len(result.HighlightAugments))
	}
	if result.HighlightAugments[0].ID != 0 {
		t.Errorf("HighlightAugments[0].ID = %d, want 0", result.HighlightAugments[0].ID)
	}
	if result.HighlightAugments[0].Name != "120 axe fighting" {
		t.Errorf("HighlightAugments[0].Name = %q, want %q", result.HighlightAugments[0].Name, "120 axe fighting")
	}
}

func TestMapV2AuctionDetailResponse_DistFallback(t *testing.T) {
	raw := `{
		"auction": {"id": 1, "state": 0, "stateName": "Finished", "owner": 1, "startingValue": 100, "currentValue": 200, "winningBid": 200, "highestBidderId": 0, "hasWinner": true, "auctionStart": 1700000000, "auctionEnd": 1700100000},
		"player": {"name": "Paladin Test", "level": 100, "vocation": 3, "vocationName": "Royal Paladin", "sex": 0, "worldName": "Secura", "lookType": 130, "lookHead": 0, "lookBody": 0, "lookLegs": 0, "lookFeet": 0, "lookAddons": 0, "direction": 0, "lookMount": 0},
		"general": {"health": 1000, "healthMax": 1000, "mana": 500, "manaMax": 500, "manaSpent": 0, "cap": 500, "stamina": 2520, "soul": 200, "experience": 1000000, "magLevel": 10, "skills": {"axe": 10, "club": 10, "sword": 10, "distance": 0, "dist": 95, "shielding": 80, "fishing": 10, "fist": 10, "magic": 10}, "mountsCount": 0, "outfitsCount": 0, "titlesCount": 0, "linkedTasks": 0, "createDate": 1700000000, "balance": 0, "totalMoney": 0, "achievementPoints": 0, "charmPoints": 0, "spentCharmPoints": 0, "availableCharmPoints": 0, "spentMinorEchoes": 0, "availableMinorEchoes": 0, "charmExpansion": false, "streakDays": 0, "huntingTaskPoints": 0, "thirdPrey": false, "thirdHunting": false, "permanentWeeklyTaskSlot": false, "preyWildcards": 0, "hirelingCount": 0, "hirelingJobs": 0, "hirelingOutfits": 0, "dust": 0, "dustMax": 0, "bossPoints": 0, "wheelPoints": 0, "maxWheelPoints": 0, "gpActive": false, "gpPoints": 0},
		"items": [], "itemsTotal": 0, "storeItems": [], "storeItemsTotal": 0,
		"outfits": [], "mounts": [], "familiars": [], "charms": [], "blessings": [],
		"titles": [], "gems": [], "bosstiaries": [], "bosstiariosTotal": 0,
		"weaponProficiency": [], "battlepassSeasons": [], "achievements": [],
		"bountyTalismans": [], "bountyPoints": 0, "totalBountyPoints": 0, "bountyRerolls": 0,
		"auras": [], "hirelingSkills": [], "hirelingWardrobe": [],
		"highlightItems": [], "highlightAugments": []
	}`

	var payload v2AuctionDetailAPIResponse
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result := mapV2AuctionDetailResponse(payload)

	if result.General.Skills.Distance != 95 {
		t.Errorf("Distance fallback: got %d, want 95 (from dist field)", result.General.Skills.Distance)
	}
	if result.Sex != "Female" {
		t.Errorf("Sex = %q, want Female", result.Sex)
	}
	if result.HasWinner != true {
		t.Errorf("HasWinner = %v, want true", result.HasWinner)
	}
}

func TestMapV2AuctionDetailResponse_HighlightItemClientIDFallback(t *testing.T) {
	raw := `{
		"auction": {"id": 2, "state": 0, "stateName": "Finished", "owner": 1, "startingValue": 100, "currentValue": 200, "winningBid": 200, "highestBidderId": 0, "hasWinner": false, "auctionStart": 1700000000, "auctionEnd": 1700100000},
		"player": {"name": "Test", "level": 50, "vocation": 0, "vocationName": "None", "sex": 2, "worldName": "Antica", "lookType": 128, "lookHead": 0, "lookBody": 0, "lookLegs": 0, "lookFeet": 0, "lookAddons": 0, "direction": 0, "lookMount": 0},
		"general": {"health": 500, "healthMax": 500, "mana": 250, "manaMax": 250, "manaSpent": 0, "cap": 300, "stamina": 2520, "soul": 200, "experience": 100000, "magLevel": 5, "skills": {"axe": 10, "club": 10, "sword": 10, "distance": 10, "shielding": 10, "fishing": 10, "fist": 10, "magic": 10}, "mountsCount": 0, "outfitsCount": 0, "titlesCount": 0, "linkedTasks": 0, "createDate": 1700000000, "balance": 0, "totalMoney": 0, "achievementPoints": 0, "charmPoints": 0, "spentCharmPoints": 0, "availableCharmPoints": 0, "spentMinorEchoes": 0, "availableMinorEchoes": 0, "charmExpansion": false, "streakDays": 0, "huntingTaskPoints": 0, "thirdPrey": false, "thirdHunting": false, "permanentWeeklyTaskSlot": false, "preyWildcards": 0, "hirelingCount": 0, "hirelingJobs": 0, "hirelingOutfits": 0, "dust": 0, "dustMax": 0, "bossPoints": 0, "wheelPoints": 0, "maxWheelPoints": 0, "gpActive": false, "gpPoints": 0},
		"items": [], "itemsTotal": 0, "storeItems": [], "storeItemsTotal": 0,
		"outfits": [], "mounts": [], "familiars": [], "charms": [], "blessings": [],
		"titles": [], "gems": [], "bosstiaries": [], "bosstiariosTotal": 0,
		"weaponProficiency": [], "battlepassSeasons": [], "achievements": [],
		"bountyTalismans": [], "bountyPoints": 0, "totalBountyPoints": 0, "bountyRerolls": 0,
		"auras": [], "hirelingSkills": [], "hirelingWardrobe": [],
		"highlightItems": [{"itemId": 0, "clientId": 9999, "tier": 0, "count": 1, "name": "Fallback Item"}],
		"highlightAugments": []
	}`

	var payload v2AuctionDetailAPIResponse
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result := mapV2AuctionDetailResponse(payload)

	if len(result.HighlightItems) != 1 {
		t.Fatalf("HighlightItems len = %d, want 1", len(result.HighlightItems))
	}
	if result.HighlightItems[0].ID != 9999 {
		t.Errorf("HighlightItems[0].ID = %d, want 9999 (clientId fallback)", result.HighlightItems[0].ID)
	}
	if result.Sex != "Unknown" {
		t.Errorf("Sex = %q, want Unknown (sex=2)", result.Sex)
	}
}

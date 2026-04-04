package scraper

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/giovannirco/rubinot-data/internal/domain"
)

type flexInt int

func (fi *flexInt) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*fi = flexInt(n)
		return nil
	}
	var bl bool
	if err := json.Unmarshal(b, &bl); err == nil {
		if bl {
			*fi = 1
		} else {
			*fi = 0
		}
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v, _ := strconv.Atoi(s)
		*fi = flexInt(v)
		return nil
	}
	*fi = 0
	return nil
}

type flexInt64 int64

func (fi *flexInt64) UnmarshalJSON(b []byte) error {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		*fi = flexInt64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v, _ := strconv.ParseInt(s, 10, 64)
		*fi = flexInt64(v)
		return nil
	}
	*fi = 0
	return nil
}

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
		ManaSpent            flexInt64 `json:"manaSpent"`
		Cap                  int         `json:"cap"`
		Stamina              int         `json:"stamina"`
		Soul                 int         `json:"soul"`
		Experience           flexInt64 `json:"experience"`
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
		Balance              flexInt64 `json:"balance"`
		TotalMoney           flexInt64 `json:"totalMoney"`
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
	Outfits         []struct {
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
		ID     flexInt `json:"id"`
		Tier   flexInt `json:"tier"`
		RaceID flexInt `json:"raceId"`
		Type   flexInt `json:"type"`
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
		ID      flexInt `json:"id"`
		Name    string  `json:"name"`
		Kills   flexInt `json:"kills"`
		Gained1 flexInt `json:"gained1"`
		Gained2 flexInt `json:"gained2"`
		Gained3 flexInt `json:"gained3"`
	} `json:"bosstiaries"`
	BosstiariosTotal  int `json:"bosstiariosTotal"`
	WeaponProficiency []struct {
		ItemID          int             `json:"itemId"`
		Experience      int             `json:"experience"`
		WeaponLevel     int             `json:"weaponLevel"`
		MasteryAchieved bool            `json:"masteryAchieved"`
		ActivePerks     json.RawMessage `json:"activePerks"`
	} `json:"weaponProficiency"`
	BattlepassSeasons []struct {
		Season     int             `json:"season"`
		Points     int             `json:"points"`
		Active     flexInt         `json:"active"`
		ShopPoints int             `json:"shoppoints"`
		Steps      json.RawMessage `json:"steps"`
	} `json:"battlepassSeasons"`
	Achievements []struct {
		ID         int   `json:"id"`
		UnlockedAt int64 `json:"unlockedAt"`
	} `json:"achievements"`
	BountyTalismans []struct {
		Type        flexInt `json:"type"`
		Level       flexInt `json:"level"`
		EffectValue flexInt `json:"effectValue"`
	} `json:"bountyTalismans"`
	BountyPoints      int `json:"bountyPoints"`
	TotalBountyPoints int `json:"totalBountyPoints"`
	BountyRerolls     int `json:"bountyRerolls"`
	Auras             []struct {
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
		charms = append(charms, domain.V2AuctionCharm{ID: int(ch.ID), Tier: int(ch.Tier), RaceID: int(ch.RaceID), Type: int(ch.Type)})
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
			ID: int(b.ID), Name: b.Name, Kills: int(b.Kills), Gained1: int(b.Gained1), Gained2: int(b.Gained2), Gained3: int(b.Gained3),
		})
	}

	weaponProf := make([]domain.V2AuctionWeaponProf, 0, len(p.WeaponProficiency))
	for _, w := range p.WeaponProficiency {
		var perks []domain.V2AuctionWeaponPerk
		_ = json.Unmarshal(w.ActivePerks, &perks)
		weaponProf = append(weaponProf, domain.V2AuctionWeaponProf{
			ItemID: w.ItemID, Experience: w.Experience, WeaponLevel: w.WeaponLevel,
			MasteryAchieved: w.MasteryAchieved, ActivePerks: perks,
		})
	}

	battlepass := make([]domain.V2AuctionBattlepass, 0, len(p.BattlepassSeasons))
	for _, b := range p.BattlepassSeasons {
		var steps []domain.V2AuctionBattlepassStep
		_ = json.Unmarshal(b.Steps, &steps)
		battlepass = append(battlepass, domain.V2AuctionBattlepass{
			Season: b.Season, Points: b.Points, Active: int(b.Active), ShopPoints: b.ShopPoints, Steps: steps,
		})
	}

	achievements := make([]domain.V2AuctionAchievement, 0, len(p.Achievements))
	for _, a := range p.Achievements {
		achievements = append(achievements, domain.V2AuctionAchievement{ID: a.ID, UnlockedAt: a.UnlockedAt})
	}

	bountyTalismans := make([]domain.V2AuctionBountyTalisman, 0, len(p.BountyTalismans))
	for _, b := range p.BountyTalismans {
		bountyTalismans = append(bountyTalismans, domain.V2AuctionBountyTalisman{
			Type: int(b.Type), Level: int(b.Level), EffectValue: int(b.EffectValue),
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
			ManaSpent: int64(p.General.ManaSpent), Cap: p.General.Cap,
			Stamina: p.General.Stamina, Soul: p.General.Soul,
			Experience: int64(p.General.Experience), MagLevel: p.General.MagLevel,
			Skills: domain.AuctionSkills{
				Axe: p.General.Skills.Axe, Club: p.General.Skills.Club,
				Sword: p.General.Skills.Sword, Distance: distSkill,
				Shielding: p.General.Skills.Shielding, Fishing: p.General.Skills.Fishing,
				Fist: p.General.Skills.Fist,
			},
			MountsCount: p.General.MountsCount, OutfitsCount: p.General.OutfitsCount,
			TitlesCount: p.General.TitlesCount, LinkedTasks: p.General.LinkedTasks,
			CreateDate: unixSecondsToRFC3339(p.General.CreateDate),
			Balance: int64(p.General.Balance), TotalMoney: int64(p.General.TotalMoney),
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

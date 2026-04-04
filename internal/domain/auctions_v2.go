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

type V2AuctionWeaponPerk struct {
	Lane  int `json:"lane"`
	Index int `json:"index"`
}

type V2AuctionWeaponProf struct {
	ItemID          int                  `json:"item_id"`
	Experience      int                  `json:"experience"`
	WeaponLevel     int                  `json:"weapon_level"`
	MasteryAchieved bool                 `json:"mastery_achieved"`
	ActivePerks     []V2AuctionWeaponPerk `json:"active_perks"`
}

type V2AuctionBattlepassStep struct {
	ID       int  `json:"id,omitempty"`
	Unlocked bool `json:"unlocked,omitempty"`
}

type V2AuctionBattlepass struct {
	Season     int                       `json:"season"`
	Points     int                       `json:"points"`
	Active     int                       `json:"active"`
	ShopPoints int                       `json:"shoppoints"`
	Steps      []V2AuctionBattlepassStep `json:"steps"`
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
	Type         string            `json:"type"`
	Page         int               `json:"page"`
	TotalResults int               `json:"total_results"`
	TotalPages   int               `json:"total_pages"`
	Entries      []V2AuctionDetail `json:"entries"`
}

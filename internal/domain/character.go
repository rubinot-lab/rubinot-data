package domain

type CharacterGuild struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Rank string `json:"rank,omitempty"`
	Nick string `json:"nick,omitempty"`
}

type CharacterHouse struct {
	HouseID int    `json:"house_id,omitempty"`
	Name    string `json:"name"`
	World   string `json:"world,omitempty"`
	TownID  int    `json:"town_id,omitempty"`
	Rent    int    `json:"rent,omitempty"`
	Size    int    `json:"size,omitempty"`
}

type CharacterOutfit struct {
	LookType   int `json:"looktype,omitempty"`
	LookHead   int `json:"lookhead,omitempty"`
	LookBody   int `json:"lookbody,omitempty"`
	LookLegs   int `json:"looklegs,omitempty"`
	LookFeet   int `json:"lookfeet,omitempty"`
	LookAddons int `json:"lookaddons,omitempty"`
}

type CharacterInfo struct {
	ID                *int             `json:"id,omitempty"`
	AccountID         *int             `json:"account_id,omitempty"`
	Name              string           `json:"name"`
	FormerNames       []string         `json:"former_names,omitempty"`
	Traded            bool             `json:"traded,omitempty"`
	AuctionURL        string           `json:"auction_url,omitempty"`
	DeletionDate      string           `json:"deletion_date,omitempty"`
	Sex               string           `json:"sex,omitempty"`
	Title             string           `json:"title,omitempty"`
	UnlockedTitles    int              `json:"unlocked_titles,omitempty"`
	Vocation          string           `json:"vocation,omitempty"`
	VocationID        int              `json:"vocation_id,omitempty"`
	Level             int              `json:"level,omitempty"`
	AchievementPoints int              `json:"achievement_points,omitempty"`
	World             string           `json:"world,omitempty"`
	WorldID           int              `json:"world_id,omitempty"`
	FormerWorlds      []string         `json:"former_worlds,omitempty"`
	Residence         string           `json:"residence,omitempty"`
	MarriedTo         string           `json:"married_to,omitempty"`
	House             *CharacterHouse  `json:"house,omitempty"`
	Houses            []CharacterHouse `json:"houses,omitempty"`
	Guild             *CharacterGuild  `json:"guild,omitempty"`
	LastLogin         string           `json:"last_login,omitempty"`
	AccountStatus     string           `json:"account_status,omitempty"`
	Comment           string           `json:"comment,omitempty"`
	IsBanned          bool             `json:"is_banned,omitempty"`
	BanReason         string           `json:"ban_reason,omitempty"`
	LoyaltyPoints     int              `json:"loyalty_points,omitempty"`
	IsHidden          bool             `json:"is_hidden,omitempty"`
	Created           string           `json:"created,omitempty"`
	AccountCreated    string           `json:"account_created,omitempty"`
	VIPTime           int64            `json:"vip_time,omitempty"`
	Outfit            *CharacterOutfit `json:"outfit,omitempty"`
	Auction           any              `json:"auction,omitempty"`
	FoundByOldName    bool             `json:"found_by_old_name,omitempty"`
}

type CharacterDeath struct {
	Time               string   `json:"time"`
	Level              int      `json:"level"`
	KilledBy           string   `json:"killed_by,omitempty"`
	IsPlayerKill       bool     `json:"is_player_kill,omitempty"`
	MostDamageBy       string   `json:"mostdamage_by,omitempty"`
	MostDamageIsPlayer bool     `json:"mostdamage_is_player,omitempty"`
	Killers            []string `json:"killers,omitempty"`
	Assists            []string `json:"assists,omitempty"`
	Reason             string   `json:"reason,omitempty"`
}

type AccountInformation struct {
	Created       string `json:"created,omitempty"`
	LoyaltyTitle  string `json:"loyalty_title,omitempty"`
	LoyaltyPoints int    `json:"loyalty_points,omitempty"`
}

type OtherCharacter struct {
	Name     string `json:"name"`
	World    string `json:"world"`
	WorldID  int    `json:"world_id,omitempty"`
	Level    int    `json:"level,omitempty"`
	Vocation string `json:"vocation,omitempty"`
	IsOnline bool   `json:"is_online,omitempty"`
}

type CharacterBadge struct {
	ID       int    `json:"id,omitempty"`
	ClientID int    `json:"client_id,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
}

type DisplayedAchievement struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CharacterResult struct {
	CharacterInfo                CharacterInfo          `json:"character_info"`
	Deaths                       []CharacterDeath       `json:"deaths,omitempty"`
	AccountInfo                  *AccountInformation    `json:"account_information,omitempty"`
	OtherCharacters              []OtherCharacter       `json:"other_characters,omitempty"`
	AccountBadges                []CharacterBadge       `json:"account_badges,omitempty"`
	DisplayedAchievements        []DisplayedAchievement `json:"displayed_achievements,omitempty"`
	CanSeeCharacterIdentifiers   bool                   `json:"can_see_character_identifiers,omitempty"`
}

type ComparisonSignals struct {
	SameAccount        bool `json:"same_account"`
	SameVipTime        bool `json:"same_vip_time"`
	SameAccountCreated bool `json:"same_account_created"`
	SameHouse          bool `json:"same_house"`
	SameGuild          bool `json:"same_guild"`
	SameOutfit         bool `json:"same_outfit"`
	SameCreated        bool `json:"same_created"`
	SameLoyaltyPoints  bool `json:"same_loyalty_points"`
}

type ComparisonResult struct {
	Characters []CharacterResult `json:"characters"`
	Signals    ComparisonSignals `json:"signals"`
}

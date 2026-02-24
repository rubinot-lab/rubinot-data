package domain

type CharacterGuild struct {
	Name string `json:"name,omitempty"`
	Rank string `json:"rank,omitempty"`
}

type CharacterHouse struct {
	HouseID int    `json:"house_id,omitempty"`
	Name    string `json:"name"`
	World   string `json:"world,omitempty"`
}

type CharacterInfo struct {
	Name              string           `json:"name"`
	FormerNames       []string         `json:"former_names,omitempty"`
	Traded            bool             `json:"traded,omitempty"`
	AuctionURL        string           `json:"auction_url,omitempty"`
	DeletionDate      string           `json:"deletion_date,omitempty"`
	Sex               string           `json:"sex,omitempty"`
	Title             string           `json:"title,omitempty"`
	UnlockedTitles    int              `json:"unlocked_titles,omitempty"`
	Vocation          string           `json:"vocation,omitempty"`
	Level             int              `json:"level,omitempty"`
	AchievementPoints int              `json:"achievement_points,omitempty"`
	World             string           `json:"world,omitempty"`
	FormerWorlds      []string         `json:"former_worlds,omitempty"`
	Residence         string           `json:"residence,omitempty"`
	MarriedTo         string           `json:"married_to,omitempty"`
	Houses            []CharacterHouse `json:"houses,omitempty"`
	Guild             *CharacterGuild  `json:"guild,omitempty"`
	LastLogin         string           `json:"last_login,omitempty"`
	AccountStatus     string           `json:"account_status,omitempty"`
	Comment           string           `json:"comment,omitempty"`
	IsBanned          bool             `json:"is_banned,omitempty"`
	BanReason         string           `json:"ban_reason,omitempty"`
}

type CharacterDeath struct {
	Time    string   `json:"time"`
	Level   int      `json:"level"`
	Killers []string `json:"killers,omitempty"`
	Assists []string `json:"assists,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

type AccountInformation struct {
	Created      string `json:"created,omitempty"`
	LoyaltyTitle string `json:"loyalty_title,omitempty"`
}

type OtherCharacter struct {
	Name    string `json:"name"`
	World   string `json:"world"`
	Status  string `json:"status,omitempty"`
	Main    bool   `json:"main,omitempty"`
	Traded  bool   `json:"traded,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
}

type CharacterResult struct {
	CharacterInfo   CharacterInfo      `json:"character_info"`
	Deaths          []CharacterDeath   `json:"deaths,omitempty"`
	AccountInfo     *AccountInformation `json:"account_information,omitempty"`
	OtherCharacters []OtherCharacter   `json:"other_characters,omitempty"`
}

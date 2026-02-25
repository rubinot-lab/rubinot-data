package domain

type GuildHall struct {
	Name    string `json:"name"`
	HouseID int    `json:"house_id,omitempty"`
}

type GuildOwner struct {
	ID       int    `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Level    int    `json:"level,omitempty"`
	Vocation int    `json:"vocation,omitempty"`
}

type GuildRank struct {
	ID    int    `json:"id,omitempty"`
	Name  string `json:"name"`
	Level int    `json:"level,omitempty"`
}

type GuildResidence struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Town string `json:"town,omitempty"`
}

type GuildMember struct {
	ID         int    `json:"id,omitempty"`
	Name       string `json:"name"`
	Title      string `json:"title,omitempty"`
	Rank       string `json:"rank,omitempty"`
	RankLevel  int    `json:"rank_level,omitempty"`
	Vocation   string `json:"vocation,omitempty"`
	VocationID int    `json:"vocation_id,omitempty"`
	Level      int    `json:"level,omitempty"`
	Joined     string `json:"joined,omitempty"`
	Status     string `json:"status,omitempty"`
	IsOnline   bool   `json:"is_online"`
}

type GuildInvite struct {
	Name string `json:"name"`
	Date string `json:"date,omitempty"`
}

type GuildResult struct {
	ID               int             `json:"id,omitempty"`
	Name             string          `json:"name"`
	MOTD             string          `json:"motd,omitempty"`
	World            string          `json:"world,omitempty"`
	WorldID          int             `json:"world_id,omitempty"`
	Description      string          `json:"description,omitempty"`
	LogoName         string          `json:"logo_name,omitempty"`
	Balance          string          `json:"balance,omitempty"`
	Guildhall        *GuildHall      `json:"guildhall,omitempty"`
	Residence        *GuildResidence `json:"residence,omitempty"`
	Active           bool            `json:"active"`
	Founded          string          `json:"founded,omitempty"`
	OpenApplications bool            `json:"open_applications"`
	Homepage         string          `json:"homepage,omitempty"`
	InWar            bool            `json:"in_war"`
	DisbandDate      string          `json:"disband_date,omitempty"`
	DisbandCondition string          `json:"disband_condition,omitempty"`
	Owner            *GuildOwner     `json:"owner,omitempty"`
	Ranks            []GuildRank     `json:"ranks,omitempty"`
	PlayersOnline    int             `json:"players_online"`
	PlayersOffline   int             `json:"players_offline"`
	MembersTotal     int             `json:"members_total"`
	MembersInvited   int             `json:"members_invited"`
	Members          []GuildMember   `json:"members,omitempty"`
	Invites          []GuildInvite   `json:"invites,omitempty"`
}

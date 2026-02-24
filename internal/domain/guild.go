package domain

type GuildHall struct {
	Name    string `json:"name"`
	HouseID int    `json:"house_id,omitempty"`
}

type GuildMember struct {
	Name     string `json:"name"`
	Title    string `json:"title,omitempty"`
	Rank     string `json:"rank,omitempty"`
	Vocation string `json:"vocation,omitempty"`
	Level    int    `json:"level,omitempty"`
	Joined   string `json:"joined,omitempty"`
	Status   string `json:"status,omitempty"`
	IsOnline bool   `json:"is_online"`
}

type GuildInvite struct {
	Name string `json:"name"`
	Date string `json:"date,omitempty"`
}

type GuildResult struct {
	Name             string        `json:"name"`
	World            string        `json:"world,omitempty"`
	Description      string        `json:"description,omitempty"`
	Guildhall        *GuildHall    `json:"guildhall,omitempty"`
	Active           bool          `json:"active"`
	Founded          string        `json:"founded,omitempty"`
	OpenApplications bool          `json:"open_applications"`
	Homepage         string        `json:"homepage,omitempty"`
	InWar            bool          `json:"in_war"`
	DisbandDate      string        `json:"disband_date,omitempty"`
	DisbandCondition string        `json:"disband_condition,omitempty"`
	PlayersOnline    int           `json:"players_online"`
	PlayersOffline   int           `json:"players_offline"`
	MembersTotal     int           `json:"members_total"`
	MembersInvited   int           `json:"members_invited"`
	Members          []GuildMember `json:"members,omitempty"`
	Invites          []GuildInvite `json:"invites,omitempty"`
}

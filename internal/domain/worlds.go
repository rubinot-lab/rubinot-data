package domain

type WorldOverview struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	PlayersOnline int    `json:"players_online"`
	Location      string `json:"location"`
	PVPType       string `json:"pvp_type"`
}

type WorldsResult struct {
	TotalPlayersOnline int             `json:"total_players_online"`
	Worlds             []WorldOverview `json:"worlds"`
}

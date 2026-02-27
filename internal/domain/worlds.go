package domain

type WorldOverview struct {
	ID            int    `json:"id,omitempty"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	PlayersOnline int    `json:"players_online"`
	Location      string `json:"location,omitempty"`
	PVPType       string `json:"pvp_type"`
	WorldType     string `json:"world_type,omitempty"`
	Locked        bool   `json:"locked"`
}

type WorldsResult struct {
	TotalPlayersOnline int             `json:"total_players_online"`
	OverallRecord      int             `json:"overall_record,omitempty"`
	OverallRecordTime  int64           `json:"overall_record_time,omitempty"`
	Worlds             []WorldOverview `json:"worlds"`
}

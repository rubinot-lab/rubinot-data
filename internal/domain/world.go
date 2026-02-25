package domain

type PlayerOnline struct {
	Name       string `json:"name"`
	Level      int    `json:"level"`
	Vocation   string `json:"vocation"`
	VocationID int    `json:"vocation_id,omitempty"`
}

type WorldInfo struct {
	ID            int    `json:"id,omitempty"`
	Status        string `json:"status,omitempty"`
	PlayersOnline int    `json:"players_online"`
	Location      string `json:"location,omitempty"`
	PVPType       string `json:"pvp_type,omitempty"`
	WorldType     string `json:"world_type,omitempty"`
	Locked        bool   `json:"locked"`
	CreationDate  string `json:"creation_date,omitempty"`
	Record        int    `json:"record,omitempty"`
	RecordTime    int64  `json:"record_time,omitempty"`
}

type WorldResult struct {
	Name          string         `json:"name"`
	Info          WorldInfo      `json:"info"`
	PlayersOnline []PlayerOnline `json:"players_online"`
}

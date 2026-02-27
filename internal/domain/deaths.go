package domain

type DeathFilters struct {
	Guild    string `json:"guild,omitempty"`
	MinLevel int    `json:"min_level,omitempty"`
	PvPOnly  *bool  `json:"pvp_only,omitempty"`
	Page     int    `json:"page,omitempty"`
}

type DeathVictim struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type DeathEntry struct {
	PlayerID           int         `json:"player_id,omitempty"`
	Date               string      `json:"date"`
	Victim             DeathVictim `json:"victim"`
	KilledBy           string      `json:"killed_by,omitempty"`
	IsPlayerKill       bool        `json:"is_player_kill,omitempty"`
	MostDamageBy       string      `json:"mostdamage_by,omitempty"`
	MostDamageIsPlayer bool        `json:"mostdamage_is_player,omitempty"`
	WorldID            int         `json:"world_id,omitempty"`
	Killers            []string    `json:"killers,omitempty"`
	IsPvP              bool        `json:"is_pvp"`
}

type DeathPagination struct {
	CurrentPage  int `json:"current_page"`
	TotalPages   int `json:"total_pages"`
	TotalCount   int `json:"total_count"`
	ItemsPerPage int `json:"items_per_page"`
}

type DeathsResult struct {
	World       string          `json:"world"`
	Filters     DeathFilters    `json:"filters,omitempty"`
	Entries     []DeathEntry    `json:"entries"`
	TotalDeaths int             `json:"total_deaths,omitempty"`
	Pagination  DeathPagination `json:"pagination,omitempty"`
}

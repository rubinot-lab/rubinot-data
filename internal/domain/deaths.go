package domain

type DeathFilters struct {
	Guild    string `json:"guild,omitempty"`
	MinLevel int    `json:"min_level,omitempty"`
	PvPOnly  *bool  `json:"pvp_only,omitempty"`
}

type DeathVictim struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type DeathEntry struct {
	Date    string      `json:"date"`
	Victim  DeathVictim `json:"victim"`
	Killers []string    `json:"killers"`
	IsPvP   bool        `json:"is_pvp"`
}

type DeathsResult struct {
	World       string       `json:"world"`
	Filters     DeathFilters `json:"filters,omitempty"`
	Entries     []DeathEntry `json:"entries"`
	TotalDeaths int          `json:"total_deaths"`
}

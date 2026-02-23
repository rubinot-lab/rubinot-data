package domain

type KillstatisticsEntry struct {
	Race                  string `json:"race"`
	LastDayPlayersKilled  int    `json:"last_day_players_killed"`
	LastDayKilled         int    `json:"last_day_killed"`
	LastWeekPlayersKilled int    `json:"last_week_players_killed"`
	LastWeekKilled        int    `json:"last_week_killed"`
}

type KillstatisticsTotal struct {
	LastDayPlayersKilled  int `json:"last_day_players_killed"`
	LastDayKilled         int `json:"last_day_killed"`
	LastWeekPlayersKilled int `json:"last_week_players_killed"`
	LastWeekKilled        int `json:"last_week_killed"`
}

type KillstatisticsResult struct {
	World   string                `json:"world"`
	Entries []KillstatisticsEntry `json:"entries"`
	Total   KillstatisticsTotal   `json:"total"`
}

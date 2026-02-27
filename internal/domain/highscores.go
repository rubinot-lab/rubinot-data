package domain

type Highscore struct {
	Rank       int    `json:"rank"`
	ID         int    `json:"id,omitempty"`
	Name       string `json:"name"`
	Vocation   string `json:"vocation,omitempty"`
	VocationID int    `json:"vocation_id,omitempty"`
	World      string `json:"world,omitempty"`
	WorldID    int    `json:"world_id,omitempty"`
	Level      int    `json:"level,omitempty"`
	Value      string `json:"value"`
	Title      string `json:"title,omitempty"`
	Traded     bool   `json:"traded,omitempty"`
	AuctionURL string `json:"auction_url,omitempty"`
}

type HighscorePage struct {
	CurrentPage  int `json:"current_page"`
	TotalPages   int `json:"total_pages"`
	TotalRecords int `json:"total_records"`
}

type HighscoresResult struct {
	World            string        `json:"world"`
	Category         string        `json:"category"`
	Vocation         string        `json:"vocation"`
	HighscoreAge     int           `json:"highscore_age,omitempty"`
	CachedAt         int64         `json:"cached_at,omitempty"`
	HighscoreList    []Highscore   `json:"highscore_list"`
	HighscorePage    HighscorePage `json:"highscore_page"`
	AvailableSeasons []int         `json:"available_seasons,omitempty"`
}

type HighscoresByWorldResult struct {
	World        string             `json:"world"`
	Category     string             `json:"category"`
	Vocation     string             `json:"vocation"`
	TotalWorlds  int                `json:"total_worlds"`
	TotalRecords int                `json:"total_records"`
	TotalEntries int                `json:"total_entries"`
	Worlds       []HighscoresResult `json:"worlds"`
}

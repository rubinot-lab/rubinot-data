package domain

type Highscore struct {
	Rank       int    `json:"rank"`
	Name       string `json:"name"`
	Vocation   string `json:"vocation,omitempty"`
	World      string `json:"world,omitempty"`
	Level      int    `json:"level,omitempty"`
	Value      int    `json:"value"`
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
	World         string        `json:"world"`
	Category      string        `json:"category"`
	Vocation      string        `json:"vocation"`
	HighscoreAge  int           `json:"highscore_age"`
	HighscoreList []Highscore   `json:"highscore_list"`
	HighscorePage HighscorePage `json:"highscore_page"`
}

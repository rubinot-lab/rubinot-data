package domain

type EventDay struct {
	Day          int      `json:"day"`
	Events       []string `json:"events"`
	ActiveEvents []string `json:"active_events"`
	EndingEvents []string `json:"ending_events"`
}

type EventsResult struct {
	Month      string     `json:"month"`
	Year       int        `json:"year"`
	LastUpdate string     `json:"last_update,omitempty"`
	Days       []EventDay `json:"days"`
	AllEvents  []string   `json:"all_events"`
}

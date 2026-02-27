package domain

type BoostedEntity struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	LookType int    `json:"looktype"`
}

type BoostedResult struct {
	Boss    BoostedEntity `json:"boss"`
	Monster BoostedEntity `json:"monster"`
}

type EventsCalendarEvent struct {
	ID                 int      `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	ColorDark          string   `json:"color_dark,omitempty"`
	ColorLight         string   `json:"color_light,omitempty"`
	DisplayPriority    int      `json:"display_priority,omitempty"`
	SpecialEffect      *string  `json:"special_effect,omitempty"`
	StartDate          *string  `json:"start_date,omitempty"`
	EndDate            *string  `json:"end_date,omitempty"`
	IsRecurring        bool     `json:"is_recurring"`
	RecurringWeekdays  []int    `json:"recurring_weekdays,omitempty"`
	RecurringMonthDays []int    `json:"recurring_month_days,omitempty"`
	RecurringStart     *string  `json:"recurring_start,omitempty"`
	RecurringEnd       *string  `json:"recurring_end,omitempty"`
	Tags               []string `json:"tags,omitempty"`
}

type EventsCalendarResult struct {
	Month       int                              `json:"month"`
	Year        int                              `json:"year"`
	Events      []EventsCalendarEvent            `json:"events"`
	EventsByDay map[string][]EventsCalendarEvent `json:"events_by_day"`
}

type MaintenanceResult struct {
	IsClosed     bool   `json:"is_closed"`
	CloseMessage string `json:"close_message"`
}

type GeoLanguageResult struct {
	Language    string `json:"language"`
	CountryCode string `json:"country_code"`
}

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

type CalendarEvent struct {
	ID                 int      `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	ColorDark          string   `json:"color_dark"`
	ColorLight         string   `json:"color_light"`
	DisplayPriority    int      `json:"display_priority"`
	SpecialEffect      string   `json:"special_effect"`
	StartDate          string   `json:"start_date"`
	EndDate            string   `json:"end_date"`
	IsRecurring        bool     `json:"is_recurring"`
	RecurringWeekdays  []int    `json:"recurring_weekdays,omitempty"`
	RecurringMonthDays []int    `json:"recurring_month_days,omitempty"`
	RecurringStart     string   `json:"recurring_start,omitempty"`
	RecurringEnd       string   `json:"recurring_end,omitempty"`
	Tags               []string `json:"tags,omitempty"`
}

type EventCalendarResult struct {
	Events []CalendarEvent `json:"events"`
}

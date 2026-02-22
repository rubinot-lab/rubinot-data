package domain

type BanishmentEntry struct {
	Date        string `json:"date"`
	Character   string `json:"character"`
	Reason      string `json:"reason"`
	Duration    string `json:"duration"`
	IsPermanent bool   `json:"is_permanent"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type BanishmentsResult struct {
	World     string            `json:"world"`
	Page      int               `json:"page"`
	TotalBans int               `json:"total_bans"`
	Entries   []BanishmentEntry `json:"entries"`
}

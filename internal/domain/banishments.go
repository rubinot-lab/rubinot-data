package domain

type BanishmentEntry struct {
	AccountID   int    `json:"account_id,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	Date        string `json:"date"`
	Character   string `json:"character"`
	Reason      string `json:"reason"`
	Duration    string `json:"duration"`
	IsPermanent bool   `json:"is_permanent"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	BannedBy    string `json:"banned_by,omitempty"`
}

type BanishmentsResult struct {
	World      string            `json:"world"`
	Page       int               `json:"page"`
	TotalBans  int               `json:"total_bans"`
	TotalPages int               `json:"total_pages,omitempty"`
	Entries    []BanishmentEntry `json:"entries"`
}

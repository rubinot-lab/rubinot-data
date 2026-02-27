package domain

type AuctionSkills struct {
	Axe       int `json:"axe,omitempty"`
	Club      int `json:"club,omitempty"`
	Distance  int `json:"distance,omitempty"`
	Fishing   int `json:"fishing,omitempty"`
	Fist      int `json:"fist,omitempty"`
	Magic     int `json:"magic,omitempty"`
	Shielding int `json:"shielding,omitempty"`
	Sword     int `json:"sword,omitempty"`
}

type AuctionOutfit struct {
	LookType   int `json:"looktype,omitempty"`
	LookHead   int `json:"lookhead,omitempty"`
	LookBody   int `json:"lookbody,omitempty"`
	LookLegs   int `json:"looklegs,omitempty"`
	LookFeet   int `json:"lookfeet,omitempty"`
	LookAddons int `json:"lookaddons,omitempty"`
	LookMount  int `json:"lookmount,omitempty"`
}

type AuctionItem struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type AuctionAugment struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type AuctionEntry struct {
	AuctionID         int              `json:"auction_id"`
	State             int              `json:"state,omitempty"`
	StateName         string           `json:"state_name,omitempty"`
	PlayerID          int              `json:"player_id,omitempty"`
	OwnerAccountID    int              `json:"owner_account_id,omitempty"`
	CharacterName     string           `json:"character_name"`
	Level             int              `json:"level"`
	Vocation          string           `json:"vocation"`
	VocationID        int              `json:"vocation_id,omitempty"`
	Sex               string           `json:"sex"`
	World             string           `json:"world"`
	WorldID           int              `json:"world_id,omitempty"`
	BidType           string           `json:"bid_type,omitempty"`
	BidValue          int              `json:"bid_value,omitempty"`
	StartingValue     int              `json:"starting_value,omitempty"`
	CurrentValue      int              `json:"current_value,omitempty"`
	AuctionStart      string           `json:"auction_start,omitempty"`
	AuctionEnd        string           `json:"auction_end,omitempty"`
	Status            string           `json:"status"`
	CharmPoints       int              `json:"charm_points,omitempty"`
	AchievementPoints int              `json:"achievement_points,omitempty"`
	MagLevel          int              `json:"mag_level,omitempty"`
	Skills            AuctionSkills    `json:"skills,omitempty"`
	Outfit            AuctionOutfit    `json:"outfit,omitempty"`
	HighlightItems    []AuctionItem    `json:"highlight_items,omitempty"`
	HighlightAugments []AuctionAugment `json:"highlight_augments,omitempty"`
}

type AuctionsPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type AuctionsResult struct {
	Type         string              `json:"type"`
	Page         int                 `json:"page"`
	TotalResults int                 `json:"total_results"`
	TotalPages   int                 `json:"total_pages"`
	Entries      []AuctionEntry      `json:"entries"`
	Pagination   *AuctionsPagination `json:"pagination,omitempty"`
}

type AuctionDetail struct {
	AuctionID         int              `json:"auction_id"`
	State             int              `json:"state,omitempty"`
	StateName         string           `json:"state_name,omitempty"`
	PlayerID          int              `json:"player_id,omitempty"`
	OwnerAccountID    int              `json:"owner_account_id,omitempty"`
	CharacterName     string           `json:"character_name"`
	Level             int              `json:"level"`
	Vocation          string           `json:"vocation"`
	VocationID        int              `json:"vocation_id,omitempty"`
	Sex               string           `json:"sex"`
	World             string           `json:"world"`
	WorldID           int              `json:"world_id,omitempty"`
	BidType           string           `json:"bid_type,omitempty"`
	BidValue          int              `json:"bid_value,omitempty"`
	StartingValue     int              `json:"starting_value,omitempty"`
	CurrentValue      int              `json:"current_value,omitempty"`
	WinningBid        int              `json:"winning_bid,omitempty"`
	HighestBidderID   int              `json:"highest_bidder_id,omitempty"`
	AuctionStart      string           `json:"auction_start,omitempty"`
	AuctionEnd        string           `json:"auction_end,omitempty"`
	Status            string           `json:"status"`
	CharmPoints       int              `json:"charm_points,omitempty"`
	AchievementPoints int              `json:"achievement_points,omitempty"`
	MagLevel          int              `json:"mag_level,omitempty"`
	Skills            AuctionSkills    `json:"skills,omitempty"`
	Outfit            AuctionOutfit    `json:"outfit,omitempty"`
	HighlightItems    []AuctionItem    `json:"highlight_items,omitempty"`
	HighlightAugments []AuctionAugment `json:"highlight_augments,omitempty"`
}

package domain

type AuctionEntry struct {
	AuctionID     string `json:"auction_id"`
	CharacterName string `json:"character_name"`
	Level         int    `json:"level"`
	Vocation      string `json:"vocation"`
	Sex           string `json:"sex"`
	World         string `json:"world"`
	BidType       string `json:"bid_type,omitempty"`
	BidValue      int    `json:"bid_value,omitempty"`
	AuctionEnd    string `json:"auction_end,omitempty"`
	Status        string `json:"status"`
}

type AuctionsResult struct {
	Type         string         `json:"type"`
	Page         int            `json:"page"`
	TotalResults int            `json:"total_results"`
	TotalPages   int            `json:"total_pages"`
	Entries      []AuctionEntry `json:"entries"`
}

type AuctionDetail struct {
	AuctionID     string `json:"auction_id"`
	CharacterName string `json:"character_name"`
	Level         int    `json:"level"`
	Vocation      string `json:"vocation"`
	Sex           string `json:"sex"`
	World         string `json:"world"`
	AuctionStart  string `json:"auction_start,omitempty"`
	AuctionEnd    string `json:"auction_end,omitempty"`
	BidType       string `json:"bid_type,omitempty"`
	BidValue      int    `json:"bid_value,omitempty"`
	Status        string `json:"status"`
}

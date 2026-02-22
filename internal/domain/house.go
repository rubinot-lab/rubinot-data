package domain

type HouseOwner struct {
	Name       string `json:"name"`
	Level      int    `json:"level,omitempty"`
	Vocation   string `json:"vocation,omitempty"`
	PaidUntil  string `json:"paid_until,omitempty"`
	MovingDate string `json:"moving_date,omitempty"`
}

type HouseAuction struct {
	CurrentBid int    `json:"current_bid"`
	Bidder     string `json:"bidder,omitempty"`
	EndDate    string `json:"end_date,omitempty"`
	NoBidYet   bool   `json:"no_bid_yet,omitempty"`
}

type HouseResult struct {
	HouseID int           `json:"house_id"`
	Name    string        `json:"name"`
	World   string        `json:"world"`
	Town    string        `json:"town"`
	Size    int           `json:"size"`
	Beds    int           `json:"beds,omitempty"`
	Rent    int           `json:"rent"`
	Status  string        `json:"status"`
	Owner   *HouseOwner   `json:"owner,omitempty"`
	Auction *HouseAuction `json:"auction,omitempty"`
}

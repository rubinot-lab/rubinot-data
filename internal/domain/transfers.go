package domain

type TransferFilters struct {
	World    string `json:"world,omitempty"`
	MinLevel int    `json:"min_level,omitempty"`
}

type TransferEntry struct {
	PlayerName       string `json:"player_name"`
	Level            int    `json:"level"`
	FormerWorld      string `json:"former_world"`
	DestinationWorld string `json:"destination_world"`
	TransferDate     string `json:"transfer_date"`
}

type TransfersResult struct {
	Filters        TransferFilters `json:"filters,omitempty"`
	Page           int             `json:"page"`
	TotalTransfers int             `json:"total_transfers"`
	Entries        []TransferEntry `json:"entries"`
}

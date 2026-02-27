package domain

type TransferFilters struct {
	World    string `json:"world,omitempty"`
	MinLevel int    `json:"min_level,omitempty"`
}

type TransferEntry struct {
	ID               int    `json:"id,omitempty"`
	PlayerID         int    `json:"player_id,omitempty"`
	PlayerName       string `json:"player_name"`
	Level            int    `json:"level"`
	FormerWorld      string `json:"former_world"`
	FormerWorldID    int    `json:"former_world_id,omitempty"`
	DestinationWorld string `json:"destination_world"`
	DestWorldID      int    `json:"destination_world_id,omitempty"`
	TransferDate     string `json:"transfer_date"`
}

type TransfersResult struct {
	Filters        TransferFilters `json:"filters,omitempty"`
	Page           int             `json:"page"`
	TotalTransfers int             `json:"total_transfers"`
	TotalPages     int             `json:"total_pages,omitempty"`
	Entries        []TransferEntry `json:"entries"`
}

package domain

type HouseEntry struct {
	HouseID     int    `json:"house_id"`
	Name        string `json:"name"`
	Size        int    `json:"size"`
	Rent        int    `json:"rent"`
	Status      string `json:"status"`
	IsRented    bool   `json:"rented"`
	IsAuctioned bool   `json:"auctioned"`
}

type HousesResult struct {
	World         string       `json:"world"`
	Town          string       `json:"town"`
	HouseList     []HouseEntry `json:"house_list"`
	GuildhallList []HouseEntry `json:"guildhall_list"`
}

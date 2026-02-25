package domain

type GuildListEntry struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	WorldID     int    `json:"world_id,omitempty"`
	LogoName    string `json:"logo_name,omitempty"`
	LogoURL     string `json:"logo_url,omitempty"`
}

type GuildsResult struct {
	World     string           `json:"world"`
	Guilds    []GuildListEntry `json:"guilds,omitempty"`
	Active    []GuildListEntry `json:"active,omitempty"`
	Formation []GuildListEntry `json:"formation,omitempty"`
}

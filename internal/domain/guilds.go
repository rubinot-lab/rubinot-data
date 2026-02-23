package domain

type GuildListEntry struct {
	Name        string `json:"name"`
	LogoURL     string `json:"logo_url,omitempty"`
	Description string `json:"description,omitempty"`
}

type GuildsResult struct {
	World     string           `json:"world"`
	Active    []GuildListEntry `json:"active"`
	Formation []GuildListEntry `json:"formation"`
}

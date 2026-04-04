package scraper

type SchemaExpectation struct {
	TopLevel []string
	Nested   map[string][]string
}

var UpstreamSchemas = map[string]SchemaExpectation{
	"/api/worlds": {
		TopLevel: []string{"worlds", "totalOnline", "overallRecord", "overallRecordTime"},
		Nested: map[string][]string{
			"worlds[0]": {"name", "pvpType", "pvpTypeLabel", "worldType", "locked", "playersOnline"},
		},
	},
	"/api/highscores": {
		TopLevel: []string{"players", "totalCount", "cachedAt"},
		Nested: map[string][]string{
			"players[0]": {"rank", "name", "level", "vocation", "worldName", "value"},
		},
	},
	"/api/deaths": {
		TopLevel: []string{"deaths", "pagination", "canSeeDeathDetails"},
		Nested: map[string][]string{
			"deaths[0]": {"time", "level", "killed_by", "is_player", "mostdamage_by", "mostdamage_is_player", "victim", "worldName"},
		},
	},
	"/api/guilds": {
		TopLevel: []string{"guilds", "totalCount", "totalPages", "currentPage"},
		Nested: map[string][]string{
			"guilds[0]": {"name", "description", "worldName", "logo_name"},
		},
	},
	"/api/transfers": {
		TopLevel: []string{"transfers", "totalResults", "totalPages", "currentPage"},
		Nested: map[string][]string{
			"transfers[0]": {"player_name", "player_level", "from_world", "to_world", "transferred_at"},
		},
	},
	"/api/boosted": {
		TopLevel: []string{"boss", "monster"},
		Nested: map[string][]string{
			"boss":    {"id", "name", "looktype", "addons", "head", "body", "legs", "feet"},
			"monster": {"id", "name", "looktype"},
		},
	},
	"/api/maintenance": {
		TopLevel: []string{"isClosed", "closeMessage"},
	},
	"/api/news": {
		TopLevel: []string{"tickers", "articles"},
		Nested: map[string][]string{
			"tickers[0]":  {"id", "message", "category_id", "category", "author", "created_at"},
			"articles[0]": {"id", "title", "slug", "summary", "content", "cover_image", "author", "category", "published_at"},
		},
	},
	"/api/events/calendar": {
		TopLevel: []string{"events"},
		Nested: map[string][]string{
			"events[0]": {"id", "name", "description", "colorDark", "colorLight", "displayPriority", "specialEffect", "startDate", "endDate", "isRecurring", "recurringWeekdays", "recurringMonthDays", "recurringStart", "recurringEnd", "tags"},
		},
	},
	"/api/bazaar": {
		TopLevel: []string{"auctions", "pagination"},
		Nested: map[string][]string{
			"auctions[0]": {"id", "state", "stateName", "owner", "startingValue", "currentValue", "auctionStart", "auctionEnd", "name", "level", "vocation", "vocationName", "sex", "worldName", "lookType", "lookHead", "lookBody", "lookLegs", "lookFeet", "lookAddons", "direction", "charmPoints", "achievementPoints", "magLevel", "skills", "highlightItems", "highlightAugments"},
		},
	},
	"/api/bazaar/{id}": {
		TopLevel: []string{"auction", "player", "general", "items", "itemsTotal", "storeItems", "storeItemsTotal", "outfits", "mounts", "familiars", "charms", "blessings", "titles", "gems", "bosstiaries", "bosstiariosTotal", "weaponProficiency", "battlepassSeasons", "achievements", "bountyTalismans", "bountyPoints", "totalBountyPoints", "bountyRerolls", "auras", "hirelingSkills", "hirelingWardrobe", "highlightItems", "highlightAugments", "isAdmin", "adminInfo", "storages"},
		Nested: map[string][]string{
			"auction": {"id", "state", "stateName", "owner", "startingValue", "currentValue", "winningBid", "highestBidderId", "hasWinner", "auctionStart", "auctionEnd"},
			"general": {"health", "healthMax", "mana", "manaMax", "manaSpent", "cap", "stamina", "soul", "experience", "magLevel", "skills", "mountsCount", "outfitsCount", "titlesCount", "linkedTasks", "createDate", "balance", "totalMoney", "achievementPoints", "charmPoints", "spentCharmPoints", "availableCharmPoints", "spentMinorEchoes", "availableMinorEchoes", "charmExpansion", "streakDays", "huntingTaskPoints", "thirdPrey", "thirdHunting", "permanentWeeklyTaskSlot", "preyWildcards", "hirelingCount", "hirelingJobs", "hirelingOutfits", "dust", "dustMax", "bossPoints", "wheelPoints", "maxWheelPoints", "gpActive", "gpPoints"},
		},
	},
}

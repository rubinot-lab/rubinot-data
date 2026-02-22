package validation

type HighscoreCategory struct {
	ID   int
	Name string
	Slug string
}

func (v *Validator) ResolveHighscoreCategory(category string) (HighscoreCategory, bool) {
	normalized := normalizeLookupValue(category)
	if normalized == "" {
		return HighscoreCategory{}, false
	}

	if aliasTarget, ok := highscoreCategoryAliases[normalized]; ok {
		normalized = aliasTarget
	}

	categoryRef, ok := v.highscoreCategoriesByKey[normalized]
	if !ok {
		return HighscoreCategory{}, false
	}
	return categoryRef, true
}

func (v *Validator) ResolveVocation(vocation string) (string, bool) {
	canonical, ok := v.vocationsByKey[normalizeLookupValue(vocation)]
	if !ok {
		return "", false
	}
	return canonical, true
}

func (v *Validator) loadHighscores() {
	for _, category := range highscoreCategories {
		v.highscoreCategoriesByKey[normalizeLookupValue(category.Slug)] = category
		v.highscoreCategoriesByKey[normalizeLookupValue(category.Name)] = category
	}

	for alias, canonical := range highscoreCategoryAliases {
		category, ok := v.highscoreCategoriesByKey[normalizeLookupValue(canonical)]
		if !ok {
			continue
		}
		v.highscoreCategoriesByKey[normalizeLookupValue(alias)] = category
	}

	for alias, canonical := range vocationAliases {
		v.vocationsByKey[normalizeLookupValue(alias)] = canonical
	}
}

var highscoreCategories = []HighscoreCategory{
	{ID: 0, Name: "Achievements", Slug: "achievements"},
	{ID: 2, Name: "Axe Fighting", Slug: "axe"},
	{ID: 4, Name: "Club Fighting", Slug: "club"},
	{ID: 5, Name: "Distance Fighting", Slug: "distance"},
	{ID: 6, Name: "Experience Points", Slug: "experience"},
	{ID: 7, Name: "Fishing", Slug: "fishing"},
	{ID: 8, Name: "Fist Fighting", Slug: "fist"},
	{ID: 10, Name: "Loyalty Points", Slug: "loyalty"},
	{ID: 11, Name: "Magic Level", Slug: "magic"},
	{ID: 12, Name: "Shielding", Slug: "shielding"},
	{ID: 13, Name: "Sword Fighting", Slug: "sword"},
	{ID: 14, Name: "Drome Score", Slug: "drome"},
	{ID: 15, Name: "Linked Tasks", Slug: "linked-tasks"},
	{ID: 16, Name: "Daily Experience (raw)", Slug: "daily-xp"},
	{ID: 18, Name: "Battle Pass", Slug: "battle-pass"},
	{ID: 19, Name: "Charm Points", Slug: "charm"},
	{ID: 20, Name: "Prestige Points", Slug: "prestige"},
	{ID: 21, Name: "Weekly Tasks", Slug: "weekly-tasks"},
	{ID: 22, Name: "Bounty Points", Slug: "bounty"},
}

var highscoreCategoryAliases = map[string]string{
	"exp":               "experience",
	"xp":                "experience",
	"distance-fighting": "distance",
	"sword-fighting":    "sword",
	"axe-fighting":      "axe",
	"club-fighting":     "club",
	"magic-level":       "magic",
}

var vocationAliases = map[string]string{
	"all":       "(all)",
	"(all)":     "(all)",
	"none":      "None",
	"knight":    "Knights",
	"knights":   "Knights",
	"ek":        "Knights",
	"paladin":   "Paladins",
	"paladins":  "Paladins",
	"rp":        "Paladins",
	"sorcerer":  "Sorcerers",
	"sorcerers": "Sorcerers",
	"ms":        "Sorcerers",
	"druid":     "Druids",
	"druids":    "Druids",
	"ed":        "Druids",
	"monk":      "Monks",
	"monks":     "Monks",
}

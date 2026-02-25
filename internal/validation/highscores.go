package validation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type HighscoreCategory struct {
	ID   int
	Name string
	Slug string
}

type HighscoreVocation struct {
	Name         string
	ProfessionID int
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

func (v *Validator) AllCategories() []HighscoreCategory {
	uniqueByID := make(map[int]HighscoreCategory)
	for _, category := range v.highscoreCategoriesByKey {
		uniqueByID[category.ID] = category
	}

	allCategories := make([]HighscoreCategory, 0, len(uniqueByID))
	for _, category := range uniqueByID {
		allCategories = append(allCategories, category)
	}

	sort.Slice(allCategories, func(i, j int) bool {
		return allCategories[i].ID < allCategories[j].ID
	})
	return allCategories
}

func (v *Validator) ResolveVocation(vocation string) (string, bool) {
	resolved, ok := v.vocationsByKey[normalizeLookupValue(vocation)]
	if !ok {
		return "", false
	}
	return resolved.Name, true
}

func (v *Validator) ResolveHighscoreVocation(vocation string) (HighscoreVocation, bool) {
	resolved, ok := v.vocationsByKey[normalizeLookupValue(vocation)]
	if !ok {
		return HighscoreVocation{}, false
	}
	return resolved, true
}

func (v *Validator) ReplaceHighscoreCategories(categories []HighscoreCategory) {
	if len(categories) == 0 {
		return
	}

	v.highscoreCategoriesByKey = make(map[string]HighscoreCategory)
	for _, category := range categories {
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
}

func ParseHighscoresCategoryOptions(html string) ([]HighscoreCategory, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	knownByID := make(map[int]HighscoreCategory)
	for _, category := range highscoreCategories {
		knownByID[category.ID] = category
	}

	uniqueByID := make(map[int]HighscoreCategory)
	doc.Find("select[name='category'] option").Each(func(_ int, option *goquery.Selection) {
		idRaw, ok := option.Attr("value")
		if !ok {
			return
		}

		id, convErr := strconv.Atoi(strings.TrimSpace(idRaw))
		if convErr != nil || id <= 0 {
			return
		}

		name := strings.TrimSpace(option.Text())
		if name == "" {
			return
		}

		if known, knownOK := knownByID[id]; knownOK {
			uniqueByID[id] = known
			return
		}

		slug := strings.ReplaceAll(normalizeLookupValue(name), " ", "-")
		uniqueByID[id] = HighscoreCategory{
			ID:   id,
			Name: name,
			Slug: slug,
		}
	})

	categories := make([]HighscoreCategory, 0, len(uniqueByID))
	for _, category := range uniqueByID {
		categories = append(categories, category)
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].ID < categories[j].ID
	})

	if len(categories) == 0 {
		return nil, fmt.Errorf("highscores category options are empty")
	}

	return categories, nil
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

	for canonical, professionID := range highscoreVocations {
		v.vocationsByKey[normalizeLookupValue(canonical)] = HighscoreVocation{
			Name:         canonical,
			ProfessionID: professionID,
		}
	}

	for alias, canonical := range vocationAliases {
		vocation, ok := v.vocationsByKey[normalizeLookupValue(canonical)]
		if !ok {
			continue
		}
		v.vocationsByKey[normalizeLookupValue(alias)] = vocation
	}
}

var highscoreCategories = []HighscoreCategory{
	{ID: 17, Name: "Achievements", Slug: "achievements"},
	{ID: 18, Name: "Battle Pass", Slug: "battle-pass"},
	{ID: 22, Name: "Bounty Points", Slug: "bounty"},
	{ID: 2, Name: "Axe Fighting", Slug: "axe"},
	{ID: 4, Name: "Club Fighting", Slug: "club"},
	{ID: 19, Name: "Charm Points", Slug: "charm"},
	{ID: 5, Name: "Distance Fighting", Slug: "distance"},
	{ID: 14, Name: "Drome Score", Slug: "drome"},
	{ID: 15, Name: "Linked Tasks", Slug: "linked-tasks"},
	{ID: 6, Name: "Experience Points", Slug: "experience"},
	{ID: 16, Name: "Daily Experience (raw)", Slug: "daily-xp"},
	{ID: 7, Name: "Fishing", Slug: "fishing"},
	{ID: 8, Name: "Fist Fighting", Slug: "fist"},
	{ID: 10, Name: "Loyalty Points", Slug: "loyalty"},
	{ID: 11, Name: "Magic Level", Slug: "magic"},
	{ID: 20, Name: "Prestige Points", Slug: "prestige"},
	{ID: 12, Name: "Shielding", Slug: "shielding"},
	{ID: 13, Name: "Sword Fighting", Slug: "sword"},
	{ID: 21, Name: "Weekly Tasks", Slug: "weekly-tasks"},
}

var highscoreCategoryAliases = map[string]string{
	"achievement":       "achievements",
	"exp":               "experience",
	"xp":                "experience",
	"distance-fighting": "distance",
	"sword-fighting":    "sword",
	"axe-fighting":      "axe",
	"club-fighting":     "club",
	"magic-level":       "magic",
}

var highscoreVocations = map[string]int{
	"(all)":     0,
	"None":      1,
	"Knights":   2,
	"Paladins":  3,
	"Sorcerers": 4,
	"Druids":    5,
	"Monks":     6,
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

package validation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type HighscoreCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
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
	{ID: 1, Name: "Experience", Slug: "experience"},
	{ID: 2, Name: "Magic Level", Slug: "magic"},
	{ID: 3, Name: "Shielding", Slug: "shielding"},
	{ID: 4, Name: "Distance", Slug: "distance"},
	{ID: 5, Name: "Sword", Slug: "sword"},
	{ID: 6, Name: "Axe", Slug: "axe"},
	{ID: 7, Name: "Club", Slug: "club"},
	{ID: 8, Name: "Fist", Slug: "fist"},
	{ID: 9, Name: "Fishing", Slug: "fishing"},
	{ID: 10, Name: "Drome Level", Slug: "dromelevel"},
	{ID: 11, Name: "Linked Tasks", Slug: "linked_tasks"},
	{ID: 12, Name: "Exp Today", Slug: "exp_today"},
	{ID: 13, Name: "Achievement Points", Slug: "achievements"},
	{ID: 14, Name: "Battle Pass", Slug: "battlepass"},
	{ID: 15, Name: "Charm Unlock Points", Slug: "charmunlockpoints"},
	{ID: 16, Name: "Prestige Points", Slug: "prestigepoints"},
	{ID: 17, Name: "Total Weekly Tasks", Slug: "totalweeklytasks"},
	{ID: 18, Name: "Total Bounty Points", Slug: "totalbountypoints"},
	{ID: 19, Name: "Charm Total Points", Slug: "charmtotalpoints"},
	{ID: 20, Name: "Boss Total Points", Slug: "bosstotalpoints"},
}

var highscoreCategoryAliases = map[string]string{
	"achievement":         "achievements",
	"achievement_points":  "achievements",
	"achievementpoints":   "achievements",
	"exp":                 "experience",
	"xp":                  "experience",
	"magiclevel":          "magic",
	"distance-fighting":   "distance",
	"dist":                "distance",
	"sword-fighting":      "sword",
	"axe-fighting":        "axe",
	"club-fighting":       "club",
	"drome":               "dromelevel",
	"drome_level":         "dromelevel",
	"linkedtasks":         "linked_tasks",
	"battle_pass":         "battlepass",
	"exp_today":           "exp_today",
	"exptoday":            "exp_today",
	"charm_unlock_points": "charmunlockpoints",
	"prestige_points":     "prestigepoints",
	"prestige":            "prestigepoints",
	"total_weekly_tasks":  "totalweeklytasks",
	"total_bounty_points": "totalbountypoints",
	"charm_total_points":  "charmtotalpoints",
	"boss_total_points":   "bosstotalpoints",
}

var highscoreVocations = map[string]int{
	"(all)":     0,
	"None":      1,
	"Sorcerers": 2,
	"Druids":    3,
	"Paladins":  4,
	"Knights":   5,
	"Monks":     9,
}

var vocationAliases = map[string]string{
	"all":       "(all)",
	"(all)":     "(all)",
	"none":      "None",
	"sorcerer":  "Sorcerers",
	"sorcerers": "Sorcerers",
	"ms":        "Sorcerers",
	"druid":     "Druids",
	"druids":    "Druids",
	"ed":        "Druids",
	"paladin":   "Paladins",
	"paladins":  "Paladins",
	"rp":        "Paladins",
	"knight":    "Knights",
	"knights":   "Knights",
	"ek":        "Knights",
	"monk":      "Monks",
	"monks":     "Monks",
}

package validation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type World struct {
	ID   int
	Name string
}

type Town struct {
	ID   int
	Name string
}

type Validator struct {
	worldsByKey map[string]World
	townsByKey  map[string]Town

	highscoreCategoriesByKey map[string]HighscoreCategory
	vocationsByKey           map[string]HighscoreVocation
}

func NewValidator(worlds []World, discoveredTowns ...Town) *Validator {
	validator := &Validator{
		worldsByKey:              make(map[string]World),
		townsByKey:               make(map[string]Town),
		highscoreCategoriesByKey: make(map[string]HighscoreCategory),
		vocationsByKey:           make(map[string]HighscoreVocation),
	}

	for _, world := range worlds {
		if world.ID <= 0 || strings.TrimSpace(world.Name) == "" {
			continue
		}
		key := normalizeLookupValue(world.Name)
		validator.worldsByKey[key] = World{ID: world.ID, Name: strings.TrimSpace(world.Name)}
	}

	townsToLoad := defaultTowns
	if len(discoveredTowns) > 0 {
		townsToLoad = discoveredTowns
	}

	for _, town := range townsToLoad {
		validator.townsByKey[normalizeLookupValue(town.Name)] = town
	}
	for alias, canonical := range townAliases {
		canonicalTown, ok := validator.townsByKey[normalizeLookupValue(canonical)]
		if !ok {
			continue
		}
		validator.townsByKey[normalizeLookupValue(alias)] = canonicalTown
	}

	validator.loadHighscores()
	return validator
}

func ParseLatestDeathsWorldOptions(html string) ([]World, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	worlds := make([]World, 0)
	doc.Find("select[name='world'] option").Each(func(_ int, option *goquery.Selection) {
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

		worlds = append(worlds, World{ID: id, Name: name})
	})

	if len(worlds) == 0 {
		return nil, fmt.Errorf("latest deaths world dropdown is empty")
	}

	return worlds, nil
}

func ParseHousesTownOptions(html string) ([]Town, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	uniqueByID := make(map[int]Town)
	doc.Find("label input[name='town']").Each(func(_ int, input *goquery.Selection) {
		idRaw, ok := input.Attr("value")
		if !ok {
			return
		}

		id, convErr := strconv.Atoi(strings.TrimSpace(idRaw))
		if convErr != nil || id <= 0 {
			return
		}

		name := strings.TrimSpace(input.Parent().Text())
		if name == "" {
			return
		}

		uniqueByID[id] = Town{ID: id, Name: name}
	})

	towns := make([]Town, 0, len(uniqueByID))
	for _, town := range uniqueByID {
		towns = append(towns, town)
	}

	sort.Slice(towns, func(i, j int) bool {
		return towns[i].ID < towns[j].ID
	})

	if len(towns) == 0 {
		return nil, fmt.Errorf("houses town options are empty")
	}

	return towns, nil
}

func (v *Validator) WorldExists(worldName string) (canonicalName string, worldID int, ok bool) {
	world, found := v.worldsByKey[normalizeLookupValue(worldName)]
	if !found {
		return "", 0, false
	}
	return world.Name, world.ID, true
}

func (v *Validator) TownExists(townName string) (canonicalName string, townID int, ok bool) {
	town, found := v.townsByKey[normalizeLookupValue(townName)]
	if !found {
		return "", 0, false
	}
	return town.Name, town.ID, true
}

func (v *Validator) AllTowns() []Town {
	uniqueByID := make(map[int]Town)
	for _, town := range v.townsByKey {
		uniqueByID[town.ID] = town
	}

	allTowns := make([]Town, 0, len(uniqueByID))
	for _, town := range uniqueByID {
		allTowns = append(allTowns, town)
	}

	sort.Slice(allTowns, func(i, j int) bool {
		return allTowns[i].ID < allTowns[j].ID
	})
	return allTowns
}

func (v *Validator) AllWorlds() []World {
	uniqueByID := make(map[int]World)
	for _, world := range v.worldsByKey {
		uniqueByID[world.ID] = world
	}

	allWorlds := make([]World, 0, len(uniqueByID))
	for _, world := range uniqueByID {
		allWorlds = append(allWorlds, world)
	}

	sort.Slice(allWorlds, func(i, j int) bool {
		return allWorlds[i].ID < allWorlds[j].ID
	})
	return allWorlds
}

var defaultTowns = []Town{
	{ID: 1, Name: "Venore"},
	{ID: 2, Name: "Thais"},
	{ID: 3, Name: "Kazordoon"},
	{ID: 4, Name: "Carlin"},
	{ID: 5, Name: "Ab Dendriel"},
	{ID: 7, Name: "Liberty Bay"},
	{ID: 8, Name: "Port Hope"},
	{ID: 9, Name: "Ankrahmun"},
	{ID: 10, Name: "Darashia"},
	{ID: 11, Name: "Edron"},
	{ID: 12, Name: "Svargrond"},
	{ID: 13, Name: "Yalahar"},
	{ID: 14, Name: "Farmine"},
	{ID: 33, Name: "Rathleton"},
	{ID: 63, Name: "Issavi"},
	{ID: 66, Name: "Moonfall"},
	{ID: 67, Name: "Silvertides"},
}

var townAliases = map[string]string{
	"ab'dendriel": "Ab Dendriel",
	"ab dendriel": "Ab Dendriel",
}

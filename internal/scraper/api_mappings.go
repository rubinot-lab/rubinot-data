package scraper

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

var worldMu sync.RWMutex

var worldIDToName = map[int]string{
	1:  "Elysian",
	9:  "Lunarian",
	10: "Spectrum",
	11: "Auroria",
	12: "Solarian",
	15: "Belaria",
	16: "Vesperia",
	17: "Bellum",
	18: "Mystian",
	21: "Tenebrium",
	22: "SerenianI",
	23: "SerenianII",
	24: "SerenianIII",
	25: "SerenianIV",
}

var worldNameToID = map[string]int{
	"elysian":     1,
	"lunarian":    9,
	"spectrum":    10,
	"auroria":     11,
	"solarian":    12,
	"belaria":     15,
	"vesperia":    16,
	"bellum":      17,
	"mystian":     18,
	"tenebrium":   21,
	"sereniani":   22,
	"serenianii":  23,
	"serenianiii": 24,
	"serenianiv":  25,
}

type WorldMapping struct {
	ID   int
	Name string
}

func UpdateWorldMappings(worlds []WorldMapping) {
	idToName := make(map[int]string, len(worlds))
	nameToID := make(map[string]int, len(worlds))
	for _, w := range worlds {
		if w.ID <= 0 || strings.TrimSpace(w.Name) == "" {
			continue
		}
		name := strings.TrimSpace(w.Name)
		idToName[w.ID] = name
		nameToID[strings.ToLower(name)] = w.ID
	}
	worldMu.Lock()
	worldIDToName = idToName
	worldNameToID = nameToID
	worldMu.Unlock()
}

var vocationIDToName = map[int]string{
	0: "None",
	1: "Sorcerer",
	2: "Druid",
	3: "Paladin",
	4: "Knight",
	5: "Master Sorcerer",
	6: "Elder Druid",
	7: "Royal Paladin",
	8: "Elite Knight",
	9: "Monk",
	10: "Exalted Monk",
}

func worldNameByID(id int) string {
	worldMu.RLock()
	value, ok := worldIDToName[id]
	worldMu.RUnlock()
	if ok {
		return value
	}
	return ""
}

func worldIDByName(name string) (int, bool) {
	worldMu.RLock()
	id, ok := worldNameToID[normalizeLookup(name)]
	worldMu.RUnlock()
	return id, ok
}

func vocationNameByID(id int) string {
	if value, ok := vocationIDToName[id]; ok {
		return value
	}
	return ""
}

func normalizeLookup(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func unixSecondsToRFC3339(seconds int64) string {
	if seconds <= 0 {
		return ""
	}
	return time.Unix(seconds, 0).UTC().Format(time.RFC3339)
}

func unixMillisToRFC3339(milliseconds int64) string {
	if milliseconds <= 0 {
		return ""
	}
	return time.UnixMilli(milliseconds).UTC().Format(time.RFC3339)
}

func unixTextToRFC3339(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return ""
	}

	if len(trimmed) >= 13 {
		return unixMillisToRFC3339(value)
	}
	return unixSecondsToRFC3339(value)
}

func unixAnyToRFC3339(raw any) string {
	switch value := raw.(type) {
	case int:
		return unixIntToRFC3339(int64(value))
	case int32:
		return unixIntToRFC3339(int64(value))
	case int64:
		return unixIntToRFC3339(value)
	case float32:
		return unixIntToRFC3339(int64(value))
	case float64:
		return unixIntToRFC3339(int64(value))
	case string:
		return unixTextToRFC3339(value)
	default:
		return ""
	}
}

func unixIntToRFC3339(value int64) string {
	if value <= 0 {
		return ""
	}
	if value >= 1_000_000_000_000 {
		return unixMillisToRFC3339(value)
	}
	return unixSecondsToRFC3339(value)
}

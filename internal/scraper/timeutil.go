package scraper

import (
	"fmt"
	"strings"
	"time"
)

var (
	rubinotBrazilLocation = mustLoadBrazilLocation()
	rubinotDateLayouts    = []string{
		"02/01/2006, 15:04:05",
		"2/1/2006, 15:04:05",
		"2 Jan 2006, 15:04:05",
		"02 Jan 2006, 15:04:05",
		"2.1.2006, 15:04:05",
		"02.01.2006, 15:04:05",
		"2.1.2006 15:04:05",
		"02.01.2006 15:04:05",
	}
)

func parseRubinotDateTimeToUTC(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}

	for _, layout := range rubinotDateLayouts {
		parsed, err := time.ParseInLocation(layout, value, rubinotBrazilLocation)
		if err == nil {
			return parsed.UTC().Format(time.RFC3339), nil
		}
	}

	return "", fmt.Errorf("unsupported rubinot datetime format: %q", raw)
}

func mustLoadBrazilLocation() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic(fmt.Sprintf("failed to load timezone America/Sao_Paulo: %v", err))
	}
	return loc
}

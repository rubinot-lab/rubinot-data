package scraper

import "strconv"

func parseInt(raw string) int {
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return parsed
}

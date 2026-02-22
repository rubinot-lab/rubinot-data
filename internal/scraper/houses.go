package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

var (
	houseListSizePattern = regexp.MustCompile(`(\d+)`)
	houseListRentPattern = regexp.MustCompile(`([\d,.]+)`)
)

func FetchHouses(
	ctx context.Context,
	baseURL string,
	worldName string,
	worldID int,
	townName string,
	townID int,
	opts FetchOptions,
) (domain.HousesResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchHouses")
	defer span.End()

	canonicalWorld := strings.TrimSpace(worldName)
	canonicalTown := strings.TrimSpace(townName)
	client := NewClient(opts)

	houseURL := fmt.Sprintf(
		"%s/?subtopic=houses&world=%d&town=%d&state=&type=houses&order=name",
		strings.TrimRight(baseURL, "/"),
		worldID,
		townID,
	)
	guildhallURL := fmt.Sprintf(
		"%s/?subtopic=houses&world=%d&town=%d&state=&type=guildhalls&order=name",
		strings.TrimRight(baseURL, "/"),
		worldID,
		townID,
	)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "houses"),
		attribute.String("rubinot.world", canonicalWorld),
		attribute.Int("rubinot.world_id", worldID),
		attribute.String("rubinot.town", canonicalTown),
		attribute.Int("rubinot.town_id", townID),
	)

	started := time.Now()
	housesHTML, err := client.Fetch(ctx, houseURL)
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		scrapeDuration.WithLabelValues("houses").Observe(time.Since(started).Seconds())
		return domain.HousesResult{}, houseURL, err
	}
	guildHTML, err := client.Fetch(ctx, guildhallURL)
	scrapeDuration.WithLabelValues("houses").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		return domain.HousesResult{}, houseURL, err
	}

	parseStart := time.Now()
	allEntries, err := parseHouseRows(housesHTML)
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		return domain.HousesResult{}, houseURL, err
	}
	guildhallsFromTypeQuery, err := parseHouseRows(guildHTML)
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		return domain.HousesResult{}, houseURL, err
	}
	parseDuration.WithLabelValues("houses").Observe(time.Since(parseStart).Seconds())
	scrapeRequests.WithLabelValues("houses", "ok").Inc()

	houses := make([]domain.HouseEntry, 0, len(allEntries))
	guildhallsFromHouses := make([]domain.HouseEntry, 0)
	for _, entry := range allEntries {
		if isGuildhallEntry(entry) {
			guildhallsFromHouses = append(guildhallsFromHouses, entry)
			continue
		}
		houses = append(houses, entry)
	}

	guildhalls := guildhallsFromTypeQuery
	if len(guildhalls) == 0 && len(guildhallsFromHouses) > 0 {
		// TODO: AMBIGUOUS — Rubinot currently returns guildhalls inside type=houses; keep fallback until upstream behavior is stable.
		guildhalls = guildhallsFromHouses
	}

	return domain.HousesResult{
		World:         canonicalWorld,
		Town:          canonicalTown,
		HouseList:     houses,
		GuildhallList: guildhalls,
	}, houseURL, nil
}

func parseHouseRows(html string) ([]domain.HouseEntry, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	out := make([]domain.HouseEntry, 0)
	doc.Find(".TableContentContainer .TableContent tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() < 4 {
			return
		}

		name := strings.TrimSpace(tds.Eq(0).Text())
		sizeText := strings.TrimSpace(tds.Eq(1).Text())
		rentText := strings.TrimSpace(tds.Eq(2).Text())
		statusRaw := strings.Join(strings.Fields(tds.Eq(3).Text()), " ")
		statusLower := strings.ToLower(statusRaw)

		if name == "" || strings.EqualFold(name, "name") {
			return
		}

		houseID := 0
		if v, ok := tr.Find("input[name='houseid']").Attr("value"); ok {
			houseID = parseInt(v)
		}

		entry := domain.HouseEntry{
			HouseID:     houseID,
			Name:        name,
			Size:        parseHouseSize(sizeText),
			Rent:        parseHouseRent(rentText),
			Status:      normalizeHouseStatus(statusLower),
			IsRented:    strings.Contains(statusLower, "rented"),
			IsAuctioned: strings.Contains(statusLower, "auction") || strings.Contains(statusLower, "no bid yet"),
		}

		if entry.Name != "" && entry.HouseID > 0 {
			out = append(out, entry)
		}
	})

	return out, nil
}

func normalizeHouseStatus(statusLower string) string {
	switch {
	case strings.Contains(statusLower, "rented"):
		return "rented"
	case strings.Contains(statusLower, "auction"):
		return "auctioned"
	case strings.Contains(statusLower, "no bid yet"):
		return "auctioned"
	default:
		return statusLower
	}
}

func isGuildhallEntry(entry domain.HouseEntry) bool {
	return strings.Contains(strings.ToLower(entry.Name), "guildhall")
}

func parseHouseSize(raw string) int {
	match := houseListSizePattern.FindStringSubmatch(raw)
	if len(match) < 2 {
		return 0
	}
	return parseInt(match[1])
}

func parseHouseRent(raw string) int {
	match := houseListRentPattern.FindStringSubmatch(raw)
	if len(match) < 2 {
		return 0
	}
	return parseInt(match[1])
}

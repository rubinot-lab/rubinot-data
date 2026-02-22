package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel/attribute"
)

type HouseEntry struct {
	HouseID     int    `json:"house_id"`
	Name        string `json:"name"`
	Size        int    `json:"size"`
	Rent        int    `json:"rent"`
	Status      string `json:"status"`
	IsRented    bool   `json:"rented"`
	IsAuctioned bool   `json:"auctioned"`
}

type HousesResult struct {
	World         string       `json:"world"`
	Town          string       `json:"town"`
	HouseList     []HouseEntry `json:"house_list"`
	GuildhallList []HouseEntry `json:"guildhall_list"`
}

func FetchHouses(ctx context.Context, baseURL, world, town string, opts FetchOptions) (HousesResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchHouses")
	defer span.End()

	formattedWorld := strings.Title(strings.ToLower(strings.TrimSpace(world)))
	formattedTown := formatTown(town)
	if strings.EqualFold(formattedTown, "ab'dendriel") {
		formattedTown = "Ab'Dendriel"
	}
	client := NewClient(opts)

	houseURL := fmt.Sprintf("%s/?subtopic=houses&world=%s&town=%s&type=houses", strings.TrimRight(baseURL, "/"), url.QueryEscape(formattedWorld), url.QueryEscape(formattedTown))
	guildhallURL := fmt.Sprintf("%s/?subtopic=houses&world=%s&town=%s&type=guildhalls", strings.TrimRight(baseURL, "/"), url.QueryEscape(formattedWorld), url.QueryEscape(formattedTown))

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "houses"),
		attribute.String("rubinot.world", formattedWorld),
		attribute.String("rubinot.town", formattedTown),
	)

	started := time.Now()
	housesHTML, err := client.Fetch(ctx, houseURL)
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		scrapeDuration.WithLabelValues("houses").Observe(time.Since(started).Seconds())
		return HousesResult{}, houseURL, err
	}
	guildHTML, err := client.Fetch(ctx, guildhallURL)
	scrapeDuration.WithLabelValues("houses").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		return HousesResult{}, houseURL, err
	}

	parseStart := time.Now()
	houses, err := parseHouseRows(housesHTML)
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		return HousesResult{}, houseURL, err
	}
	guildhalls, err := parseHouseRows(guildHTML)
	if err != nil {
		scrapeRequests.WithLabelValues("houses", "error").Inc()
		return HousesResult{}, houseURL, err
	}
	parseDuration.WithLabelValues("houses").Observe(time.Since(parseStart).Seconds())
	scrapeRequests.WithLabelValues("houses", "ok").Inc()

	return HousesResult{
		World:         formattedWorld,
		Town:          formattedTown,
		HouseList:     houses,
		GuildhallList: guildhalls,
	}, houseURL, nil
}

func parseHouseRows(html string) ([]HouseEntry, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	out := make([]HouseEntry, 0)
	doc.Find(".TableContentContainer .TableContent tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() < 4 {
			return
		}

		name := strings.TrimSpace(tds.Eq(0).Text())
		sizeText := strings.TrimSpace(tds.Eq(1).Text())
		rentText := strings.TrimSpace(tds.Eq(2).Text())
		statusText := strings.TrimSpace(strings.ToLower(tds.Eq(3).Text()))

		if name == "" || strings.EqualFold(name, "name") {
			return
		}

		houseID := 0
		if v, ok := tr.Find("input[name='houseid']").Attr("value"); ok {
			houseID = parseInt(v)
		}

		entry := HouseEntry{
			HouseID:     houseID,
			Name:        name,
			Size:        parseInt(sizeText),
			Rent:        parseInt(strings.ReplaceAll(strings.ReplaceAll(rentText, "gold", ""), "k", "000")),
			Status:      statusText,
			IsRented:    strings.Contains(statusText, "rented"),
			IsAuctioned: strings.Contains(statusText, "auction"),
		}

		if entry.Name != "" && entry.HouseID > 0 {
			out = append(out, entry)
		}
	})

	return out, nil
}

func formatTown(town string) string {
	town = strings.TrimSpace(strings.ReplaceAll(town, "+", " "))
	parts := strings.Fields(strings.ToLower(town))
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

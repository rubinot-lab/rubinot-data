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

var playersOnlinePattern = regexp.MustCompile(`(?i)([\d,.]+)\s+players online`)

func FetchWorlds(ctx context.Context, baseURL string, opts FetchOptions) (domain.WorldsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorlds")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/?subtopic=worlds", strings.TrimRight(baseURL, "/"))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "worlds"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("worlds").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("worlds", "error").Inc()
		return domain.WorldsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("worlds", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseWorldsHTML(htmlBody)
	parseDuration.WithLabelValues("worlds").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.WorldsResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func parseWorldsHTML(html string) (domain.WorldsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return domain.WorldsResult{}, err
	}

	result := domain.WorldsResult{
		TotalPlayersOnline: parseTotalPlayersOnline(doc),
		Worlds:             make([]domain.WorldOverview, 0),
	}

	targetTable := findWorldsTable(doc)
	if targetTable == nil || targetTable.Length() == 0 {
		return result, nil
	}

	targetTable.Find("tr").Slice(1, goquery.ToEnd).Each(func(_ int, row *goquery.Selection) {
		columns := row.Find("td")
		if columns.Length() < 4 {
			return
		}

		worldName := strings.TrimSpace(columns.Eq(0).Text())
		if worldName == "" {
			return
		}

		overview := domain.WorldOverview{
			Name:          worldName,
			Status:        "online",
			PlayersOnline: parseInt(strings.TrimSpace(columns.Eq(1).Text())),
			Location:      strings.TrimSpace(columns.Eq(2).Text()),
			PVPType:       strings.TrimSpace(columns.Eq(3).Text()),
		}

		result.Worlds = append(result.Worlds, overview)
	})

	return result, nil
}

func parseTotalPlayersOnline(doc *goquery.Document) int {
	totalPlayers := 0
	doc.Find(".InfoBarSmallElement").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		matches := playersOnlinePattern.FindStringSubmatch(item.Text())
		if len(matches) != 2 {
			return true
		}

		totalPlayers = parseInt(matches[1])
		return false
	})
	return totalPlayers
}

func findWorldsTable(doc *goquery.Document) *goquery.Selection {
	var worldsTable *goquery.Selection
	doc.Find("table.TableContent").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		header := strings.ToLower(strings.Join(strings.Fields(table.Find("tr").First().Text()), " "))
		if strings.Contains(header, "world") &&
			strings.Contains(header, "online") &&
			strings.Contains(header, "location") &&
			strings.Contains(header, "pvp type") {
			worldsTable = table
			return false
		}
		return true
	})
	return worldsTable
}

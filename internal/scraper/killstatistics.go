package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

func FetchKillstatistics(
	ctx context.Context,
	baseURL string,
	worldName string,
	worldID int,
	opts FetchOptions,
) (domain.KillstatisticsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchKillstatistics")
	defer span.End()

	started := time.Now()
	sourceURL := fmt.Sprintf("%s/?subtopic=killstatistics&world=%d", strings.TrimRight(baseURL, "/"), worldID)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "killstatistics"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
		attribute.String("rubinot.source_url", sourceURL),
	)

	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("killstatistics").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("killstatistics", "error").Inc()
		return domain.KillstatisticsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("killstatistics", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseKillstatisticsHTML(worldName, htmlBody)
	parseDuration.WithLabelValues("killstatistics").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.KillstatisticsResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func parseKillstatisticsHTML(worldName, htmlBody string) (domain.KillstatisticsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.KillstatisticsResult{}, err
	}

	result := domain.KillstatisticsResult{
		World:   worldName,
		Entries: make([]domain.KillstatisticsEntry, 0),
	}

	table := findKillstatisticsTable(doc)
	if table == nil {
		return result, nil
	}

	table.Find("tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 5 {
			return
		}

		race := normalizeText(cells.Eq(0).Text())
		if race == "" || strings.EqualFold(race, "race") {
			return
		}

		entry := domain.KillstatisticsEntry{
			Race:                  race,
			LastDayPlayersKilled:  parseInt(normalizeText(cells.Eq(1).Text())),
			LastDayKilled:         parseInt(normalizeText(cells.Eq(2).Text())),
			LastWeekPlayersKilled: parseInt(normalizeText(cells.Eq(3).Text())),
			LastWeekKilled:        parseInt(normalizeText(cells.Eq(4).Text())),
		}

		if strings.EqualFold(race, "total") {
			result.Total = domain.KillstatisticsTotal{
				LastDayPlayersKilled:  entry.LastDayPlayersKilled,
				LastDayKilled:         entry.LastDayKilled,
				LastWeekPlayersKilled: entry.LastWeekPlayersKilled,
				LastWeekKilled:        entry.LastWeekKilled,
			}
			return
		}

		result.Entries = append(result.Entries, entry)
	})

	return result, nil
}

func findKillstatisticsTable(doc *goquery.Document) *goquery.Selection {
	var target *goquery.Selection
	doc.Find("table.TableContent").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		content := strings.ToLower(normalizeText(table.Text()))
		if strings.Contains(content, "race") &&
			strings.Contains(content, "killed players") &&
			strings.Contains(content, "killed by players") &&
			strings.Contains(content, "last day") &&
			strings.Contains(content, "last week") {
			target = table
			return false
		}
		return true
	})
	return target
}

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

func FetchGuilds(ctx context.Context, baseURL, worldName string, worldID int, opts FetchOptions) (domain.GuildsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchGuilds")
	defer span.End()

	started := time.Now()
	sourceURL := fmt.Sprintf("%s/?subtopic=guilds&world=%d", strings.TrimRight(baseURL, "/"), worldID)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guilds"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
		attribute.String("rubinot.source_url", sourceURL),
	)

	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("guilds").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("guilds", "error").Inc()
		return domain.GuildsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("guilds", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseGuildsHTML(worldName, htmlBody)
	parseDuration.WithLabelValues("guilds").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.GuildsResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func parseGuildsHTML(worldName, htmlBody string) (domain.GuildsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.GuildsResult{}, err
	}

	result := domain.GuildsResult{
		World:     worldName,
		Active:    []domain.GuildListEntry{},
		Formation: []domain.GuildListEntry{},
	}

	if activeContainer := findContainerByHeaders(doc, []string{"active guilds"}); activeContainer != nil {
		result.Active = parseGuildListTable(activeContainer)
	}

	if formationContainer := findContainerByHeaders(doc, []string{"guilds in formation", "in formation"}); formationContainer != nil {
		result.Formation = parseGuildListTable(formationContainer)
	}

	if len(result.Active) == 0 && len(result.Formation) == 0 {
		fallback := parseGuildListTable(doc.Selection)
		result.Active = fallback
	}

	return result, nil
}

func parseGuildListTable(container *goquery.Selection) []domain.GuildListEntry {
	entries := make([]domain.GuildListEntry, 0)

	container.Find(".TableContent tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		nameCell := cells.Eq(1)
		nameLink := nameCell.Find("a[href*='GuildName=']").First()
		name := normalizeText(nameLink.Text())
		if name == "" {
			return
		}

		entry := domain.GuildListEntry{Name: name}
		if logoURL, exists := cells.Eq(0).Find("img").First().Attr("src"); exists {
			entry.LogoURL = strings.TrimSpace(logoURL)
		}
		if cells.Length() >= 3 {
			entry.Description = normalizeText(cells.Eq(2).Text())
		}

		entries = append(entries, entry)
	})

	return entries
}

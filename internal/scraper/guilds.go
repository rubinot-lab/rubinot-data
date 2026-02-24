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
	sourceURL := fmt.Sprintf("%s/?subtopic=guilds", strings.TrimRight(baseURL, "/"))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guilds"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
		attribute.String("rubinot.source_url", sourceURL),
	)

	fetchURL := buildFormSubmitDataURI(sourceURL, fmt.Sprintf("world=%d", worldID))
	htmlBody, err := client.Fetch(ctx, fetchURL)
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

	if activeContainer := findContainerByHeaders(doc, []string{"active guilds", "guilds ativas", "guilds activas"}); activeContainer != nil {
		result.Active = parseGuildListTable(activeContainer)
	}

	if formationContainer := findContainerByHeaders(doc, []string{"guilds in formation", "in formation", "guilds em formação", "guilds em formacao", "em formação", "em formacao"}); formationContainer != nil {
		result.Formation = parseGuildListTable(formationContainer)
	}

	if len(result.Active) == 0 && len(result.Formation) == 0 {
		fallback := parseGuildListTable(doc.Selection)
		result.Active = fallback
	}

	return result, nil
}

func buildFormSubmitDataURI(actionURL, formData string) string {
	target := strings.Replace(actionURL, "://www.", "://", 1)
	html := fmt.Sprintf(`<form method="post" action="%s">`, target)
	for _, pair := range strings.Split(formData, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			html += fmt.Sprintf(`<input name="%s" value="%s">`, parts[0], parts[1])
		}
	}
	html += `</form><script>document.forms[0].submit()</script>`
	return "data:text/html," + html
}

func parseGuildListTable(container *goquery.Selection) []domain.GuildListEntry {
	entries := make([]domain.GuildListEntry, 0)

	container.Find(".TableContent tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		nameCell := cells.Eq(1)
		name := normalizeText(nameCell.Find("b").First().Text())
		if name == "" {
			name = normalizeText(nameCell.Find("a").First().Text())
		}
		if name == "" {
			return
		}

		if hiddenName, exists := row.Find("input[name='GuildName']").First().Attr("value"); exists && strings.TrimSpace(hiddenName) != "" {
			name = strings.TrimSpace(hiddenName)
		}

		nameLower := strings.ToLower(name)
		cell0Lower := strings.ToLower(normalizeText(cells.Eq(0).Text()))
		if nameLower == "description" || nameLower == "name" || cell0Lower == "logo" {
			return
		}

		entry := domain.GuildListEntry{Name: name}
		if logoURL, exists := cells.Eq(0).Find("img").First().Attr("src"); exists {
			entry.LogoURL = strings.TrimSpace(logoURL)
		}

		fullCellText := normalizeText(nameCell.Text())
		description := strings.TrimSpace(strings.TrimPrefix(fullCellText, name))
		if description != "" {
			entry.Description = description
		}

		entries = append(entries, entry)
	})

	return entries
}

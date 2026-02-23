package scraper

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

var (
	deathRowIndexPattern = regexp.MustCompile(`^\d+\.$`)
	deathDatePattern     = regexp.MustCompile(`\d{2}\.\d{2}\.\d{4},\s*\d{2}:\d{2}:\d{2}`)
	deathLevelPattern    = regexp.MustCompile(`(?i)\bat level\s+(\d+)`)
	deathByPattern       = regexp.MustCompile(`(?i)\bby\s+(.+?)(?:\.\s*$|$)`)
	noDeathsPattern      = regexp.MustCompile(`(?i)no one died on`)
)

type DeathsFilters struct {
	Guild    string
	MinLevel int
	PvPOnly  *bool
}

func FetchDeaths(
	ctx context.Context,
	baseURL string,
	worldName string,
	worldID int,
	filters DeathsFilters,
	opts FetchOptions,
) (domain.DeathsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchDeaths")
	defer span.End()

	canonicalWorld := strings.TrimSpace(worldName)
	sourceURL := buildDeathsURL(baseURL, worldID, filters)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "deaths"),
		attribute.String("rubinot.world", canonicalWorld),
		attribute.Int("rubinot.world_id", worldID),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("deaths").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("deaths", "error").Inc()
		return domain.DeathsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("deaths", "ok").Inc()

	parseStart := time.Now()
	result, parseErr := parseDeathsHTML(canonicalWorld, filters, htmlBody)
	parseDuration.WithLabelValues("deaths").Observe(time.Since(parseStart).Seconds())
	if parseErr != nil {
		return domain.DeathsResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func buildDeathsURL(baseURL string, worldID int, filters DeathsFilters) string {
	sourceURL := fmt.Sprintf("%s/?subtopic=latestdeaths&world=%d", strings.TrimRight(baseURL, "/"), worldID)
	if filters.Guild != "" {
		sourceURL += "&guild=" + url.QueryEscape(filters.Guild)
	}
	if filters.MinLevel > 0 {
		sourceURL += fmt.Sprintf("&level=%d", filters.MinLevel)
	}
	if filters.PvPOnly != nil && *filters.PvPOnly {
		sourceURL += "&pvp=1"
	}
	return sourceURL
}

func parseDeathsHTML(worldName string, filters DeathsFilters, html string) (domain.DeathsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return domain.DeathsResult{}, err
	}

	result := domain.DeathsResult{
		World: worldName,
		Filters: domain.DeathFilters{
			Guild:    filters.Guild,
			MinLevel: filters.MinLevel,
			PvPOnly:  filters.PvPOnly,
		},
		Entries: make([]domain.DeathEntry, 0),
	}

	if noDeathsPattern.MatchString(strings.ToLower(doc.Text())) {
		return result, nil
	}

	rows := findDeathRows(doc)
	if len(rows) == 0 {
		return domain.DeathsResult{}, validation.NewError(validation.ErrorUpstreamUnknown, "deaths table not found", nil)
	}

	for _, row := range rows {
		tds := row.Find("td")
		if tds.Length() < 3 {
			continue
		}

		dateRaw := strings.TrimSpace(tds.Eq(1).Text())
		dateUTC, dateErr := parseRubinotDateTimeToUTC(dateRaw)
		if dateErr != nil {
			return domain.DeathsResult{}, validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("invalid death date %q", dateRaw), dateErr)
		}

		deathCell := tds.Eq(2)
		links := deathCell.Find("a")
		victimName := strings.TrimSpace(links.First().Text())
		if victimName == "" {
			continue
		}

		text := strings.Join(strings.Fields(deathCell.Text()), " ")
		level := 0
		levelMatch := deathLevelPattern.FindStringSubmatch(text)
		if len(levelMatch) == 2 {
			level = parseInt(levelMatch[1])
		}

		playerKillers := make([]string, 0)
		if links.Length() > 1 {
			links.Slice(1, goquery.ToEnd).Each(func(_ int, link *goquery.Selection) {
				killer := strings.TrimSpace(link.Text())
				if killer != "" {
					playerKillers = append(playerKillers, killer)
				}
			})
		}

		killers := playerKillers
		if len(killers) == 0 {
			killers = parseMonsterKillers(text)
		}

		entry := domain.DeathEntry{
			Date: dateUTC,
			Victim: domain.DeathVictim{
				Name:  victimName,
				Level: level,
			},
			Killers: killers,
			IsPvP:   len(playerKillers) > 0,
		}
		result.Entries = append(result.Entries, entry)
	}

	result.TotalDeaths = len(result.Entries)
	return result, nil
}

func findDeathRows(doc *goquery.Document) []*goquery.Selection {
	rows := make([]*goquery.Selection, 0)
	doc.Find("table.TableContent").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		candidateRows := table.Find("tr[bgcolor]")
		if candidateRows.Length() == 0 {
			return true
		}

		first := candidateRows.First()
		tds := first.Find("td")
		if tds.Length() < 3 {
			return true
		}

		indexText := strings.TrimSpace(tds.Eq(0).Text())
		dateText := strings.TrimSpace(tds.Eq(1).Text())
		if !deathRowIndexPattern.MatchString(indexText) || !deathDatePattern.MatchString(dateText) {
			return true
		}

		candidateRows.Each(func(_ int, row *goquery.Selection) {
			rows = append(rows, row)
		})
		return false
	})
	return rows
}

func parseMonsterKillers(text string) []string {
	byMatch := deathByPattern.FindStringSubmatch(text)
	if len(byMatch) != 2 {
		return []string{}
	}

	rawKillers := strings.TrimSpace(byMatch[1])
	if rawKillers == "" {
		return []string{}
	}

	candidate := strings.ReplaceAll(rawKillers, " and ", ",")
	parts := strings.Split(candidate, ",")
	killers := make([]string, 0, len(parts))
	for _, part := range parts {
		killer := strings.TrimSpace(strings.TrimSuffix(part, "."))
		if killer == "" {
			continue
		}
		killers = append(killers, killer)
	}
	return killers
}

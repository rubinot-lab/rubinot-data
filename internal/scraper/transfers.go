package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

var (
	transfersResultsPattern = regexp.MustCompile(`(?i)results:\s*([\d,.]+)`)
	transfersEmptyPattern   = regexp.MustCompile(`(?i)no recent transfers found`)
)

type TransfersFilters struct {
	WorldID   int
	WorldName string
	MinLevel  int
	Page      int
}

func FetchTransfers(
	ctx context.Context,
	baseURL string,
	filters TransfersFilters,
	opts FetchOptions,
) (domain.TransfersResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchTransfers")
	defer span.End()

	sourceURL := buildTransfersURL(baseURL, filters)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "transfers"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.world", filters.WorldName),
		attribute.Int("rubinot.world_id", filters.WorldID),
		attribute.Int("rubinot.page", filters.Page),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("transfers").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("transfers", "error").Inc()
		return domain.TransfersResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("transfers", "ok").Inc()

	parseStart := time.Now()
	result, parseErr := parseTransfersHTML(filters, htmlBody)
	parseDuration.WithLabelValues("transfers").Observe(time.Since(parseStart).Seconds())
	if parseErr != nil {
		return domain.TransfersResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func buildTransfersURL(baseURL string, filters TransfersFilters) string {
	sourceURL := fmt.Sprintf("%s/?subtopic=transferstatistics", strings.TrimRight(baseURL, "/"))
	// TODO: AMBIGUOUS — upstream filter names changed over time; keep world/level/currentpage compatibility from existing contract.
	if filters.WorldID > 0 {
		sourceURL += fmt.Sprintf("&world=%d", filters.WorldID)
	}
	if filters.MinLevel > 0 {
		sourceURL += fmt.Sprintf("&level=%d", filters.MinLevel)
	}
	if filters.Page > 1 {
		sourceURL += fmt.Sprintf("&currentpage=%d", filters.Page)
	}
	return sourceURL
}

func parseTransfersHTML(filters TransfersFilters, html string) (domain.TransfersResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return domain.TransfersResult{}, err
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	result := domain.TransfersResult{
		Filters: domain.TransferFilters{
			World:    strings.TrimSpace(filters.WorldName),
			MinLevel: filters.MinLevel,
		},
		Page:    page,
		Entries: make([]domain.TransferEntry, 0),
	}

	if transfersEmptyPattern.MatchString(strings.ToLower(doc.Text())) {
		result.TotalTransfers = parseTransfersTotal(doc)
		return result, nil
	}

	table := findTransfersTable(doc)
	if table == nil || table.Length() == 0 {
		return domain.TransfersResult{}, validation.NewError(validation.ErrorUpstreamUnknown, "transfers table not found", nil)
	}

	var parseErr error
	table.Find("tr[bgcolor]").EachWithBreak(func(_ int, row *goquery.Selection) bool {
		tds := row.Find("td")
		if tds.Length() < 5 {
			return true
		}

		playerName := strings.TrimSpace(tds.Eq(0).Text())
		level := parseInt(strings.TrimSpace(tds.Eq(1).Text()))
		formerWorld := strings.TrimSpace(tds.Eq(2).Text())
		destinationWorld := strings.TrimSpace(tds.Eq(3).Text())
		dateRaw := strings.TrimSpace(tds.Eq(4).Text())
		if playerName == "" || dateRaw == "" {
			return true
		}

		transferDate, dateErr := parseRubinotDateTimeToUTC(dateRaw)
		if dateErr != nil {
			parseErr = validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("invalid transfer date %q", dateRaw), dateErr)
			return false
		}

		entry := domain.TransferEntry{
			PlayerName:       playerName,
			Level:            level,
			FormerWorld:      formerWorld,
			DestinationWorld: destinationWorld,
			TransferDate:     transferDate,
		}
		result.Entries = append(result.Entries, entry)
		return true
	})

	if parseErr != nil {
		return domain.TransfersResult{}, parseErr
	}

	total := parseTransfersTotal(doc)
	if total == 0 {
		total = len(result.Entries)
	}
	result.TotalTransfers = total
	return result, nil
}

func findTransfersTable(doc *goquery.Document) *goquery.Selection {
	var table *goquery.Selection
	doc.Find("table.TableContent").EachWithBreak(func(_ int, candidate *goquery.Selection) bool {
		header := strings.ToLower(strings.Join(strings.Fields(candidate.Find("tr").First().Text()), " "))
		if strings.Contains(header, "player name") &&
			strings.Contains(header, "former world") &&
			strings.Contains(header, "destination world") &&
			strings.Contains(header, "transfer date") {
			table = candidate
			return false
		}
		return true
	})
	return table
}

func parseTransfersTotal(doc *goquery.Document) int {
	match := transfersResultsPattern.FindStringSubmatch(doc.Text())
	if len(match) < 2 {
		return 0
	}
	return parseInt(match[1])
}

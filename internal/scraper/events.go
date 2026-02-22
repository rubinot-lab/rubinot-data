package scraper

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

var (
	eventsMonthYearPattern = regexp.MustCompile(`([A-Za-z]+)\s+(\d{4})`)
)

func FetchEventsSchedule(
	ctx context.Context,
	baseURL string,
	month int,
	year int,
	opts FetchOptions,
) (domain.EventsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchEventsSchedule")
	defer span.End()

	sourceURL := buildEventsURL(baseURL, month, year)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "events"),
		attribute.Int("rubinot.month", month),
		attribute.Int("rubinot.year", year),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("events").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("events", "error").Inc()
		return domain.EventsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("events", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseEventsHTML(htmlBody)
	parseDuration.WithLabelValues("events").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.EventsResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func buildEventsURL(baseURL string, month int, year int) string {
	sourceURL := fmt.Sprintf("%s/?subtopic=eventcalendar", strings.TrimRight(baseURL, "/"))
	if month > 0 && year > 0 {
		sourceURL += fmt.Sprintf("&calendarmonth=%d&calendaryear=%d", month, year)
	}
	return sourceURL
}

func parseEventsHTML(htmlBody string) (domain.EventsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.EventsResult{}, err
	}

	result := domain.EventsResult{
		Days:      make([]domain.EventDay, 0),
		AllEvents: make([]string, 0),
	}

	headerText := normalizeText(doc.Find(".eventscheduleheaderdateblock").First().Text())
	if match := eventsMonthYearPattern.FindStringSubmatch(headerText); len(match) == 3 {
		result.Month = match[1]
		result.Year = parseInt(match[2])
	}

	lastUpdateRaw := normalizeText(doc.Find(".eventscheduleheaderblockright").First().Text())
	if parsed, parseErr := parseRubinotEventLastUpdateToUTC(lastUpdateRaw); parseErr == nil {
		result.LastUpdate = parsed
	}

	table := findEventsCalendarTable(doc)
	if table == nil || table.Length() == 0 {
		return domain.EventsResult{}, validation.NewError(validation.ErrorUpstreamUnknown, "event calendar table not found", nil)
	}

	allEventsSet := make(map[string]struct{})
	table.Find("tr").Slice(1, goquery.ToEnd).Each(func(_ int, row *goquery.Selection) {
		row.Find("td").Each(func(_ int, cell *goquery.Selection) {
			dayEntry, ok := parseEventDayCell(cell)
			if !ok {
				return
			}
			for _, name := range dayEntry.Events {
				allEventsSet[name] = struct{}{}
			}
			result.Days = append(result.Days, dayEntry)
		})
	})

	if len(allEventsSet) > 0 {
		for eventName := range allEventsSet {
			result.AllEvents = append(result.AllEvents, eventName)
		}
		sort.Strings(result.AllEvents)
	}

	return result, nil
}

func findEventsCalendarTable(doc *goquery.Document) *goquery.Selection {
	if table := doc.Find("#eventscheduletable"); table.Length() > 0 {
		return table.First()
	}

	var target *goquery.Selection
	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		header := strings.ToLower(normalizeText(table.Find("tr").First().Text()))
		if strings.Contains(header, "monday") && strings.Contains(header, "sunday") {
			target = table
			return false
		}
		return true
	})
	return target
}

func parseEventDayCell(cell *goquery.Selection) (domain.EventDay, bool) {
	lines := splitEventCellLines(cell.Text())
	if len(lines) == 0 {
		return domain.EventDay{}, false
	}

	day, err := strconv.Atoi(lines[0])
	if err != nil || day <= 0 {
		return domain.EventDay{}, false
	}

	events := make([]string, 0)
	activeEvents := make([]string, 0)
	endingEvents := make([]string, 0)

	for _, raw := range lines[1:] {
		if _, err := strconv.Atoi(raw); err == nil {
			continue
		}

		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		isEnding := strings.HasPrefix(trimmed, "*")
		eventName := strings.TrimSpace(strings.TrimPrefix(trimmed, "*"))
		if eventName == "" {
			continue
		}

		events = append(events, eventName)
		if isEnding {
			endingEvents = append(endingEvents, eventName)
		} else {
			activeEvents = append(activeEvents, eventName)
		}
	}

	if len(events) == 0 {
		return domain.EventDay{}, false
	}

	return domain.EventDay{
		Day:          day,
		Events:       events,
		ActiveEvents: activeEvents,
		EndingEvents: endingEvents,
	}, true
}

func splitEventCellLines(raw string) []string {
	rawLines := strings.Split(raw, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		normalized := strings.TrimSpace(line)
		if normalized == "" {
			continue
		}
		lines = append(lines, normalized)
	}
	return lines
}

func parseRubinotEventLastUpdateToUTC(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}

	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, rubinotBrazilLocation)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

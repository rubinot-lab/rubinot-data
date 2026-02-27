package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

var monthYearPattern = regexp.MustCompile(`(?i)([\p{L}]+)\s+(\d{4})`)

func FetchEventsSchedule(
	ctx context.Context,
	baseURL string,
	month int,
	year int,
	opts FetchOptions,
) (domain.EventsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchEventsSchedule")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/events", strings.TrimRight(baseURL, "/"))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "events"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.Int("rubinot.month", month),
		attribute.Int("rubinot.year", year),
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
		ParseErrors.WithLabelValues("events", "parse").Inc()
		return domain.EventsResult{}, sourceURL, parseErr
	}
	ParseItems.WithLabelValues("events").Set(float64(len(result.Days)))

	if month > 0 {
		result.Days = filterCurrentMonthDays(result.Days)
	}
	if year > 0 {
		result.Year = year
	}
	if month > 0 {
		result.Month = strconv.Itoa(month)
	}

	return result, sourceURL, nil
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

	monthText := strings.TrimSpace(doc.Find("span.flex.items-center.gap-2").First().Text())
	if monthText == "" {
		monthText = strings.TrimSpace(doc.Find("h2, h3").First().Text())
	}
	if month, year := parseMonthYear(monthText); year > 0 {
		result.Month = month
		result.Year = year
	} else {
		now := time.Now().UTC()
		result.Month = now.Month().String()
		result.Year = now.Year()
	}

	calendar := doc.Find("table.w-full.table-fixed.border-collapse.text-xs")
	if calendar.Length() == 0 {
		calendar = doc.Find("table").First()
	}
	if calendar.Length() == 0 {
		return result, nil
	}

	uniqueEvents := make(map[string]struct{})
	calendar.Find("tr").Slice(1, goquery.ToEnd).Each(func(_ int, row *goquery.Selection) {
		row.Find("td").Each(func(_ int, cell *goquery.Selection) {
			if strings.Contains(cell.AttrOr("class", ""), "calendar-other-month") {
				return
			}

			day, ok := parseEventDayCell(cell)
			if !ok {
				return
			}
			for _, event := range day.Events {
				if _, exists := uniqueEvents[event]; exists {
					continue
				}
				uniqueEvents[event] = struct{}{}
				result.AllEvents = append(result.AllEvents, event)
			}
			result.Days = append(result.Days, day)
		})
	})

	sortEventDays(result.Days)
	return result, nil
}

func parseMonthYear(raw string) (string, int) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", 0
	}

	matches := monthYearPattern.FindStringSubmatch(trimmed)
	if len(matches) != 3 {
		return "", 0
	}
	year, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", 0
	}
	return strings.TrimSpace(matches[1]), year
}

func filterCurrentMonthDays(days []domain.EventDay) []domain.EventDay {
	filtered := make([]domain.EventDay, 0, len(days))
	for _, day := range days {
		if day.Day <= 0 || day.Day > 31 {
			continue
		}
		filtered = append(filtered, day)
	}
	return filtered
}

func parseEventDayCell(cell *goquery.Selection) (domain.EventDay, bool) {
	events := make([]string, 0)
	dayValue := 0

	divs := cell.Find("div")
	if divs.Length() > 0 {
		dayText := strings.TrimSpace(divs.First().Text())
		parsedDay, err := strconv.Atoi(dayText)
		if err == nil {
			dayValue = parsedDay
		}
		divs.Slice(1, goquery.ToEnd).Each(func(_ int, div *goquery.Selection) {
			cleaned := strings.TrimSpace(strings.TrimPrefix(div.Text(), "*"))
			if cleaned == "" {
				return
			}
			events = append(events, cleaned)
		})
	}

	if dayValue == 0 {
		textLines := splitEventCellLines(cell.Text())
		if len(textLines) == 0 {
			return domain.EventDay{}, false
		}
		parsedDay, err := strconv.Atoi(textLines[0])
		if err != nil {
			return domain.EventDay{}, false
		}
		dayValue = parsedDay
		for _, line := range textLines[1:] {
			cleaned := strings.TrimSpace(strings.TrimPrefix(line, "*"))
			if cleaned == "" {
				continue
			}
			events = append(events, cleaned)
		}
	}

	if dayValue <= 0 || dayValue > 31 {
		return domain.EventDay{}, false
	}

	unique := make(map[string]struct{})
	normalizedEvents := make([]string, 0, len(events))
	for _, event := range events {
		if _, exists := unique[event]; exists {
			continue
		}
		unique[event] = struct{}{}
		normalizedEvents = append(normalizedEvents, event)
	}

	return domain.EventDay{
		Day:          dayValue,
		Events:       normalizedEvents,
		ActiveEvents: normalizedEvents,
		EndingEvents: make([]string, 0),
	}, true
}

func splitEventCellLines(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := strings.TrimSpace(part)
		if normalized == "" {
			continue
		}
		lines = append(lines, normalized)
	}
	return lines
}

func sortEventDays(days []domain.EventDay) {
	for i := 0; i < len(days)-1; i++ {
		for j := i + 1; j < len(days); j++ {
			if days[j].Day < days[i].Day {
				days[i], days[j] = days[j], days[i]
			}
		}
	}
}

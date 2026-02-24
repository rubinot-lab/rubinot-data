package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

func FetchWorld(ctx context.Context, baseURL, world string, opts FetchOptions) (domain.WorldResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorld")
	defer span.End()

	started := time.Now()
	canonicalWorld := strings.TrimSpace(world)
	sourceURL := fmt.Sprintf("%s/?subtopic=worlds&world=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(canonicalWorld))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "world"),
		attribute.String("rubinot.world", canonicalWorld),
		attribute.String("rubinot.source_url", sourceURL),
	)

	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("world").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("world", "error").Inc()
		return domain.WorldResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("world", "ok").Inc()

	parseStart := time.Now()
	result, source, parseErr := parseWorldHTML(canonicalWorld, sourceURL, htmlBody)
	parseDuration.WithLabelValues("world").Observe(time.Since(parseStart).Seconds())
	if parseErr != nil {
		return domain.WorldResult{}, sourceURL, parseErr
	}

	return result, source, nil
}

func parseWorldHTML(formatted, sourceURL, htmlBody string) (domain.WorldResult, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.WorldResult{}, sourceURL, err
	}

	result := domain.WorldResult{Name: formatted}
	worldInfo, found, parseInfoErr := parseWorldInfo(doc)
	if parseInfoErr != nil {
		return domain.WorldResult{}, sourceURL, validation.NewError(validation.ErrorUpstreamUnknown, parseInfoErr.Error(), parseInfoErr)
	}
	if !found {
		return domain.WorldResult{}, sourceURL, validation.NewError(validation.ErrorEntityNotFound, "world not found", nil)
	}
	result.Info = worldInfo
	result.PlayersOnline = parsePlayers(doc)

	if strings.EqualFold(result.Info.Status, "offline") {
		result.PlayersOnline = []domain.PlayerOnline{}
	}
	if result.Info.PlayersOnline == 0 {
		result.Info.PlayersOnline = len(result.PlayersOnline)
	}

	return result, sourceURL, nil
}

func parseWorldInfo(doc *goquery.Document) (domain.WorldInfo, bool, error) {
	info := domain.WorldInfo{}
	found := false
	var creationDateRaw string
	doc.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() != 2 {
			return
		}
		label := strings.TrimSpace(strings.TrimSuffix(tds.Eq(0).Text(), ":"))
		value := strings.TrimSpace(tds.Eq(1).Text())
		switch strings.ToLower(label) {
		case "status":
			info.Status = strings.ToLower(value)
			found = true
		case "players online":
			info.PlayersOnline = parseInt(value)
		case "location":
			info.Location = value
		case "pvp type":
			info.PVPType = value
		case "creation date":
			creationDateRaw = value
		}
	})

	if creationDateRaw != "" {
		normalizedDate, err := parseRubinotDateToUTC(creationDateRaw)
		if err != nil {
			return domain.WorldInfo{}, false, fmt.Errorf("invalid world creation date %q: %w", creationDateRaw, err)
		}
		info.CreationDate = normalizedDate
	}

	return info, found, nil
}

func parsePlayers(doc *goquery.Document) []domain.PlayerOnline {
	out := make([]domain.PlayerOnline, 0)
	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		headers := strings.ToLower(strings.Join(strings.Fields(table.Find("tr").First().Text()), " "))
		if !(strings.Contains(headers, "name") && strings.Contains(headers, "level") && strings.Contains(headers, "vocation")) {
			return true
		}

		table.Find("tr").Slice(1, goquery.ToEnd).Each(func(_ int, tr *goquery.Selection) {
			tds := tr.Find("td")
			if tds.Length() < 3 {
				return
			}
			name := strings.TrimSpace(tds.Eq(0).Text())
			lvl := parseInt(strings.TrimSpace(tds.Eq(1).Text()))
			voc := strings.TrimSpace(tds.Eq(2).Text())
			if name == "" || lvl <= 0 || voc == "" {
				return
			}
			out = append(out, domain.PlayerOnline{Name: name, Level: lvl, Vocation: voc})
		})
		return false
	})
	return out
}

func parseInt(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, " ", "")
	i, _ := strconv.Atoi(s)
	return i
}

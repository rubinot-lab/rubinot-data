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
	"go.opentelemetry.io/otel/attribute"
)

func FetchWorld(ctx context.Context, baseURL, world string, opts FetchOptions) (domain.WorldResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorld")
	defer span.End()

	started := time.Now()
	formatted := strings.Title(strings.ToLower(strings.TrimSpace(world)))
	sourceURL := fmt.Sprintf("%s/?subtopic=worlds&world=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(formatted))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "world"),
		attribute.String("rubinot.world", formatted),
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
	result, source, parseErr := parseWorldHTML(formatted, sourceURL, htmlBody)
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
	result.Info = parseWorldInfo(doc)
	result.PlayersOnline = parsePlayers(doc)

	if result.Info.PlayersOnline == 0 {
		result.Info.PlayersOnline = len(result.PlayersOnline)
	}

	return result, sourceURL, nil
}

func parseWorldInfo(doc *goquery.Document) domain.WorldInfo {
	info := domain.WorldInfo{}
	doc.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() != 2 {
			return
		}
		label := strings.TrimSpace(strings.TrimSuffix(tds.Eq(0).Text(), ":"))
		value := strings.TrimSpace(tds.Eq(1).Text())
		switch strings.ToLower(label) {
		case "status":
			info.Status = value
		case "players online":
			info.PlayersOnline = parseInt(value)
		case "location":
			info.Location = value
		case "pvp type":
			info.PVPType = value
		case "creation date":
			info.CreationDate = value
		}
	})
	return info
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
	i, _ := strconv.Atoi(s)
	return i
}

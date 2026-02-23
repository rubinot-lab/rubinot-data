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
	banishmentsHeaderPattern    = regexp.MustCompile(`(?i)\bplayer\b.*\breason\b.*\bbanned\b.*\bexpires\b`)
	banishmentsPermanentPattern = regexp.MustCompile(`(?i)lifetime|permanent|deletion`)
)

func FetchBanishments(
	ctx context.Context,
	baseURL string,
	worldName string,
	worldID int,
	page int,
	opts FetchOptions,
) (domain.BanishmentsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchBanishments")
	defer span.End()

	if page < 1 {
		page = 1
	}

	sourceURL := buildBanishmentsURL(baseURL, worldID, page)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "banishments"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
		attribute.Int("rubinot.page", page),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("banishments").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("banishments", "error").Inc()
		return domain.BanishmentsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("banishments", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseBanishmentsHTML(worldName, page, htmlBody)
	parseDuration.WithLabelValues("banishments").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.BanishmentsResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func buildBanishmentsURL(baseURL string, worldID int, page int) string {
	sourceURL := fmt.Sprintf("%s/?subtopic=bans&world=%d", strings.TrimRight(baseURL, "/"), worldID)
	if page > 1 {
		sourceURL += fmt.Sprintf("&currentpage=%d", page)
	}
	return sourceURL
}

func parseBanishmentsHTML(worldName string, page int, html string) (domain.BanishmentsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return domain.BanishmentsResult{}, err
	}

	if page < 1 {
		page = 1
	}
	result := domain.BanishmentsResult{
		World:   strings.TrimSpace(worldName),
		Page:    page,
		Entries: make([]domain.BanishmentEntry, 0),
	}

	table := findBanishmentsTable(doc)
	if table == nil || table.Length() == 0 {
		if isBanishmentsSelectionPage(doc) {
			// TODO: AMBIGUOUS — current live /?subtopic=bans HTML renders only world selection (no bans table) for all tested worlds.
			return result, nil
		}
		return domain.BanishmentsResult{}, validation.NewError(validation.ErrorUpstreamUnknown, "banishments table not found", nil)
	}

	var parseRowErr error
	table.Find("tr[bgcolor]").EachWithBreak(func(_ int, row *goquery.Selection) bool {
		entry, ok, rowErr := parseBanishmentRow(row)
		if rowErr != nil {
			parseRowErr = rowErr
			return false
		}
		if !ok {
			return true
		}
		result.Entries = append(result.Entries, entry)
		return true
	})

	if parseRowErr != nil {
		return domain.BanishmentsResult{}, parseRowErr
	}

	result.TotalBans = len(result.Entries)
	return result, nil
}

func findBanishmentsTable(doc *goquery.Document) *goquery.Selection {
	var target *goquery.Selection
	doc.Find("table.TableContent").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		header := strings.ToLower(normalizeText(table.Find("tr").First().Text()))
		if banishmentsHeaderPattern.MatchString(header) {
			target = table
			return false
		}
		return true
	})
	return target
}

func isBanishmentsSelectionPage(doc *goquery.Document) bool {
	return doc.Find("#bans").Length() > 0 &&
		doc.Find("form[action*='subtopic=bans']").Length() > 0 &&
		doc.Find("select[name='world']").Length() > 0
}

func parseBanishmentRow(row *goquery.Selection) (domain.BanishmentEntry, bool, error) {
	tds := row.Find("td")
	if tds.Length() < 4 {
		return domain.BanishmentEntry{}, false, nil
	}

	character := normalizeText(tds.Eq(0).Find("a").First().Text())
	if character == "" {
		character = normalizeText(tds.Eq(0).Text())
	}
	reason := normalizeText(tds.Eq(1).Text())
	bannedRaw := normalizeText(tds.Eq(2).Text())
	duration := normalizeText(tds.Eq(3).Text())

	if character == "" || reason == "" || bannedRaw == "" || duration == "" {
		return domain.BanishmentEntry{}, false, nil
	}

	bannedAt, err := parseRubinotDateTimeToUTC(bannedRaw)
	if err != nil {
		bannedAt, err = parseRubinotDateToUTC(bannedRaw)
		if err != nil {
			return domain.BanishmentEntry{}, false, validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("invalid banished date %q", bannedRaw), err)
		}
	}

	entry := domain.BanishmentEntry{
		Date:        bannedAt,
		Character:   character,
		Reason:      reason,
		Duration:    duration,
		IsPermanent: banishmentsPermanentPattern.MatchString(duration),
	}

	if !entry.IsPermanent {
		if expiresAt, expiresErr := parseRubinotDateTimeToUTC(duration); expiresErr == nil {
			entry.ExpiresAt = expiresAt
		} else if expiresAt, dateErr := parseRubinotDateToUTC(duration); dateErr == nil {
			entry.ExpiresAt = expiresAt
		}
	}

	return entry, true, nil
}

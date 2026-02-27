package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type banishmentsAPIResponse struct {
	Bans []struct {
		AccountID     int    `json:"account_id"`
		AccountName   string `json:"account_name"`
		MainCharacter string `json:"main_character"`
		Reason        string `json:"reason"`
		BannedAt      string `json:"banned_at"`
		ExpiresAt     string `json:"expires_at"`
		BannedBy      string `json:"banned_by"`
		IsPermanent   bool   `json:"is_permanent"`
	} `json:"bans"`
	TotalCount  int   `json:"totalCount"`
	TotalPages  int   `json:"totalPages"`
	CurrentPage int   `json:"currentPage"`
	CachedAt    int64 `json:"cachedAt"`
}

func FetchBanishments(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	page int,
	opts FetchOptions,
) (domain.BanishmentsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchBanishments")
	defer span.End()

	if page <= 0 {
		page = 1
	}

	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	query.Set("page", strconv.Itoa(page))
	sourceURL := fmt.Sprintf("%s/api/bans?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "banishments"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.page", page),
	)

	started := time.Now()
	var payload banishmentsAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("banishments").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("banishments", "error").Inc()
		return domain.BanishmentsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("banishments", "ok").Inc()

	parseStarted := time.Now()
	entries := mapBanishmentEntries(payload)

	result := domain.BanishmentsResult{
		World:      worldName,
		Page:       payload.CurrentPage,
		TotalBans:  payload.TotalCount,
		TotalPages: payload.TotalPages,
		Entries:    entries,
	}
	if result.Page == 0 {
		result.Page = page
	}

	parseDuration.WithLabelValues("banishments").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("banishments").Set(float64(len(result.Entries)))

	return result, sourceURL, nil
}

func FetchAllBanishments(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	opts FetchOptions,
) (domain.BanishmentsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllBanishments")
	defer span.End()

	client := NewClient(opts)
	buildURL := func(page int) string {
		query := url.Values{}
		query.Set("world", strconv.Itoa(worldID))
		query.Set("page", strconv.Itoa(page))
		return fmt.Sprintf("%s/api/bans?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	}

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "banishments"),
		attribute.String("rubinot.world", worldName),
	)

	started := time.Now()
	bodies, sources, err := client.FetchAllPages(
		ctx,
		buildURL(1),
		buildURL,
		func(body string) (int, error) {
			return banishmentsTotalPagesFromBody(body)
		},
	)
	scrapeDuration.WithLabelValues("banishments").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("banishments", "error").Inc()
		return domain.BanishmentsResult{}, sources, err
	}
	scrapeRequests.WithLabelValues("banishments", "ok").Inc()

	parseStarted := time.Now()
	entries := make([]domain.BanishmentEntry, 0)
	totalBans := 0
	for idx, body := range bodies {
		var payload banishmentsAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("banishments", "decode_error").Inc()
			return domain.BanishmentsResult{}, sources, parseErr
		}
		if idx == 0 {
			totalBans = payload.TotalCount
		}
		entries = append(entries, mapBanishmentEntries(payload)...)
	}
	if totalBans <= 0 {
		totalBans = len(entries)
	}

	result := domain.BanishmentsResult{
		World:      worldName,
		Page:       1,
		TotalBans:  totalBans,
		TotalPages: 1,
		Entries:    entries,
	}

	parseDuration.WithLabelValues("banishments").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("banishments").Set(float64(len(result.Entries)))
	return result, sources, nil
}

func mapBanishmentEntries(payload banishmentsAPIResponse) []domain.BanishmentEntry {
	entries := make([]domain.BanishmentEntry, 0, len(payload.Bans))
	for _, row := range payload.Bans {
		permanent := row.IsPermanent || strings.TrimSpace(row.ExpiresAt) == "-1"
		expiresAt := unixTextToRFC3339(row.ExpiresAt)
		duration := "Temporary"
		if permanent {
			duration = "Permanent"
			expiresAt = ""
		}

		entries = append(entries, domain.BanishmentEntry{
			AccountID:   row.AccountID,
			AccountName: strings.TrimSpace(row.AccountName),
			Date:        unixTextToRFC3339(row.BannedAt),
			Character:   strings.TrimSpace(row.MainCharacter),
			Reason:      strings.TrimSpace(row.Reason),
			Duration:    duration,
			IsPermanent: permanent,
			ExpiresAt:   expiresAt,
			BannedBy:    strings.TrimSpace(row.BannedBy),
		})
	}
	return entries
}

func banishmentsTotalPagesFromBody(body string) (int, error) {
	var payload struct {
		TotalPages int `json:"totalPages"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return 0, fmt.Errorf("decode banishments pagination: %w", err)
	}
	if payload.TotalPages <= 0 {
		return 1, nil
	}
	return payload.TotalPages, nil
}

package scraper

import (
	"context"
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

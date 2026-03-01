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

type TransfersFilters struct {
	WorldID   int
	WorldName string
	MinLevel  int
	Page      int
}

type transfersAPIResponse struct {
	Transfers []struct {
		ID            int         `json:"id"`
		PlayerID      int         `json:"player_id"`
		PlayerName    string      `json:"player_name"`
		PlayerLevel   int         `json:"player_level"`
		FromWorldID   int         `json:"from_world_id"`
		ToWorldID     int         `json:"to_world_id"`
		FromWorld     string      `json:"from_world"`
		ToWorld       string      `json:"to_world"`
		TransferredAt interface{} `json:"transferred_at"`
	} `json:"transfers"`
	TotalResults int `json:"totalResults"`
	TotalPages   int `json:"totalPages"`
	CurrentPage  int `json:"currentPage"`
}

func FetchTransfers(
	ctx context.Context,
	baseURL string,
	filters TransfersFilters,
	opts FetchOptions,
) (domain.TransfersResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchTransfers")
	defer span.End()

	page := filters.Page
	if page <= 0 {
		page = 1
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	if filters.WorldID > 0 {
		query.Set("world", strconv.Itoa(filters.WorldID))
	}
	if filters.MinLevel > 0 {
		query.Set("level", strconv.Itoa(filters.MinLevel))
	}

	sourceURL := fmt.Sprintf("%s/api/transfers?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "transfers"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.Int("rubinot.page", page),
	)

	started := time.Now()
	var payload transfersAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("transfers").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("transfers", "error").Inc()
		return domain.TransfersResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("transfers", "ok").Inc()

	parseStarted := time.Now()
	entries := mapTransferEntries(payload)

	result := domain.TransfersResult{
		Filters: domain.TransferFilters{
			World:    filters.WorldName,
			MinLevel: filters.MinLevel,
		},
		Page:           payload.CurrentPage,
		TotalTransfers: payload.TotalResults,
		TotalPages:     payload.TotalPages,
		Entries:        entries,
	}
	if result.Page == 0 {
		result.Page = page
	}

	parseDuration.WithLabelValues("transfers").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("transfers").Set(float64(len(result.Entries)))

	return result, sourceURL, nil
}

func FetchAllTransfers(
	ctx context.Context,
	baseURL string,
	filters TransfersFilters,
	opts FetchOptions,
) (domain.TransfersResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllTransfers")
	defer span.End()

	client := NewClient(opts)
	buildURL := func(page int) string {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		if filters.WorldID > 0 {
			query.Set("world", strconv.Itoa(filters.WorldID))
		}
		if filters.MinLevel > 0 {
			query.Set("level", strconv.Itoa(filters.MinLevel))
		}
		return fmt.Sprintf("%s/api/transfers?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	}

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "transfers"),
		attribute.String("rubinot.source_url", buildURL(1)),
	)

	started := time.Now()
	bodies, sources, err := client.FetchAllPages(
		ctx,
		buildURL(1),
		buildURL,
		func(body string) (int, error) {
			return transfersTotalPagesFromBody(body)
		},
	)
	scrapeDuration.WithLabelValues("transfers").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("transfers", "error").Inc()
		return domain.TransfersResult{}, sources, err
	}
	scrapeRequests.WithLabelValues("transfers", "ok").Inc()

	parseStarted := time.Now()
	entries := make([]domain.TransferEntry, 0)
	totalTransfers := 0
	for idx, body := range bodies {
		var payload transfersAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("transfers", "decode_error").Inc()
			return domain.TransfersResult{}, sources, parseErr
		}
		if idx == 0 {
			totalTransfers = payload.TotalResults
		}
		entries = append(entries, mapTransferEntries(payload)...)
	}
	if totalTransfers <= 0 {
		totalTransfers = len(entries)
	}

	result := domain.TransfersResult{
		Filters: domain.TransferFilters{
			World:    filters.WorldName,
			MinLevel: filters.MinLevel,
		},
		Page:           1,
		TotalTransfers: totalTransfers,
		TotalPages:     1,
		Entries:        entries,
	}

	parseDuration.WithLabelValues("transfers").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("transfers").Set(float64(len(result.Entries)))
	return result, sources, nil
}

func mapTransferEntries(payload transfersAPIResponse) []domain.TransferEntry {
	entries := make([]domain.TransferEntry, 0, len(payload.Transfers))
	for _, row := range payload.Transfers {
		formerWorld := strings.TrimSpace(row.FromWorld)
		if formerWorld == "" {
			formerWorld = worldNameByID(row.FromWorldID)
		}
		destWorld := strings.TrimSpace(row.ToWorld)
		if destWorld == "" {
			destWorld = worldNameByID(row.ToWorldID)
		}
		entries = append(entries, domain.TransferEntry{
			ID:               row.ID,
			PlayerID:         row.PlayerID,
			PlayerName:       strings.TrimSpace(row.PlayerName),
			Level:            row.PlayerLevel,
			FormerWorld:      formerWorld,
			FormerWorldID:    row.FromWorldID,
			DestinationWorld: destWorld,
			DestWorldID:      row.ToWorldID,
			TransferDate:     unixAnyToRFC3339(row.TransferredAt),
		})
	}
	return entries
}

func transfersTotalPagesFromBody(body string) (int, error) {
	var payload struct {
		TotalPages int `json:"totalPages"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return 0, fmt.Errorf("decode transfers pagination: %w", err)
	}
	if payload.TotalPages <= 0 {
		return 1, nil
	}
	return payload.TotalPages, nil
}

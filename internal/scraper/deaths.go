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

type DeathsFilters struct {
	Guild    string
	MinLevel int
	PvPOnly  *bool
	Page     int
}

type deathsAPIResponse struct {
	Deaths []struct {
		PlayerID           int    `json:"player_id"`
		Time               string `json:"time"`
		Level              int    `json:"level"`
		KilledBy           string `json:"killed_by"`
		IsPlayer           int    `json:"is_player"`
		MostDamageBy       string `json:"mostdamage_by"`
		MostDamageIsPlayer int    `json:"mostdamage_is_player"`
		Victim             string `json:"victim"`
		WorldID            int    `json:"world_id"`
	} `json:"deaths"`
	Pagination struct {
		CurrentPage  int `json:"currentPage"`
		TotalPages   int `json:"totalPages"`
		TotalCount   int `json:"totalCount"`
		ItemsPerPage int `json:"itemsPerPage"`
	} `json:"pagination"`
}

func FetchDeaths(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	filters DeathsFilters,
	opts FetchOptions,
) (domain.DeathsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchDeaths")
	defer span.End()

	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	if filters.Page > 0 {
		query.Set("page", strconv.Itoa(filters.Page))
	}
	if filters.MinLevel > 0 {
		query.Set("level", strconv.Itoa(filters.MinLevel))
	}
	if filters.PvPOnly != nil {
		query.Set("pvp", strconv.FormatBool(*filters.PvPOnly))
	}
	if strings.TrimSpace(filters.Guild) != "" {
		query.Set("guild", strings.TrimSpace(filters.Guild))
	}

	sourceURL := fmt.Sprintf("%s/api/deaths?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "deaths"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
	)

	started := time.Now()
	var payload deathsAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("deaths").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("deaths", "error").Inc()
		return domain.DeathsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("deaths", "ok").Inc()

	parseStarted := time.Now()
	result := mapDeathsResponse(worldName, filters, payload)
	parseDuration.WithLabelValues("deaths").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("deaths").Set(float64(len(result.Entries)))

	return result, sourceURL, nil
}

func FetchAllDeaths(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	filters DeathsFilters,
	opts FetchOptions,
) (domain.DeathsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllDeaths")
	defer span.End()

	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "deaths"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
	)

	buildURL := func(page int) string {
		query := url.Values{}
		query.Set("world", strconv.Itoa(worldID))
		query.Set("page", strconv.Itoa(page))
		if filters.MinLevel > 0 {
			query.Set("level", strconv.Itoa(filters.MinLevel))
		}
		if filters.PvPOnly != nil {
			query.Set("pvp", strconv.FormatBool(*filters.PvPOnly))
		}
		if strings.TrimSpace(filters.Guild) != "" {
			query.Set("guild", strings.TrimSpace(filters.Guild))
		}
		return fmt.Sprintf("%s/api/deaths?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	}

	started := time.Now()
	bodies, sources, err := client.FetchAllPages(
		ctx,
		buildURL(1),
		buildURL,
		func(body string) (int, error) {
			return deathsTotalPagesFromBody(body)
		},
	)
	scrapeDuration.WithLabelValues("deaths").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("deaths", "error").Inc()
		return domain.DeathsResult{}, sources, err
	}
	scrapeRequests.WithLabelValues("deaths", "ok").Inc()

	parseStarted := time.Now()
	entries := make([]domain.DeathEntry, 0)
	totalDeaths := 0
	itemsPerPage := 0
	for idx, body := range bodies {
		var payload deathsAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("deaths", "decode_error").Inc()
			return domain.DeathsResult{}, sources, parseErr
		}
		if idx == 0 {
			totalDeaths = payload.Pagination.TotalCount
			itemsPerPage = payload.Pagination.ItemsPerPage
		}

		pageFilters := filters
		pageFilters.Page = idx + 1
		pageResult := mapDeathsResponse(worldName, pageFilters, payload)
		entries = append(entries, pageResult.Entries...)
	}
	if totalDeaths <= 0 {
		totalDeaths = len(entries)
	}

	result := domain.DeathsResult{
		World: worldName,
		Filters: domain.DeathFilters{
			Guild:    filters.Guild,
			MinLevel: filters.MinLevel,
			PvPOnly:  filters.PvPOnly,
			Page:     1,
		},
		Entries:     entries,
		TotalDeaths: totalDeaths,
		Pagination: domain.DeathPagination{
			CurrentPage:  1,
			TotalPages:   1,
			TotalCount:   totalDeaths,
			ItemsPerPage: itemsPerPage,
		},
	}

	parseDuration.WithLabelValues("deaths").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("deaths").Set(float64(len(result.Entries)))
	return result, sources, nil
}

func deathsTotalPagesFromBody(body string) (int, error) {
	var payload struct {
		Pagination struct {
			TotalPages int `json:"totalPages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return 0, fmt.Errorf("decode deaths pagination: %w", err)
	}
	if payload.Pagination.TotalPages <= 0 {
		return 1, nil
	}
	return payload.Pagination.TotalPages, nil
}

func mapDeathsResponse(worldName string, filters DeathsFilters, payload deathsAPIResponse) domain.DeathsResult {
	entries := make([]domain.DeathEntry, 0, len(payload.Deaths))
	for _, row := range payload.Deaths {
		killedBy := strings.TrimSpace(row.KilledBy)
		mostDamageBy := strings.TrimSpace(row.MostDamageBy)

		killers := make([]string, 0, 2)
		if killedBy != "" {
			killers = append(killers, killedBy)
		}
		if mostDamageBy != "" && !strings.EqualFold(mostDamageBy, killedBy) {
			killers = append(killers, mostDamageBy)
		}

		entry := domain.DeathEntry{
			PlayerID: row.PlayerID,
			Date:     unixTextToRFC3339(row.Time),
			Victim: domain.DeathVictim{
				Name:  strings.TrimSpace(row.Victim),
				Level: row.Level,
			},
			KilledBy:           killedBy,
			IsPlayerKill:       row.IsPlayer == 1,
			MostDamageBy:       mostDamageBy,
			MostDamageIsPlayer: row.MostDamageIsPlayer == 1,
			WorldID:            row.WorldID,
			Killers:            killers,
			IsPvP:              row.IsPlayer == 1 || row.MostDamageIsPlayer == 1,
		}
		entries = append(entries, entry)
	}

	page := payload.Pagination.CurrentPage
	if page == 0 {
		page = filters.Page
		if page == 0 {
			page = 1
		}
	}

	return domain.DeathsResult{
		World: worldName,
		Filters: domain.DeathFilters{
			Guild:    filters.Guild,
			MinLevel: filters.MinLevel,
			PvPOnly:  filters.PvPOnly,
			Page:     page,
		},
		Entries:     entries,
		TotalDeaths: payload.Pagination.TotalCount,
		Pagination: domain.DeathPagination{
			CurrentPage:  page,
			TotalPages:   payload.Pagination.TotalPages,
			TotalCount:   payload.Pagination.TotalCount,
			ItemsPerPage: payload.Pagination.ItemsPerPage,
		},
	}
}

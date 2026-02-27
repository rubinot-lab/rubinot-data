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

type guildsAPIResponse struct {
	Guilds []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		WorldID     int    `json:"world_id"`
		LogoName    string `json:"logo_name"`
	} `json:"guilds"`
	TotalCount  int `json:"totalCount"`
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
}

func FetchGuilds(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	page int,
	opts FetchOptions,
) (domain.GuildsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchGuilds")
	defer span.End()

	if page <= 0 {
		page = 1
	}

	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	query.Set("page", strconv.Itoa(page))
	sourceURL := fmt.Sprintf("%s/api/guilds?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guilds"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.page", page),
	)

	started := time.Now()
	var payload guildsAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("guilds").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("guilds", "error").Inc()
		return domain.GuildsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("guilds", "ok").Inc()

	parseStarted := time.Now()
	items := mapGuildListEntries(payload)

	result := domain.GuildsResult{
		World:  worldName,
		Guilds: items,
		Active: items,
		Pagination: &domain.GuildsPagination{
			CurrentPage: payload.CurrentPage,
			TotalPages:  payload.TotalPages,
			TotalCount:  payload.TotalCount,
		},
	}
	if result.Pagination.CurrentPage == 0 {
		result.Pagination.CurrentPage = page
	}
	parseDuration.WithLabelValues("guilds").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("guilds").Set(float64(len(items)))

	return result, sourceURL, nil
}

func FetchAllGuilds(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	opts FetchOptions,
) (domain.GuildsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllGuilds")
	defer span.End()

	client := NewClient(opts)
	buildURL := func(page int) string {
		query := url.Values{}
		query.Set("world", strconv.Itoa(worldID))
		query.Set("page", strconv.Itoa(page))
		return fmt.Sprintf("%s/api/guilds?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	}

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guilds"),
		attribute.String("rubinot.world", worldName),
	)

	started := time.Now()
	bodies, sources, err := client.FetchAllPages(
		ctx,
		buildURL(1),
		buildURL,
		func(body string) (int, error) {
			return guildsTotalPagesFromBody(body)
		},
	)
	scrapeDuration.WithLabelValues("guilds").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("guilds", "error").Inc()
		return domain.GuildsResult{}, sources, err
	}
	scrapeRequests.WithLabelValues("guilds", "ok").Inc()

	parseStarted := time.Now()
	items := make([]domain.GuildListEntry, 0)
	totalCount := 0
	for idx, body := range bodies {
		var payload guildsAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("guilds", "decode_error").Inc()
			return domain.GuildsResult{}, sources, parseErr
		}
		if idx == 0 {
			totalCount = payload.TotalCount
		}
		items = append(items, mapGuildListEntries(payload)...)
	}
	if totalCount <= 0 {
		totalCount = len(items)
	}

	result := domain.GuildsResult{
		World:  worldName,
		Guilds: items,
		Active: items,
		Pagination: &domain.GuildsPagination{
			CurrentPage: 1,
			TotalPages:  1,
			TotalCount:  totalCount,
		},
	}

	parseDuration.WithLabelValues("guilds").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("guilds").Set(float64(len(items)))
	return result, sources, nil
}

func FetchAllGuildsDetails(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	opts FetchOptions,
) (domain.GuildsDetailsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllGuildsDetails")
	defer span.End()

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guilds"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
	)

	guildsList, listSources, err := FetchAllGuilds(ctx, baseURL, worldName, worldID, opts)
	if err != nil {
		return domain.GuildsDetailsResult{}, listSources, err
	}

	detailURLs := make([]string, 0, len(guildsList.Guilds))
	for _, guild := range guildsList.Guilds {
		guildName := strings.TrimSpace(guild.Name)
		if guildName == "" {
			continue
		}
		detailURLs = append(detailURLs, fmt.Sprintf("%s/api/guilds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(guildName)))
	}

	if len(detailURLs) == 0 {
		return domain.GuildsDetailsResult{
			World:  worldName,
			Guilds: []domain.GuildResult{},
		}, listSources, nil
	}

	client := NewClient(opts)
	started := time.Now()
	bodies, err := fetchBatchJSONBodies(ctx, client, detailURLs)
	scrapeDuration.WithLabelValues("guilds").Observe(time.Since(started).Seconds())
	allSources := append(append([]string{}, listSources...), detailURLs...)
	if err != nil {
		scrapeRequests.WithLabelValues("guilds", "error").Inc()
		return domain.GuildsDetailsResult{}, allSources, err
	}
	scrapeRequests.WithLabelValues("guilds", "ok").Inc()

	parseStarted := time.Now()
	guilds := make([]domain.GuildResult, 0, len(bodies))
	for _, body := range bodies {
		var payload guildAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("guilds", "decode_error").Inc()
			return domain.GuildsDetailsResult{}, allSources, parseErr
		}
		guilds = append(guilds, mapGuildResponse(payload))
	}

	result := domain.GuildsDetailsResult{
		World:  worldName,
		Guilds: guilds,
	}
	parseDuration.WithLabelValues("guilds").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("guilds").Set(float64(len(guilds)))
	return result, allSources, nil
}

func mapGuildListEntries(payload guildsAPIResponse) []domain.GuildListEntry {
	items := make([]domain.GuildListEntry, 0, len(payload.Guilds))
	for _, guild := range payload.Guilds {
		logoName := strings.TrimSpace(guild.LogoName)
		logoURL := ""
		if logoName != "" {
			logoURL = fmt.Sprintf("https://static.rubinot.com/guilds/%s", url.PathEscape(logoName))
		}

		items = append(items, domain.GuildListEntry{
			ID:          guild.ID,
			Name:        strings.TrimSpace(guild.Name),
			Description: strings.TrimSpace(guild.Description),
			WorldID:     guild.WorldID,
			LogoName:    logoName,
			LogoURL:     logoURL,
		})
	}
	return items
}

func guildsTotalPagesFromBody(body string) (int, error) {
	var payload struct {
		TotalPages int `json:"totalPages"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return 0, fmt.Errorf("decode guilds pagination: %w", err)
	}
	if payload.TotalPages <= 0 {
		return 1, nil
	}
	return payload.TotalPages, nil
}

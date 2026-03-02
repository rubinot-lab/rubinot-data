package scraper

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

const highscoresPageSize = 50

type highscoresAPIResponse struct {
	Players []struct {
		Rank      int         `json:"rank"`
		ID        int         `json:"id"`
		Name      string      `json:"name"`
		Level     int         `json:"level"`
		Vocation  int         `json:"vocation"`
		WorldID   int         `json:"world_id"`
		WorldName string      `json:"worldName"`
		Value     interface{} `json:"value"`
	} `json:"players"`
	TotalCount       int   `json:"totalCount"`
	CachedAt         int64 `json:"cachedAt"`
	AvailableSeasons []int `json:"availableSeasons"`
}

func FetchHighscores(
	ctx context.Context,
	baseURL,
	world string,
	worldID int,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
	page int,
	opts FetchOptions,
) (domain.HighscoresResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchHighscores")
	defer span.End()

	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "highscores"),
		attribute.String("rubinot.world", world),
		attribute.String("rubinot.category", category.Slug),
		attribute.Int("rubinot.profession_id", vocation.ProfessionID),
		attribute.Int("rubinot.page", page),
	)

	payload, sourceURL, err := fetchHighscoresPayload(ctx, client, baseURL, fmt.Sprintf("%d", worldID), category, vocation)
	if err != nil {
		scrapeRequests.WithLabelValues("highscores", "error").Inc()
		return domain.HighscoresResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("highscores", "ok").Inc()

	parseStarted := time.Now()
	result, mapErr := mapHighscoresResponse(world, category, vocation, page, payload)
	parseDuration.WithLabelValues("highscores").Observe(time.Since(parseStarted).Seconds())
	if mapErr != nil {
		ParseErrors.WithLabelValues("highscores", "page_out_of_bounds").Inc()
		return domain.HighscoresResult{}, sourceURL, mapErr
	}
	ParseItems.WithLabelValues("highscores").Set(float64(len(result.HighscoreList)))

	return result, sourceURL, nil
}

func FetchAllHighscores(
	ctx context.Context,
	baseURL,
	world string,
	worldID int,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
	opts FetchOptions,
) (domain.HighscoresResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllHighscores")
	defer span.End()

	client := NewClient(opts)
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "highscores"),
		attribute.String("rubinot.world", world),
		attribute.String("rubinot.category", category.Slug),
		attribute.Int("rubinot.profession_id", vocation.ProfessionID),
	)

	payload, sourceURL, err := fetchHighscoresPayload(ctx, client, baseURL, fmt.Sprintf("%d", worldID), category, vocation)
	if err != nil {
		scrapeRequests.WithLabelValues("highscores", "error").Inc()
		return domain.HighscoresResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("highscores", "ok").Inc()

	parseStarted := time.Now()
	items := make([]domain.Highscore, 0, len(payload.Players))
	for _, row := range payload.Players {
		items = append(items, domain.Highscore{
			Rank:       row.Rank,
			ID:         row.ID,
			Name:       strings.TrimSpace(row.Name),
			Vocation:   fallbackString(vocationNameByID(row.Vocation), "Unknown"),
			VocationID: row.Vocation,
			World:      resolveHighscoreWorldName(row.WorldName, row.WorldID, world),
			WorldID:    row.WorldID,
			Level:      row.Level,
			Value:      fmt.Sprintf("%v", row.Value),
		})
	}

	totalRecords := payload.TotalCount
	if totalRecords <= 0 {
		totalRecords = len(items)
	}
	if totalRecords < 0 {
		totalRecords = 0
	}

	result := domain.HighscoresResult{
		World:         world,
		Category:      category.Slug,
		Vocation:      vocation.Name,
		CachedAt:      payload.CachedAt,
		HighscoreList: items,
		HighscorePage: domain.HighscorePage{
			CurrentPage:  1,
			TotalPages:   1,
			TotalRecords: totalRecords,
		},
		AvailableSeasons: payload.AvailableSeasons,
	}

	parseDuration.WithLabelValues("highscores").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("highscores").Set(float64(len(result.HighscoreList)))
	return result, sourceURL, nil
}

func FetchHighscoresAllWorlds(
	ctx context.Context,
	baseURL string,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
	page int,
	opts FetchOptions,
) (domain.HighscoresResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchHighscoresAllWorlds")
	defer span.End()

	client := NewClient(opts)
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "highscores"),
		attribute.String("rubinot.world", "all"),
		attribute.String("rubinot.category", category.Slug),
		attribute.Int("rubinot.profession_id", vocation.ProfessionID),
		attribute.Int("rubinot.page", page),
	)

	payload, sourceURL, err := fetchHighscoresPayload(ctx, client, baseURL, "all", category, vocation)
	if err != nil {
		scrapeRequests.WithLabelValues("highscores", "error").Inc()
		return domain.HighscoresResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("highscores", "ok").Inc()

	parseStarted := time.Now()
	result, mapErr := mapHighscoresResponse("all", category, vocation, page, payload)
	parseDuration.WithLabelValues("highscores").Observe(time.Since(parseStarted).Seconds())
	if mapErr != nil {
		ParseErrors.WithLabelValues("highscores", "page_out_of_bounds").Inc()
		return domain.HighscoresResult{}, sourceURL, mapErr
	}
	ParseItems.WithLabelValues("highscores").Set(float64(len(result.HighscoreList)))
	return result, sourceURL, nil
}

func FetchAllHighscoresAllWorlds(
	ctx context.Context,
	baseURL string,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
	opts FetchOptions,
) (domain.HighscoresResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllHighscoresAllWorlds")
	defer span.End()

	client := NewClient(opts)
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "highscores"),
		attribute.String("rubinot.world", "all"),
		attribute.String("rubinot.category", category.Slug),
		attribute.Int("rubinot.profession_id", vocation.ProfessionID),
	)

	payload, sourceURL, err := fetchHighscoresPayload(ctx, client, baseURL, "all", category, vocation)
	if err != nil {
		scrapeRequests.WithLabelValues("highscores", "error").Inc()
		return domain.HighscoresResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("highscores", "ok").Inc()

	parseStarted := time.Now()
	items := make([]domain.Highscore, 0, len(payload.Players))
	for _, row := range payload.Players {
		items = append(items, domain.Highscore{
			Rank:       row.Rank,
			ID:         row.ID,
			Name:       strings.TrimSpace(row.Name),
			Vocation:   fallbackString(vocationNameByID(row.Vocation), "Unknown"),
			VocationID: row.Vocation,
			World:      resolveHighscoreWorldName(row.WorldName, row.WorldID, "all"),
			WorldID:    row.WorldID,
			Level:      row.Level,
			Value:      fmt.Sprintf("%v", row.Value),
		})
	}

	totalRecords := payload.TotalCount
	if totalRecords <= 0 {
		totalRecords = len(items)
	}
	if totalRecords < 0 {
		totalRecords = 0
	}

	result := domain.HighscoresResult{
		World:         "all",
		Category:      category.Slug,
		Vocation:      vocation.Name,
		CachedAt:      payload.CachedAt,
		HighscoreList: items,
		HighscorePage: domain.HighscorePage{
			CurrentPage:  1,
			TotalPages:   1,
			TotalRecords: totalRecords,
		},
		AvailableSeasons: payload.AvailableSeasons,
	}
	parseDuration.WithLabelValues("highscores").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("highscores").Set(float64(len(result.HighscoreList)))
	return result, sourceURL, nil
}

func FetchAllHighscoresPerWorld(
	ctx context.Context,
	baseURL string,
	worlds []validation.World,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
	opts FetchOptions,
) (domain.HighscoresByWorldResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllHighscoresPerWorld")
	defer span.End()

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "highscores"),
		attribute.String("rubinot.world", "all"),
		attribute.String("rubinot.category", category.Slug),
		attribute.Int("rubinot.profession_id", vocation.ProfessionID),
		attribute.Int("rubinot.world_count", len(worlds)),
	)

	if len(worlds) == 0 {
		return domain.HighscoresByWorldResult{
			World:        "all",
			Category:     category.Slug,
			Vocation:     vocation.Name,
			TotalWorlds:  0,
			TotalRecords: 0,
			TotalEntries: 0,
			Worlds:       []domain.HighscoresResult{},
		}, []string{}, nil
	}

	results := make([]domain.HighscoresResult, 0, len(worlds))
	sources := make([]string, 0, len(worlds))
	totalRecords := 0
	totalEntries := 0

	for _, world := range worlds {
		highscores, sourceURL, err := FetchAllHighscores(
			ctx,
			baseURL,
			world.Name,
			world.ID,
			category,
			vocation,
			opts,
		)
		sources = append(sources, sourceURL)
		if err != nil {
			return domain.HighscoresByWorldResult{}, sources, err
		}
		results = append(results, highscores)
		totalRecords += highscores.HighscorePage.TotalRecords
		totalEntries += len(highscores.HighscoreList)
	}

	return domain.HighscoresByWorldResult{
		World:        "all",
		Category:     category.Slug,
		Vocation:     vocation.Name,
		TotalWorlds:  len(results),
		TotalRecords: totalRecords,
		TotalEntries: totalEntries,
		Worlds:       results,
	}, sources, nil
}

func fetchHighscoresPayload(
	ctx context.Context,
	client *Client,
	baseURL string,
	worldRef string,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
) (highscoresAPIResponse, string, error) {
	query := url.Values{}
	if strings.TrimSpace(worldRef) != "" {
		query.Set("world", strings.TrimSpace(worldRef))
	}
	query.Set("category", category.Slug)
	query.Set("vocation", fmt.Sprintf("%d", vocation.ProfessionID))
	sourceURL := fmt.Sprintf("%s/api/highscores?%s", strings.TrimRight(baseURL, "/"), query.Encode())

	started := time.Now()
	var payload highscoresAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("highscores").Observe(time.Since(started).Seconds())
	if err != nil {
		return highscoresAPIResponse{}, sourceURL, err
	}

	return payload, sourceURL, nil
}

func mapHighscoresResponse(
	world string,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
	page int,
	payload highscoresAPIResponse,
) (domain.HighscoresResult, error) {
	totalRecords := payload.TotalCount
	if totalRecords <= 0 {
		totalRecords = len(payload.Players)
	}
	if totalRecords < 0 {
		totalRecords = 0
	}

	totalPages := 0
	if totalRecords > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(highscoresPageSize)))
	}
	if totalPages == 0 {
		totalPages = 1
	}

	if page > totalPages {
		return domain.HighscoresResult{}, validation.NewError(validation.ErrorPageOutOfBounds, "page out of bounds", nil)
	}

	start := (page - 1) * highscoresPageSize
	if start < 0 {
		start = 0
	}
	if start > len(payload.Players) {
		start = len(payload.Players)
	}
	end := start + highscoresPageSize
	if end > len(payload.Players) {
		end = len(payload.Players)
	}
	pageRows := payload.Players[start:end]

	items := make([]domain.Highscore, 0, len(pageRows))
	for _, row := range pageRows {
		items = append(items, domain.Highscore{
			Rank:       row.Rank,
			ID:         row.ID,
			Name:       strings.TrimSpace(row.Name),
			Vocation:   fallbackString(vocationNameByID(row.Vocation), "Unknown"),
			VocationID: row.Vocation,
			World:      resolveHighscoreWorldName(row.WorldName, row.WorldID, world),
			WorldID:    row.WorldID,
			Level:      row.Level,
			Value:      fmt.Sprintf("%v", row.Value),
		})
	}

	return domain.HighscoresResult{
		World:         world,
		Category:      category.Slug,
		Vocation:      vocation.Name,
		CachedAt:      payload.CachedAt,
		HighscoreList: items,
		HighscorePage: domain.HighscorePage{
			CurrentPage:  page,
			TotalPages:   totalPages,
			TotalRecords: totalRecords,
		},
		AvailableSeasons: payload.AvailableSeasons,
	}, nil
}

func resolveHighscoreWorldName(upstreamName string, worldID int, fallback string) string {
	if trimmed := strings.TrimSpace(upstreamName); trimmed != "" {
		return trimmed
	}
	if resolved := worldNameByID(worldID); resolved != "" {
		return resolved
	}
	return fallback
}

func fallbackString(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

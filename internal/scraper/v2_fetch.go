package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
)

func V2FetchWorlds(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.WorldsResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/worlds", strings.TrimRight(baseURL, "/"))
	var payload worldsAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.WorldsResult{}, sourceURL, err
	}
	return mapWorldsResponse(payload), sourceURL, nil
}

func V2FetchWorld(ctx context.Context, oc *OptimizedClient, baseURL, world string) (domain.WorldResult, string, error) {
	canonicalWorld := strings.TrimSpace(world)
	sourceURL := fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(canonicalWorld))
	var payload worldDetailAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.WorldResult{}, sourceURL, err
	}
	return mapWorldResponse(canonicalWorld, payload), sourceURL, nil
}

func V2FetchCharacter(ctx context.Context, oc *OptimizedClient, baseURL, characterName string) (domain.CharacterResult, string, error) {
	query := url.Values{}
	query.Set("name", strings.TrimSpace(characterName))
	sourceURL := fmt.Sprintf("%s/api/characters/search?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	var payload characterAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.CharacterResult{}, sourceURL, err
	}
	if payload.Player == nil {
		return domain.CharacterResult{}, sourceURL, validation.NewError(validation.ErrorEntityNotFound, "character not found", nil)
	}
	return mapCharacterResponse(payload), sourceURL, nil
}

func V2FetchGuild(ctx context.Context, oc *OptimizedClient, baseURL, guildName string) (domain.GuildResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/guilds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(strings.TrimSpace(guildName)))
	var payload guildAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.GuildResult{}, sourceURL, err
	}
	return mapGuildResponse(payload), sourceURL, nil
}

func V2FetchGuilds(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID, page int) (domain.GuildsResult, string, error) {
	if page <= 0 {
		page = 1
	}
	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	query.Set("page", strconv.Itoa(page))
	sourceURL := fmt.Sprintf("%s/api/guilds?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	var payload guildsAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.GuildsResult{}, sourceURL, err
	}
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
	return result, sourceURL, nil
}

func V2FetchDeaths(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int, filters DeathsFilters) (domain.DeathsResult, string, error) {
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
	var payload deathsAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.DeathsResult{}, sourceURL, err
	}
	return mapDeathsResponse(worldName, filters, payload), sourceURL, nil
}

func V2FetchBanishments(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID, page int) (domain.BanishmentsResult, string, error) {
	if page <= 0 {
		page = 1
	}
	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	query.Set("page", strconv.Itoa(page))
	sourceURL := fmt.Sprintf("%s/api/bans?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	var payload banishmentsAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.BanishmentsResult{}, sourceURL, err
	}
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
	return result, sourceURL, nil
}

func V2FetchTransfers(ctx context.Context, oc *OptimizedClient, baseURL string, filters TransfersFilters) (domain.TransfersResult, string, error) {
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
	var payload transfersAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.TransfersResult{}, sourceURL, err
	}
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
	return result, sourceURL, nil
}

func V2FetchHighscores(
	ctx context.Context,
	oc *OptimizedClient,
	baseURL string,
	worldName string,
	worldID int,
	category validation.HighscoreCategory,
	vocation validation.HighscoreVocation,
) (domain.HighscoresResult, string, error) {
	query := url.Values{}
	if worldID > 0 {
		query.Set("world", strconv.Itoa(worldID))
	}
	query.Set("category", category.Slug)
	query.Set("vocation", fmt.Sprintf("%d", vocation.ProfessionID))
	sourceURL := fmt.Sprintf("%s/api/highscores?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	var payload highscoresAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.HighscoresResult{}, sourceURL, err
	}

	world := strings.TrimSpace(worldName)
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

	return domain.HighscoresResult{
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
	}, sourceURL, nil
}

func V2FetchKillstatistics(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int) (domain.KillstatisticsResult, string, error) {
	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	sourceURL := fmt.Sprintf("%s/api/killstats?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	var payload killstatisticsAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.KillstatisticsResult{}, sourceURL, err
	}
	return mapKillstatisticsResponse(worldName, payload), sourceURL, nil
}

func V2FetchBoosted(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.BoostedResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/boosted", strings.TrimRight(baseURL, "/"))
	var payload boostedAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.BoostedResult{}, sourceURL, err
	}
	return domain.BoostedResult{
		Boss: domain.BoostedEntity{
			ID:       payload.Boss.ID,
			Name:     strings.TrimSpace(payload.Boss.Name),
			LookType: payload.Boss.LookType,
		},
		Monster: domain.BoostedEntity{
			ID:       payload.Monster.ID,
			Name:     strings.TrimSpace(payload.Monster.Name),
			LookType: payload.Monster.LookType,
		},
	}, sourceURL, nil
}

func V2FetchMaintenance(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.MaintenanceResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/maintenance", strings.TrimRight(baseURL, "/"))
	var payload maintenanceAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.MaintenanceResult{}, sourceURL, err
	}
	return domain.MaintenanceResult{
		IsClosed:     payload.IsClosed,
		CloseMessage: strings.TrimSpace(payload.CloseMessage),
	}, sourceURL, nil
}

func V2FetchCurrentAuctions(ctx context.Context, oc *OptimizedClient, baseURL string, page int) (domain.AuctionsResult, string, error) {
	return v2FetchAuctionList(ctx, oc, baseURL, "current", page)
}

func V2FetchAuctionHistory(ctx context.Context, oc *OptimizedClient, baseURL string, page int) (domain.AuctionsResult, string, error) {
	return v2FetchAuctionList(ctx, oc, baseURL, "history", page)
}

func v2FetchAuctionList(ctx context.Context, oc *OptimizedClient, baseURL, auctionType string, page int) (domain.AuctionsResult, string, error) {
	if page <= 0 {
		page = 1
	}
	sourceURL := buildAuctionListURL(baseURL, auctionType, page)
	var payload auctionListAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.AuctionsResult{}, sourceURL, err
	}
	entries := make([]domain.AuctionEntry, 0, len(payload.Auctions))
	for _, row := range payload.Auctions {
		entries = append(entries, mapAuctionListEntry(row))
	}
	result := domain.AuctionsResult{
		Type:         auctionType,
		Page:         payload.Pagination.Page,
		TotalResults: payload.Pagination.Total,
		TotalPages:   payload.Pagination.TotalPages,
		Entries:      entries,
		Pagination: &domain.AuctionsPagination{
			Page:       payload.Pagination.Page,
			Limit:      payload.Pagination.Limit,
			Total:      payload.Pagination.Total,
			TotalPages: payload.Pagination.TotalPages,
		},
	}
	if result.Page == 0 {
		result.Page = page
	}
	return result, sourceURL, nil
}

func V2FetchAuctionDetail(ctx context.Context, oc *OptimizedClient, baseURL string, auctionID int) (domain.AuctionDetail, []string, error) {
	sourceURL := fmt.Sprintf("%s/api/bazaar/%d", strings.TrimRight(baseURL, "/"), auctionID)
	var payload auctionDetailAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return domain.AuctionDetail{}, []string{sourceURL}, err
	}
	return mapAuctionDetailResponse(payload), []string{sourceURL}, nil
}

func V2FetchNewsByID(ctx context.Context, oc *OptimizedClient, baseURL string, newsID int) (domain.NewsResult, []string, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	payload, err := v2FetchNewsPayload(ctx, oc, sourceURL)
	if err != nil {
		return domain.NewsResult{}, []string{sourceURL}, err
	}
	article, ok := findNewsArticleByID(payload, newsID)
	if ok {
		return article, []string{sourceURL}, nil
	}
	ticker, ok := findNewsTickerByID(payload, newsID)
	if ok {
		return ticker, []string{sourceURL}, nil
	}
	return domain.NewsResult{}, []string{sourceURL}, validation.NewError(validation.ErrorEntityNotFound, "news entry not found", nil)
}

func V2FetchNewsArchive(ctx context.Context, oc *OptimizedClient, baseURL string, archiveDays int) (domain.NewsListResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	payload, err := v2FetchNewsPayload(ctx, oc, sourceURL)
	if err != nil {
		return domain.NewsListResult{}, sourceURL, err
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -archiveDays)
	entries := buildNewsArchiveEntries(payload, cutoff)
	return domain.NewsListResult{
		Mode:        "archive",
		ArchiveDays: archiveDays,
		Entries:     entries,
	}, sourceURL, nil
}

func V2FetchNewsLatest(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.NewsListResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	payload, err := v2FetchNewsPayload(ctx, oc, sourceURL)
	if err != nil {
		return domain.NewsListResult{}, sourceURL, err
	}
	entries := make([]newsListEntryWithTime, 0, len(payload.Articles))
	for _, article := range payload.Articles {
		at := parseNewsTimestamp(article.PublishedAt)
		entries = append(entries, newsListEntryWithTime{
			entry: domain.NewsListEntry{
				ID:          article.ID,
				Date:        article.PublishedAt,
				Title:       strings.TrimSpace(article.Title),
				Category:    strings.TrimSpace(article.Category.Name),
				Type:        "article",
				URL:         fmt.Sprintf("/news/%s", strings.TrimSpace(article.Slug)),
				Author:      strings.TrimSpace(article.Author),
				Slug:        strings.TrimSpace(article.Slug),
				Summary:     strings.TrimSpace(article.Summary),
				CategoryRef: toNewsCategory(article.Category.ID, article.Category.Name, article.Category.Slug, article.Category.Color, article.Category.Icon, article.Category.IconURL),
			},
			time: at,
		})
	}
	sortNewsEntries(entries)
	resultEntries := make([]domain.NewsListEntry, 0, len(entries))
	for _, entry := range entries {
		resultEntries = append(resultEntries, entry.entry)
	}
	return domain.NewsListResult{
		Mode:    "latest",
		Entries: resultEntries,
	}, sourceURL, nil
}

func V2FetchNewsTicker(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.NewsListResult, string, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(baseURL, "/"))
	payload, err := v2FetchNewsPayload(ctx, oc, sourceURL)
	if err != nil {
		return domain.NewsListResult{}, sourceURL, err
	}
	entries := make([]newsListEntryWithTime, 0, len(payload.Tickers))
	for _, ticker := range payload.Tickers {
		at := parseNewsTimestamp(ticker.CreatedAt)
		entries = append(entries, newsListEntryWithTime{
			entry: domain.NewsListEntry{
				ID:          ticker.ID,
				Date:        ticker.CreatedAt,
				Category:    strings.TrimSpace(ticker.Category.Name),
				Type:        "ticker",
				Message:     ticker.Message,
				Author:      strings.TrimSpace(ticker.Author),
				CategoryRef: toNewsCategory(ticker.Category.ID, ticker.Category.Name, ticker.Category.Slug, ticker.Category.Color, ticker.Category.Icon, ticker.Category.IconURL),
			},
			time: at,
		})
	}
	sortNewsEntries(entries)
	resultEntries := make([]domain.NewsListEntry, 0, len(entries))
	for _, entry := range entries {
		resultEntries = append(resultEntries, entry.entry)
	}
	return domain.NewsListResult{
		Mode:    "newsticker",
		Entries: resultEntries,
	}, sourceURL, nil
}

func v2FetchNewsPayload(ctx context.Context, oc *OptimizedClient, sourceURL string) (newsAPIResponse, error) {
	var payload newsAPIResponse
	if err := oc.FetchJSON(ctx, sourceURL, &payload); err != nil {
		return newsAPIResponse{}, err
	}
	return payload, nil
}

func V2FetchWorldBatch(ctx context.Context, oc *OptimizedClient, baseURL string, worlds []validation.World) ([]domain.WorldResult, []string, error) {
	apiURLs := make([]string, 0, len(worlds))
	for _, world := range worlds {
		apiURLs = append(apiURLs, fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(strings.TrimSpace(world.Name))))
	}
	bodies, err := oc.BatchFetchJSON(ctx, apiURLs)
	if err != nil {
		return nil, apiURLs, err
	}
	results := make([]domain.WorldResult, 0, len(worlds))
	for _, world := range worlds {
		apiURL := fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(strings.TrimSpace(world.Name)))
		body, ok := bodies[apiURL]
		if !ok {
			return nil, apiURLs, fmt.Errorf("missing response for %s", apiURL)
		}
		var payload worldDetailAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			return nil, apiURLs, parseErr
		}
		results = append(results, mapWorldResponse(world.Name, payload))
	}
	return results, apiURLs, nil
}

func V2FetchWorldDetails(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int) (domain.WorldDetailsResult, []string, error) {
	worldResult, worldSource, err := V2FetchWorld(ctx, oc, baseURL, worldName)
	if err != nil {
		return domain.WorldDetailsResult{}, []string{worldSource}, err
	}
	if len(worldResult.PlayersOnline) == 0 {
		return domain.WorldDetailsResult{
			Name:       worldResult.Name,
			Info:       worldResult.Info,
			Characters: []domain.CharacterResult{},
		}, []string{worldSource}, nil
	}
	characterURLs := make([]string, 0, len(worldResult.PlayersOnline))
	for _, player := range worldResult.PlayersOnline {
		query := url.Values{}
		query.Set("name", strings.TrimSpace(player.Name))
		characterURLs = append(characterURLs, fmt.Sprintf("%s/api/characters/search?%s", strings.TrimRight(baseURL, "/"), query.Encode()))
	}
	bodies, batchErr := oc.BatchFetchJSON(ctx, characterURLs)
	sources := append([]string{worldSource}, characterURLs...)
	if batchErr != nil {
		return domain.WorldDetailsResult{}, sources, batchErr
	}
	characters := make([]domain.CharacterResult, 0, len(bodies))
	for _, charURL := range characterURLs {
		body, ok := bodies[charURL]
		if !ok {
			continue
		}
		var payload characterAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			return domain.WorldDetailsResult{}, sources, parseErr
		}
		if payload.Player == nil {
			continue
		}
		characters = append(characters, mapCharacterResponse(payload))
	}
	return domain.WorldDetailsResult{
		Name:       worldResult.Name,
		Info:       worldResult.Info,
		Characters: characters,
	}, sources, nil
}

func V2FetchWorldDashboard(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int) (domain.WorldDashboardResult, []string, error) {
	worldURL := fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(strings.TrimSpace(worldName)))
	deathsURL := fmt.Sprintf("%s/api/deaths?page=1&world=%d", strings.TrimRight(baseURL, "/"), worldID)
	killstatsURL := fmt.Sprintf("%s/api/killstats?world=%d", strings.TrimRight(baseURL, "/"), worldID)
	batchURLs := []string{worldURL, deathsURL, killstatsURL}
	bodies, err := oc.BatchFetchJSON(ctx, batchURLs)
	if err != nil {
		return domain.WorldDashboardResult{}, batchURLs, err
	}

	var worldPayload worldDetailAPIResponse
	if parseErr := parseJSONBody(bodies[worldURL], &worldPayload); parseErr != nil {
		return domain.WorldDashboardResult{}, batchURLs, parseErr
	}
	worldResult := mapWorldResponse(worldName, worldPayload)

	var deathsPayload deathsAPIResponse
	if parseErr := parseJSONBody(bodies[deathsURL], &deathsPayload); parseErr != nil {
		return domain.WorldDashboardResult{}, batchURLs, parseErr
	}
	deathsResult := mapDeathsResponse(worldName, DeathsFilters{Page: 1}, deathsPayload)

	var killstatsPayload killstatisticsAPIResponse
	if parseErr := parseJSONBody(bodies[killstatsURL], &killstatsPayload); parseErr != nil {
		return domain.WorldDashboardResult{}, batchURLs, parseErr
	}
	killstatsResult := mapKillstatisticsResponse(worldName, killstatsPayload)

	return domain.WorldDashboardResult{
		World:          worldResult,
		RecentDeaths:   deathsResult,
		KillStatistics: killstatsResult,
	}, batchURLs, nil
}

func V2FetchKillstatisticsBatch(ctx context.Context, oc *OptimizedClient, baseURL string, worlds []validation.World) ([]domain.KillstatisticsResult, []string, error) {
	apiURLs := make([]string, 0, len(worlds))
	for _, world := range worlds {
		query := url.Values{}
		query.Set("world", strconv.Itoa(world.ID))
		apiURLs = append(apiURLs, fmt.Sprintf("%s/api/killstats?%s", strings.TrimRight(baseURL, "/"), query.Encode()))
	}
	bodies, err := oc.BatchFetchJSON(ctx, apiURLs)
	if err != nil {
		return nil, apiURLs, err
	}
	results := make([]domain.KillstatisticsResult, 0, len(worlds))
	for i, apiURL := range apiURLs {
		body, ok := bodies[apiURL]
		if !ok {
			return nil, apiURLs, fmt.Errorf("missing response for %s", apiURL)
		}
		var payload killstatisticsAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			return nil, apiURLs, parseErr
		}
		results = append(results, mapKillstatisticsResponse(worlds[i].Name, payload))
	}
	return results, apiURLs, nil
}

func V2FetchAllDeaths(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int, filters DeathsFilters) (domain.DeathsResult, []string, error) {
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

	return v2FetchAllPaginated(ctx, oc, buildURL, func(body string) (int, error) {
		return deathsTotalPagesFromBody(body)
	}, func(bodies []string, sources []string) (domain.DeathsResult, []string, error) {
		entries := make([]domain.DeathEntry, 0)
		totalDeaths := 0
		itemsPerPage := 0
		for idx, body := range bodies {
			var payload deathsAPIResponse
			if parseErr := parseJSONBody(body, &payload); parseErr != nil {
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
		return domain.DeathsResult{
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
		}, sources, nil
	})
}

func V2FetchAllBanishments(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int) (domain.BanishmentsResult, []string, error) {
	buildURL := func(page int) string {
		query := url.Values{}
		query.Set("world", strconv.Itoa(worldID))
		query.Set("page", strconv.Itoa(page))
		return fmt.Sprintf("%s/api/bans?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	}

	return v2FetchAllPaginated(ctx, oc, buildURL, func(body string) (int, error) {
		return banishmentsTotalPagesFromBody(body)
	}, func(bodies []string, sources []string) (domain.BanishmentsResult, []string, error) {
		entries := make([]domain.BanishmentEntry, 0)
		totalBans := 0
		for idx, body := range bodies {
			var payload banishmentsAPIResponse
			if parseErr := parseJSONBody(body, &payload); parseErr != nil {
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
		return domain.BanishmentsResult{
			World:      worldName,
			Page:       1,
			TotalBans:  totalBans,
			TotalPages: 1,
			Entries:    entries,
		}, sources, nil
	})
}

func V2FetchAllTransfers(ctx context.Context, oc *OptimizedClient, baseURL string, filters TransfersFilters) (domain.TransfersResult, []string, error) {
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

	return v2FetchAllPaginated(ctx, oc, buildURL, func(body string) (int, error) {
		return transfersTotalPagesFromBody(body)
	}, func(bodies []string, sources []string) (domain.TransfersResult, []string, error) {
		entries := make([]domain.TransferEntry, 0)
		totalTransfers := 0
		for idx, body := range bodies {
			var payload transfersAPIResponse
			if parseErr := parseJSONBody(body, &payload); parseErr != nil {
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
		return domain.TransfersResult{
			Filters: domain.TransferFilters{
				World:    filters.WorldName,
				MinLevel: filters.MinLevel,
			},
			Page:           1,
			TotalTransfers: totalTransfers,
			TotalPages:     1,
			Entries:        entries,
		}, sources, nil
	})
}

func V2FetchAllGuilds(ctx context.Context, oc *OptimizedClient, baseURL, worldName string, worldID int) (domain.GuildsResult, []string, error) {
	buildURL := func(page int) string {
		query := url.Values{}
		query.Set("world", strconv.Itoa(worldID))
		query.Set("page", strconv.Itoa(page))
		return fmt.Sprintf("%s/api/guilds?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	}

	return v2FetchAllPaginated(ctx, oc, buildURL, func(body string) (int, error) {
		return guildsTotalPagesFromBody(body)
	}, func(bodies []string, sources []string) (domain.GuildsResult, []string, error) {
		items := make([]domain.GuildListEntry, 0)
		totalCount := 0
		for idx, body := range bodies {
			var payload guildsAPIResponse
			if parseErr := parseJSONBody(body, &payload); parseErr != nil {
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
		return domain.GuildsResult{
			World:  worldName,
			Guilds: items,
			Active: items,
			Pagination: &domain.GuildsPagination{
				CurrentPage: 1,
				TotalPages:  1,
				TotalCount:  totalCount,
			},
		}, sources, nil
	})
}

func V2FetchAllCurrentAuctions(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.AuctionsResult, []string, error) {
	return v2FetchAllAuctions(ctx, oc, baseURL, "current")
}

func V2FetchAllAuctionHistory(ctx context.Context, oc *OptimizedClient, baseURL string) (domain.AuctionsResult, []string, error) {
	return v2FetchAllAuctions(ctx, oc, baseURL, "history")
}

func v2FetchAllAuctions(ctx context.Context, oc *OptimizedClient, baseURL, auctionType string) (domain.AuctionsResult, []string, error) {
	buildURL := func(page int) string {
		return buildAuctionListURL(baseURL, auctionType, page)
	}

	return v2FetchAllPaginated(ctx, oc, buildURL, func(body string) (int, error) {
		return auctionsTotalPagesFromBody(body)
	}, func(bodies []string, sources []string) (domain.AuctionsResult, []string, error) {
		entries := make([]domain.AuctionEntry, 0)
		totalResults := 0
		for idx, body := range bodies {
			var payload auctionListAPIResponse
			if parseErr := parseJSONBody(body, &payload); parseErr != nil {
				return domain.AuctionsResult{}, sources, parseErr
			}
			if idx == 0 {
				totalResults = payload.Pagination.Total
			}
			for _, row := range payload.Auctions {
				entries = append(entries, mapAuctionListEntry(row))
			}
		}
		if totalResults <= 0 {
			totalResults = len(entries)
		}
		return domain.AuctionsResult{
			Type:         auctionType,
			Page:         1,
			TotalResults: totalResults,
			TotalPages:   1,
			Entries:      entries,
			Pagination: &domain.AuctionsPagination{
				Page:       1,
				Limit:      auctionListLimit,
				Total:      totalResults,
				TotalPages: 1,
			},
		}, sources, nil
	})
}

const maxPaginatedPages = 50

func v2FetchAllPaginated[T any](
	ctx context.Context,
	oc *OptimizedClient,
	buildURL func(page int) string,
	extractTotalPages func(body string) (int, error),
	aggregate func(bodies []string, sources []string) (T, []string, error),
) (T, []string, error) {
	var zero T

	page1URL := buildURL(1)
	page1Body, err := oc.Fetcher.FetchJSON(ctx, page1URL)
	if err != nil {
		return zero, []string{page1URL}, err
	}

	totalPages, err := extractTotalPages(page1Body)
	if err != nil {
		return zero, []string{page1URL}, err
	}
	if totalPages <= 0 {
		totalPages = 1
	}
	if totalPages > maxPaginatedPages {
		totalPages = maxPaginatedPages
	}

	if totalPages == 1 {
		return aggregate([]string{page1Body}, []string{page1URL})
	}

	remainingURLs := make([]string, 0, totalPages-1)
	for page := 2; page <= totalPages; page++ {
		remainingURLs = append(remainingURLs, buildURL(page))
	}

	batchBodies, batchErr := oc.BatchFetchJSON(ctx, remainingURLs)
	if batchErr != nil {
		allSources := append([]string{page1URL}, remainingURLs...)
		return zero, allSources, batchErr
	}

	allBodies := make([]string, 0, totalPages)
	allSources := make([]string, 0, totalPages)
	allBodies = append(allBodies, page1Body)
	allSources = append(allSources, page1URL)
	for _, pageURL := range remainingURLs {
		body, ok := batchBodies[pageURL]
		if !ok {
			return zero, append([]string{page1URL}, remainingURLs...), fmt.Errorf("missing response for %s", pageURL)
		}
		allBodies = append(allBodies, body)
		allSources = append(allSources, pageURL)
	}

	return aggregate(allBodies, allSources)
}

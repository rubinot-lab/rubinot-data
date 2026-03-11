package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/scraper"
	"github.com/giovannirco/rubinot-data/internal/validation"
)

func v2GetWorlds(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	worlds, sourceURL, err := scraper.V2FetchWorlds(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "worlds",
		Payload:    worlds,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetWorld(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("name"))
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results, sources, err := scraper.V2FetchWorldBatch(c.Request.Context(), oc, resolvedBaseURL, worlds)
		if err != nil {
			return endpointResult{Sources: sources}, err
		}
		return endpointResult{
			PayloadKey: "worlds",
			Payload:    results,
			Sources:    sources,
		}, nil
	}

	canonicalWorld, _, ok := validator.WorldExists(worldInput)
	if !ok {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	world, sourceURL, err := scraper.V2FetchWorld(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "world",
		Payload:    world,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetWorldDetails(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("name"))
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.WorldDetailsResult, 0, len(worlds))
		sources := make([]string, 0)
		for _, world := range worlds {
			worldDetails, worldSources, err := scraper.V2FetchWorldDetails(c.Request.Context(), oc, resolvedBaseURL, world.Name, world.ID)
			if err != nil {
				return endpointResult{Sources: append(sources, worldSources...)}, err
			}
			results = append(results, worldDetails)
			sources = append(sources, worldSources...)
		}
		return endpointResult{
			PayloadKey: "worlds",
			Payload:    results,
			Sources:    sources,
		}, nil
	}

	canonicalWorld, worldID, ok := validator.WorldExists(worldInput)
	if !ok {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	world, sources, err := scraper.V2FetchWorldDetails(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "world",
		Payload:    world,
		Sources:    sources,
	}, nil
}

func v2GetWorldDashboard(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("name"))
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.WorldDashboardResult, 0, len(worlds))
		sources := make([]string, 0)
		for _, world := range worlds {
			dashboard, worldSources, err := scraper.V2FetchWorldDashboard(c.Request.Context(), oc, resolvedBaseURL, world.Name, world.ID)
			if err != nil {
				return endpointResult{Sources: append(sources, worldSources...)}, err
			}
			results = append(results, dashboard)
			sources = append(sources, worldSources...)
		}
		return endpointResult{
			PayloadKey: "worlds",
			Payload:    results,
			Sources:    sources,
		}, nil
	}

	canonicalWorld, worldID, ok := validator.WorldExists(worldInput)
	if !ok {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	dashboard, sources, err := scraper.V2FetchWorldDashboard(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "dashboard",
		Payload:    dashboard,
		Sources:    sources,
	}, nil
}

func v2GetHighscores(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	categoryInput := strings.TrimSpace(c.Param("category"))
	vocationInput := strings.TrimSpace(c.Param("vocation"))

	category, categoryOK := validator.ResolveHighscoreCategory(categoryInput)
	if !categoryOK {
		return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, "highscore category does not exist", nil)
	}

	vocation, vocationOK := validator.ResolveHighscoreVocation(vocationInput)
	if !vocationOK {
		return endpointResult{}, validation.NewError(validation.ErrorVocationDoesNotExist, "vocation does not exist", nil)
	}

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.HighscoresResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			highscores, sourceURL, err := scraper.V2FetchHighscores(c.Request.Context(), oc, resolvedBaseURL, world.Name, category, vocation)
			if err != nil {
				return endpointResult{Sources: append(allSources, sourceURL)}, err
			}
			results = append(results, highscores)
			allSources = append(allSources, sourceURL)
		}
		return endpointResult{
			PayloadKey: "highscores",
			Payload:    results,
			Sources:    allSources,
		}, nil
	}

	canonicalWorld, _, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	highscores, sourceURL, err := scraper.V2FetchHighscores(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, category, vocation)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "highscores",
		Payload:    highscores,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetKillstatistics(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results, sources, err := scraper.V2FetchKillstatisticsBatch(c.Request.Context(), oc, resolvedBaseURL, worlds)
		if err != nil {
			return endpointResult{Sources: sources}, err
		}
		return endpointResult{
			PayloadKey: "killstatistics",
			Payload:    results,
			Sources:    sources,
		}, nil
	}

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	killstatistics, sourceURL, err := scraper.V2FetchKillstatistics(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "killstatistics",
		Payload:    killstatistics,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetDeaths(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	filters, err := parseDeathsFilters(c)
	if err != nil {
		return endpointResult{}, err
	}

	deaths, sourceURL, fetchErr := scraper.V2FetchDeaths(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID, filters)
	if fetchErr != nil {
		return endpointResult{Sources: []string{sourceURL}}, fetchErr
	}
	return endpointResult{
		PayloadKey: "deaths",
		Payload:    deaths,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetAllDeaths(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	levelFilter, levelErr := validation.ParseLevelFilter(c.Query("level"))
	if levelErr != nil {
		return endpointResult{}, levelErr
	}

	pvpValue, pvpProvided, pvpErr := validation.ParsePvPOnlyFilter(c.Query("pvp"))
	if pvpErr != nil {
		return endpointResult{}, pvpErr
	}

	var pvpOnly *bool
	if pvpProvided {
		pvpOnly = &pvpValue
	}

	filters := scraper.DeathsFilters{
		MinLevel: levelFilter,
		PvPOnly:  pvpOnly,
		Guild:    strings.TrimSpace(c.Query("guild")),
	}

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.DeathsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			deaths, sources, err := scraper.V2FetchAllDeaths(c.Request.Context(), oc, resolvedBaseURL, world.Name, world.ID, filters)
			if err != nil {
				return endpointResult{Sources: append(allSources, sources...)}, err
			}
			results = append(results, deaths)
			allSources = append(allSources, sources...)
		}
		return endpointResult{
			PayloadKey: "deaths",
			Payload:    results,
			Sources:    allSources,
		}, nil
	}

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	deaths, sources, err := scraper.V2FetchAllDeaths(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID, filters)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "deaths",
		Payload:    deaths,
		Sources:    sources,
	}, nil
}

func v2GetBanishments(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	page := 1
	pageInput := strings.TrimSpace(c.Query("page"))
	if pageInput != "" {
		parsedPage, pageErr := validation.ParsePage(pageInput)
		if pageErr != nil {
			return endpointResult{}, pageErr
		}
		page = parsedPage
	}

	banishments, sourceURL, err := scraper.V2FetchBanishments(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID, page)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "banishments",
		Payload:    banishments,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetAllBanishments(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.BanishmentsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			banishments, sources, err := scraper.V2FetchAllBanishments(c.Request.Context(), oc, resolvedBaseURL, world.Name, world.ID)
			if err != nil {
				return endpointResult{Sources: append(allSources, sources...)}, err
			}
			results = append(results, banishments)
			allSources = append(allSources, sources...)
		}
		return endpointResult{
			PayloadKey: "banishments",
			Payload:    results,
			Sources:    allSources,
		}, nil
	}

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	banishments, sources, err := scraper.V2FetchAllBanishments(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "banishments",
		Payload:    banishments,
		Sources:    sources,
	}, nil
}

func v2GetTransfers(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Query("world"))
	levelInput := strings.TrimSpace(c.Query("level"))
	pageInput := strings.TrimSpace(c.Query("page"))

	worldID := 0
	canonicalWorld := ""
	if worldInput != "" {
		var worldOK bool
		canonicalWorld, worldID, worldOK = validator.WorldExists(worldInput)
		if !worldOK {
			return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
		}
	}

	minLevel, levelErr := validation.ParseLevelFilter(levelInput)
	if levelErr != nil {
		return endpointResult{}, levelErr
	}

	page := 1
	if pageInput != "" {
		var pageErr error
		page, pageErr = validation.ParsePage(pageInput)
		if pageErr != nil {
			return endpointResult{}, pageErr
		}
	}

	transfers, sourceURL, err := scraper.V2FetchTransfers(c.Request.Context(), oc, resolvedBaseURL, scraper.TransfersFilters{
		WorldID:   worldID,
		WorldName: canonicalWorld,
		MinLevel:  minLevel,
		Page:      page,
	})
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "transfers",
		Payload:    transfers,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetAllTransfers(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Query("world"))
	levelInput := strings.TrimSpace(c.Query("level"))

	worldID := 0
	canonicalWorld := ""
	if worldInput != "" {
		var worldOK bool
		canonicalWorld, worldID, worldOK = validator.WorldExists(worldInput)
		if !worldOK {
			return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
		}
	}

	minLevel, levelErr := validation.ParseLevelFilter(levelInput)
	if levelErr != nil {
		return endpointResult{}, levelErr
	}

	transfers, sources, err := scraper.V2FetchAllTransfers(c.Request.Context(), oc, resolvedBaseURL, scraper.TransfersFilters{
		WorldID:   worldID,
		WorldName: canonicalWorld,
		MinLevel:  minLevel,
	})
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "transfers",
		Payload:    transfers,
		Sources:    sources,
	}, nil
}

func v2GetCharacter(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	characterInput := strings.TrimSpace(c.Param("name"))
	canonicalName, validationErr := validation.IsCharacterNameValid(characterInput)
	if validationErr != nil {
		return endpointResult{}, validationErr
	}

	character, sourceURL, err := scraper.V2FetchCharacter(c.Request.Context(), oc, resolvedBaseURL, canonicalName)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "character",
		Payload:    character,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetGuild(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	guildInput := strings.TrimSpace(c.Param("name"))
	canonicalName, validationErr := validation.IsGuildNameValid(guildInput)
	if validationErr != nil {
		return endpointResult{}, validationErr
	}

	guild, sourceURL, err := scraper.V2FetchGuild(c.Request.Context(), oc, resolvedBaseURL, canonicalName)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "guild",
		Payload:    guild,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetGuilds(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	page := 1
	pageInput := strings.TrimSpace(c.Query("page"))
	if pageInput != "" {
		parsedPage, pageErr := validation.ParsePage(pageInput)
		if pageErr != nil {
			return endpointResult{}, pageErr
		}
		page = parsedPage
	}

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.GuildsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			guilds, sourceURL, err := scraper.V2FetchGuilds(c.Request.Context(), oc, resolvedBaseURL, world.Name, world.ID, 1)
			if err != nil {
				return endpointResult{Sources: append(allSources, sourceURL)}, err
			}
			results = append(results, guilds)
			allSources = append(allSources, sourceURL)
		}
		return endpointResult{
			PayloadKey: "guilds",
			Payload:    results,
			Sources:    allSources,
		}, nil
	}

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	guilds, sourceURL, err := scraper.V2FetchGuilds(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID, page)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "guilds",
		Payload:    guilds,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetAllGuilds(c *gin.Context, validator *validation.Validator, oc *scraper.OptimizedClient) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.GuildsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			guilds, sources, err := scraper.V2FetchAllGuilds(c.Request.Context(), oc, resolvedBaseURL, world.Name, world.ID)
			if err != nil {
				return endpointResult{Sources: append(allSources, sources...)}, err
			}
			results = append(results, guilds)
			allSources = append(allSources, sources...)
		}
		return endpointResult{
			PayloadKey: "guilds",
			Payload:    results,
			Sources:    allSources,
		}, nil
	}

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	guilds, sources, err := scraper.V2FetchAllGuilds(c.Request.Context(), oc, resolvedBaseURL, canonicalWorld, worldID)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "guilds",
		Payload:    guilds,
		Sources:    sources,
	}, nil
}

func v2GetBoosted(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	boosted, sourceURL, err := scraper.V2FetchBoosted(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	boosted.Boss.ImageURL = fmt.Sprintf("/v2/outfit?type=%d", boosted.Boss.LookType)
	boosted.Monster.ImageURL = fmt.Sprintf("/v2/outfit?type=%d", boosted.Monster.LookType)

	return endpointResult{
		PayloadKey: "boosted",
		Payload:    boosted,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetMaintenance(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	maintenance, sourceURL, err := scraper.V2FetchMaintenance(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "maintenance",
		Payload:    maintenance,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetCurrentAuctions(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	pageInput := strings.TrimSpace(c.Param("page"))
	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}

	auctions, sourceURL, err := scraper.V2FetchCurrentAuctions(c.Request.Context(), oc, resolvedBaseURL, page)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetAllCurrentAuctions(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	auctions, sources, err := scraper.V2FetchAllCurrentAuctions(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func v2GetAuctionHistory(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	pageInput := strings.TrimSpace(c.Param("page"))
	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}

	auctions, sourceURL, err := scraper.V2FetchAuctionHistory(c.Request.Context(), oc, resolvedBaseURL, page)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetAllAuctionHistory(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	auctions, sources, err := scraper.V2FetchAllAuctionHistory(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func v2GetAuctionDetail(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	auctionIDInput := strings.TrimSpace(c.Param("id"))
	auctionID, idErr := validation.ParseAuctionID(auctionIDInput)
	if idErr != nil {
		return endpointResult{}, idErr
	}

	auction, sources, err := scraper.V2FetchAuctionDetail(c.Request.Context(), oc, resolvedBaseURL, auctionID)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "auction",
		Payload:    auction,
		Sources:    sources,
	}, nil
}

func v2GetNewsByID(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	newsIDInput := strings.TrimSpace(c.Param("news_id"))
	newsID, parseErr := validation.ParseNewsID(newsIDInput)
	if parseErr != nil {
		return endpointResult{}, parseErr
	}

	news, sources, err := scraper.V2FetchNewsByID(c.Request.Context(), oc, resolvedBaseURL, newsID)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "news",
		Payload:    news,
		Sources:    sources,
	}, nil
}

func v2GetNewsArchive(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	archiveDays, daysErr := validation.ParseArchiveDays(c.Query("days"), 90)
	if daysErr != nil {
		return endpointResult{}, daysErr
	}

	newsList, sourceURL, err := scraper.V2FetchNewsArchive(c.Request.Context(), oc, resolvedBaseURL, archiveDays)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "newslist",
		Payload:    newsList,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetNewsLatest(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	newsList, sourceURL, err := scraper.V2FetchNewsLatest(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "newslist",
		Payload:    newsList,
		Sources:    []string{sourceURL},
	}, nil
}

func v2GetNewsTicker(c *gin.Context, oc *scraper.OptimizedClient) (endpointResult, error) {
	newsList, sourceURL, err := scraper.V2FetchNewsTicker(c.Request.Context(), oc, resolvedBaseURL)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "newslist",
		Payload:    newsList,
		Sources:    []string{sourceURL},
	}, nil
}

func parseDeathsFilters(c *gin.Context) (scraper.DeathsFilters, error) {
	page := 1
	pageInput := strings.TrimSpace(c.Query("page"))
	if pageInput != "" {
		parsedPage, pageErr := validation.ParsePage(pageInput)
		if pageErr != nil {
			return scraper.DeathsFilters{}, pageErr
		}
		page = parsedPage
	}

	levelFilter, levelErr := validation.ParseLevelFilter(c.Query("level"))
	if levelErr != nil {
		return scraper.DeathsFilters{}, levelErr
	}

	pvpValue, pvpProvided, pvpErr := validation.ParsePvPOnlyFilter(c.Query("pvp"))
	if pvpErr != nil {
		return scraper.DeathsFilters{}, pvpErr
	}

	var pvpOnly *bool
	if pvpProvided {
		pvpOnly = &pvpValue
	}

	return scraper.DeathsFilters{
		MinLevel: levelFilter,
		PvPOnly:  pvpOnly,
		Page:     page,
		Guild:    strings.TrimSpace(c.Query("guild")),
	}, nil
}

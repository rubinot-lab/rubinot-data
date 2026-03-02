package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/scraper"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultRubinotBaseURL  = "https://rubinot.com.br"
	defaultScrapeTimeoutMS = 120000
	defaultServiceVersion  = "dev"
)

var (
	resolvedBaseURL  string
	resolvedOpts     scraper.FetchOptions
	currentValidator atomic.Pointer[validation.Validator]
)

func NewRouter() (*gin.Engine, error) {
	resolvedBaseURL = getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	resolvedOpts = scrapeFetchOptions()

	validator, err := bootstrapValidator(context.Background())
	if err != nil {
		return nil, err
	}
	currentValidator.Store(validator)
	startValidatorRefresh()

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(metricsMiddleware())

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "rubinot-data api up"})
	})
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/versions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "rubinot-data",
			"version": getEnv("APP_VERSION", defaultServiceVersion),
			"commit":  getEnv("APP_COMMIT", defaultAPICommit),
		})
	})
	router.GET("/openapi.json", docsSpec)
	router.GET("/docs", docsPage)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/v1")
	{
		v1.GET("/worlds", handleEndpoint(getWorlds))
		v1.GET("/world/:name/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getWorldDetails(c, getValidator())
		}))
		v1.GET("/world/:name/dashboard", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getWorldDashboard(c, getValidator())
		}))
		v1.GET("/world/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getWorld(c, getValidator())
		}))
		v1.GET("/highscores/categories", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			validator := getValidator()
			return endpointResult{
				PayloadKey: "categories",
				Payload:    validator.AllCategories(),
				Sources:    []string{},
			}, nil
		}))
		v1.GET("/highscores/:world", redirectHighscoresWorld)
		v1.GET("/highscores/:world/:category", redirectHighscoresCategory)
		v1.GET("/highscores/:world/:category/:vocation", redirectHighscoresVocation)
		v1.GET("/highscores/:world/:category/:vocation/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllHighscores(c, getValidator())
		}))
		v1.GET("/highscores/:world/:category/:vocation/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getHighscores(c, getValidator())
		}))
		v1.GET("/killstatistics/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getKillstatistics(c, getValidator())
		}))
		v1.GET("/news/id/:news_id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getNewsByID(c)
		}))
		v1.GET("/news/archive", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getNewsArchive(c)
		}))
		v1.GET("/news/latest", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getNewsLatest(c)
		}))
		v1.GET("/news/newsticker", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getNewsNewsticker(c)
		}))
		v1.GET("/boosted", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getBoosted(c)
		}))
		v1.GET("/maintenance", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getMaintenance(c)
		}))
		v1.GET("/geo-language", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getGeoLanguage(c)
		}))
		v1.GET("/outfit", getOutfit)
		v1.GET("/outfit/:name", getOutfitByCharacterName)
		v1.GET("/events/schedule", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getEventsSchedule(c)
		}))
		v1.GET("/events/calendar", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getEventsCalendar(c)
		}))
		v1.GET("/auctions/current/all/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllCurrentAuctionsDetails(c)
		}))
		v1.GET("/auctions/current/:page/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getCurrentAuctionsDetails(c)
		}))
		v1.GET("/auctions/current/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllCurrentAuctions(c)
		}))
		v1.GET("/auctions/current/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getCurrentAuctions(c)
		}))
		v1.GET("/auctions/history/all/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllAuctionHistoryDetails(c)
		}))
		v1.GET("/auctions/history/:page/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAuctionHistoryDetails(c)
		}))
		v1.GET("/auctions/history/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllAuctionHistory(c)
		}))
		v1.GET("/auctions/history/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAuctionHistory(c)
		}))
		v1.GET("/auctions/:id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAuctionDetail(c)
		}))
		v1.GET("/deaths/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllDeaths(c, getValidator())
		}))
		v1.GET("/deaths/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getDeaths(c, getValidator())
		}))
		v1.GET("/banishments/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllBanishments(c, getValidator())
		}))
		v1.GET("/banishments/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getBanishments(c, getValidator())
		}))
		v1.GET("/transfers/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllTransfers(c, getValidator())
		}))
		v1.GET("/transfers", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getTransfers(c, getValidator())
		}))
		v1.GET("/character/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getCharacter(c)
		}))
		v1.GET("/guild/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getGuild(c)
		}))
		v1.GET("/guilds/:world/all/details", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllGuildsDetails(c, getValidator())
		}))
		v1.GET("/guilds/:world/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllGuilds(c, getValidator())
		}))
		v1.GET("/guilds/:world/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getGuildsPage(c, getValidator())
		}))
		v1.GET("/guilds/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getGuilds(c, getValidator())
		}))
		v1.GET("/house/:world/:house_id", handleEndpoint(deprecatedHousesEndpoint))
		v1.GET("/houses/towns", handleEndpoint(deprecatedHousesEndpoint))
		v1.GET("/houses/:world/:town", handleEndpoint(deprecatedHousesEndpoint))

		if getEnv("ENABLE_RAW_PROXY", "") == "true" {
			v1.POST("/upstream/raw", handleEndpoint(postUpstreamRaw))
		}
		v1.POST("/characters/batch", handleEndpoint(postCharactersBatch))
		v1.POST("/characters/compare", handleEndpoint(postCharactersCompare))
		v1.POST("/highscores/:category/cross-world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return postHighscoresCrossWorld(c, getValidator())
		}))
		v1.POST("/highscores/multi-category", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return postHighscoresMultiCategory(c, getValidator())
		}))
		v1.POST("/guilds/batch", handleEndpoint(postGuildsBatch))
		v1.POST("/auctions/filter", handleEndpoint(postAuctionsFilter))
		v1.POST("/killstatistics/batch", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return postKillstatisticsBatch(c, getValidator())
		}))
		v1.GET("/bans", handleEndpoint(getBans))
		v1.GET("/news/all", handleEndpoint(getNewsAll))
	}

	return router, nil
}

func getWorlds(c *gin.Context) (endpointResult, error) {
	worlds, sourceURL, err := scraper.FetchWorlds(c.Request.Context(), resolvedBaseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "worlds",
		Payload:    worlds,
		Sources:    []string{sourceURL},
	}, nil
}

func getWorld(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("name"))
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.WorldResult, 0, len(worlds))
		sources := make([]string, 0, len(worlds))
		for _, world := range worlds {
			worldResult, sourceURL, err := scraper.FetchWorld(c.Request.Context(), resolvedBaseURL, world.Name, resolvedOpts)
			if err != nil {
				return endpointResult{Sources: append(sources, sourceURL)}, err
			}
			results = append(results, worldResult)
			sources = append(sources, sourceURL)
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

	baseURL := resolvedBaseURL
	world, sourceURL, err := scraper.FetchWorld(c.Request.Context(), baseURL, canonicalWorld, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "world",
		Payload:    world,
		Sources:    []string{sourceURL},
	}, nil
}

func getWorldDetails(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("name"))
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.WorldDetailsResult, 0, len(worlds))
		sources := make([]string, 0)
		for _, world := range worlds {
			worldDetails, worldSources, err := scraper.FetchWorldDetails(c.Request.Context(), resolvedBaseURL, world.Name, world.ID, resolvedOpts)
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

	world, sources, err := scraper.FetchWorldDetails(c.Request.Context(), resolvedBaseURL, canonicalWorld, worldID, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "world",
		Payload:    world,
		Sources:    sources,
	}, nil
}

func getWorldDashboard(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("name"))
	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.WorldDashboardResult, 0, len(worlds))
		sources := make([]string, 0)
		for _, world := range worlds {
			worldDashboard, worldSources, err := scraper.FetchWorldDashboard(c.Request.Context(), resolvedBaseURL, world.Name, world.ID, resolvedOpts)
			if err != nil {
				return endpointResult{Sources: append(sources, worldSources...)}, err
			}
			results = append(results, worldDashboard)
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

	dashboard, sources, err := scraper.FetchWorldDashboard(c.Request.Context(), resolvedBaseURL, canonicalWorld, worldID, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "dashboard",
		Payload:    dashboard,
		Sources:    sources,
	}, nil
}

func getCharacter(c *gin.Context) (endpointResult, error) {
	characterInput := strings.TrimSpace(c.Param("name"))
	canonicalName, validationErr := validation.IsCharacterNameValid(characterInput)
	if validationErr != nil {
		return endpointResult{}, validationErr
	}

	baseURL := resolvedBaseURL
	character, sourceURL, err := scraper.FetchCharacter(c.Request.Context(), baseURL, canonicalName, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "character",
		Payload:    character,
		Sources:    []string{sourceURL},
	}, nil
}

func getGuild(c *gin.Context) (endpointResult, error) {
	guildInput := strings.TrimSpace(c.Param("name"))
	canonicalName, validationErr := validation.IsGuildNameValid(guildInput)
	if validationErr != nil {
		return endpointResult{}, validationErr
	}

	baseURL := resolvedBaseURL
	guild, sourceURL, err := scraper.FetchGuild(c.Request.Context(), baseURL, canonicalName, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "guild",
		Payload:    guild,
		Sources:    []string{sourceURL},
	}, nil
}

func getGuilds(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	page := 1
	pageInput := strings.TrimSpace(c.Query("page"))
	if pageInput != "" {
		parsedPage, pageErr := validation.ParsePage(pageInput)
		if pageErr != nil {
			return endpointResult{}, pageErr
		}
		page = parsedPage
	}
	return getGuildsForPage(c, validator, page)
}

func getGuildsPage(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	pageInput := strings.TrimSpace(c.Param("page"))
	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}
	return getGuildsForPage(c, validator, page)
}

func getAllGuilds(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.GuildsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			guilds, sources, err := scraper.FetchAllGuilds(c.Request.Context(), resolvedBaseURL, world.Name, world.ID, resolvedOpts)
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

	guilds, sources, err := scraper.FetchAllGuilds(c.Request.Context(), resolvedBaseURL, canonicalWorld, worldID, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "guilds",
		Payload:    guilds,
		Sources:    sources,
	}, nil
}

func getAllGuildsDetails(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.GuildsDetailsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			guilds, sources, err := scraper.FetchAllGuildsDetails(c.Request.Context(), resolvedBaseURL, world.Name, world.ID, resolvedOpts)
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

	guilds, sources, err := scraper.FetchAllGuildsDetails(c.Request.Context(), resolvedBaseURL, canonicalWorld, worldID, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "guilds",
		Payload:    guilds,
		Sources:    sources,
	}, nil
}

func getGuildsForPage(c *gin.Context, validator *validation.Validator, page int) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	guilds, sourceURL, err := scraper.FetchGuilds(c.Request.Context(), resolvedBaseURL, canonicalWorld, worldID, page, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "guilds",
		Payload:    guilds,
		Sources:    []string{sourceURL},
	}, nil
}

func deprecatedHousesEndpoint(_ *gin.Context) (endpointResult, error) {
	return endpointResult{}, validation.NewError(
		validation.ErrorEndpointDeprecated,
		"houses endpoints are deprecated: house data is available via /v1/character/:name",
		nil,
	)
}

func getHouses(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	townInput := strings.TrimSpace(c.Param("town"))

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}
	canonicalTown, townID, townOK := validator.TownExists(townInput)
	if !townOK {
		return endpointResult{}, validation.NewError(validation.ErrorTownDoesNotExist, "town does not exist", nil)
	}

	baseURL := resolvedBaseURL
	houses, sourceURL, err := scraper.FetchHouses(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		canonicalTown,
		townID,
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "houses",
		Payload:    houses,
		Sources:    []string{sourceURL},
	}, nil
}

func getHouse(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	houseIDInput := strings.TrimSpace(c.Param("house_id"))

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	houseID, parseErr := validation.ParseHouseID(houseIDInput)
	if parseErr != nil {
		return endpointResult{}, parseErr
	}

	baseURL := resolvedBaseURL
	house, sourceURL, err := scraper.FetchHouse(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		houseID,
		validator.AllTowns(),
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "house",
		Payload:    house,
		Sources:    []string{sourceURL},
	}, nil
}

func redirectHighscoresWorld(c *gin.Context) {
	world := strings.TrimSpace(c.Param("world"))
	location := fmt.Sprintf("/v1/highscores/%s/experience/all/1", url.PathEscape(world))
	c.Redirect(http.StatusFound, location)
}

func redirectHighscoresCategory(c *gin.Context) {
	world := strings.TrimSpace(c.Param("world"))
	category := strings.TrimSpace(c.Param("category"))
	location := fmt.Sprintf("/v1/highscores/%s/%s/all/1", url.PathEscape(world), url.PathEscape(category))
	c.Redirect(http.StatusFound, location)
}

func redirectHighscoresVocation(c *gin.Context) {
	world := strings.TrimSpace(c.Param("world"))
	category := strings.TrimSpace(c.Param("category"))
	vocation := strings.TrimSpace(c.Param("vocation"))
	location := fmt.Sprintf("/v1/highscores/%s/%s/%s/1", url.PathEscape(world), url.PathEscape(category), url.PathEscape(vocation))
	c.Redirect(http.StatusFound, location)
}

func getHighscores(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	categoryInput := strings.TrimSpace(c.Param("category"))
	vocationInput := strings.TrimSpace(c.Param("vocation"))
	pageInput := strings.TrimSpace(c.Param("page"))

	category, categoryOK := validator.ResolveHighscoreCategory(categoryInput)
	if !categoryOK {
		return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, "highscore category does not exist", nil)
	}

	vocation, vocationOK := validator.ResolveHighscoreVocation(vocationInput)
	if !vocationOK {
		return endpointResult{}, validation.NewError(validation.ErrorVocationDoesNotExist, "vocation does not exist", nil)
	}

	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}

	var (
		highscores domain.HighscoresResult
		sourceURL  string
		err        error
	)
	if isAllWorldsToken(worldInput) {
		highscores, sourceURL, err = scraper.FetchHighscoresAllWorlds(
			c.Request.Context(),
			resolvedBaseURL,
			category,
			vocation,
			page,
			resolvedOpts,
		)
	} else {
		canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
		if !worldOK {
			return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
		}
		highscores, sourceURL, err = scraper.FetchHighscores(
			c.Request.Context(),
			resolvedBaseURL,
			canonicalWorld,
			worldID,
			category,
			vocation,
			page,
			resolvedOpts,
		)
	}
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "highscores",
		Payload:    highscores,
		Sources:    []string{sourceURL},
	}, nil
}

func getAllHighscores(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
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
		grouped, sources, err := scraper.FetchAllHighscoresPerWorld(
			c.Request.Context(),
			resolvedBaseURL,
			validator.AllWorlds(),
			category,
			vocation,
			resolvedOpts,
		)
		if err != nil {
			return endpointResult{Sources: sources}, err
		}
		return endpointResult{
			PayloadKey: "highscores",
			Payload:    grouped,
			Sources:    sources,
		}, nil
	}

	var (
		highscores domain.HighscoresResult
		sourceURL  string
		err        error
	)
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}
	highscores, sourceURL, err = scraper.FetchAllHighscores(
		c.Request.Context(),
		resolvedBaseURL,
		canonicalWorld,
		worldID,
		category,
		vocation,
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "highscores",
		Payload:    highscores,
		Sources:    []string{sourceURL},
	}, nil
}

func getKillstatistics(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results, sources, err := scraper.FetchAllWorldsKillstatistics(c.Request.Context(), resolvedBaseURL, worlds, resolvedOpts)
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

	baseURL := resolvedBaseURL
	killstatistics, sourceURL, err := scraper.FetchKillstatistics(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "killstatistics",
		Payload:    killstatistics,
		Sources:    []string{sourceURL},
	}, nil
}

func getNewsByID(c *gin.Context) (endpointResult, error) {
	newsIDInput := strings.TrimSpace(c.Param("news_id"))
	newsID, parseErr := validation.ParseNewsID(newsIDInput)
	if parseErr != nil {
		return endpointResult{}, parseErr
	}

	baseURL := resolvedBaseURL
	news, sources, err := scraper.FetchNewsByID(c.Request.Context(), baseURL, newsID, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "news",
		Payload:    news,
		Sources:    sources,
	}, nil
}

func getNewsArchive(c *gin.Context) (endpointResult, error) {
	archiveDays, daysErr := validation.ParseArchiveDays(c.Query("days"), 90)
	if daysErr != nil {
		return endpointResult{}, daysErr
	}

	baseURL := resolvedBaseURL
	newsList, sourceURL, err := scraper.FetchNewsArchive(c.Request.Context(), baseURL, archiveDays, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "newslist",
		Payload:    newsList,
		Sources:    []string{sourceURL},
	}, nil
}

func getNewsLatest(c *gin.Context) (endpointResult, error) {
	baseURL := resolvedBaseURL
	newsList, sourceURL, err := scraper.FetchNewsLatest(c.Request.Context(), baseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "newslist",
		Payload:    newsList,
		Sources:    []string{sourceURL},
	}, nil
}

func getNewsNewsticker(c *gin.Context) (endpointResult, error) {
	baseURL := resolvedBaseURL
	newsList, sourceURL, err := scraper.FetchNewsTicker(c.Request.Context(), baseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "newslist",
		Payload:    newsList,
		Sources:    []string{sourceURL},
	}, nil
}

func getBoosted(c *gin.Context) (endpointResult, error) {
	baseURL := resolvedBaseURL
	boosted, sourceURL, err := scraper.FetchBoosted(c.Request.Context(), baseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	boosted.Boss.ImageURL = fmt.Sprintf("/v1/outfit?type=%d", boosted.Boss.LookType)
	boosted.Monster.ImageURL = fmt.Sprintf("/v1/outfit?type=%d", boosted.Monster.LookType)

	return endpointResult{
		PayloadKey: "boosted",
		Payload:    boosted,
		Sources:    []string{sourceURL},
	}, nil
}

func getMaintenance(c *gin.Context) (endpointResult, error) {
	baseURL := resolvedBaseURL
	maintenance, sourceURL, err := scraper.FetchMaintenance(c.Request.Context(), baseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "maintenance",
		Payload:    maintenance,
		Sources:    []string{sourceURL},
	}, nil
}

func getGeoLanguage(c *gin.Context) (endpointResult, error) {
	baseURL := resolvedBaseURL
	geoLanguage, sourceURL, err := scraper.FetchGeoLanguage(c.Request.Context(), baseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "geo_language",
		Payload:    geoLanguage,
		Sources:    []string{sourceURL},
	}, nil
}

func getOutfit(c *gin.Context) {
	baseURL := resolvedBaseURL
	body, contentType, sourceURL, err := scraper.FetchOutfitImage(c.Request.Context(), baseURL, c.Request.URL.RawQuery, resolvedOpts)
	if err != nil {
		writeOutfitError(c, err, []string{sourceURL})
		return
	}

	writeOutfitResponse(c, body, contentType, sourceURL)
}

func getOutfitByCharacterName(c *gin.Context) {
	characterInput := strings.TrimSpace(c.Param("name"))
	canonicalName, validationErr := validation.IsCharacterNameValid(characterInput)
	if validationErr != nil {
		writeOutfitError(c, validationErr, []string{})
		return
	}

	character, characterSourceURL, err := scraper.FetchCharacter(c.Request.Context(), resolvedBaseURL, canonicalName, resolvedOpts)
	if err != nil {
		writeOutfitError(c, err, []string{characterSourceURL})
		return
	}

	if character.CharacterInfo.Outfit == nil {
		writeOutfitError(c, validation.NewError(validation.ErrorUpstreamUnknown, "character outfit is not available", nil), []string{characterSourceURL})
		return
	}

	query := c.Request.URL.Query()
	setOutfitDefaultParam(query, "type", "looktype", strconv.Itoa(character.CharacterInfo.Outfit.LookType))
	setOutfitDefaultParam(query, "head", "lookhead", strconv.Itoa(character.CharacterInfo.Outfit.LookHead))
	setOutfitDefaultParam(query, "body", "lookbody", strconv.Itoa(character.CharacterInfo.Outfit.LookBody))
	setOutfitDefaultParam(query, "legs", "looklegs", strconv.Itoa(character.CharacterInfo.Outfit.LookLegs))
	setOutfitDefaultParam(query, "feet", "lookfeet", strconv.Itoa(character.CharacterInfo.Outfit.LookFeet))
	setOutfitDefaultParam(query, "addons", "lookaddons", strconv.Itoa(character.CharacterInfo.Outfit.LookAddons))

	body, contentType, outfitSourceURL, err := scraper.FetchOutfitImage(c.Request.Context(), resolvedBaseURL, query.Encode(), resolvedOpts)
	if err != nil {
		writeOutfitError(c, err, []string{characterSourceURL, outfitSourceURL})
		return
	}

	writeOutfitResponse(c, body, contentType, outfitSourceURL)
}

func setOutfitDefaultParam(values url.Values, primary, legacy, fallback string) {
	if strings.TrimSpace(values.Get(primary)) == "" && strings.TrimSpace(values.Get(legacy)) == "" {
		values.Set(primary, fallback)
	}
}

func writeOutfitResponse(c *gin.Context, body []byte, contentType, sourceURL string) {
	formattedBody, formattedContentType, err := maybeFormatOutfitImage(c.Query("format"), body, contentType)
	if err != nil {
		writeOutfitError(c, validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("failed to render outfit image: %v", err), err), []string{sourceURL})
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Header("X-Source-URL", sourceURL)
	c.Data(http.StatusOK, formattedContentType, formattedBody)
}

func writeOutfitError(c *gin.Context, err error, sources []string) {
	errorCode := resolveErrorCode(err)
	httpCode := statusCodeFromErrorCode(errorCode)
	message := resolveErrorMessage(errorCode, err)
	if httpCode == http.StatusBadRequest {
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}
		validationRejections.WithLabelValues(route, strconv.Itoa(errorCode)).Inc()
	}
	c.JSON(httpCode, errorEnvelope(httpCode, errorCode, message, sources))
}

func getEventsSchedule(c *gin.Context) (endpointResult, error) {
	month, monthErr := validation.ParseMonth(c.Query("month"))
	if monthErr != nil {
		return endpointResult{}, monthErr
	}

	year, yearErr := validation.ParseYear(c.Query("year"))
	if yearErr != nil {
		return endpointResult{}, yearErr
	}

	if month > 0 && year == 0 {
		return endpointResult{}, validation.NewError(validation.ErrorYearInvalid, "year is required when month is provided", nil)
	}
	if year > 0 && month == 0 {
		return endpointResult{}, validation.NewError(validation.ErrorMonthInvalid, "month is required when year is provided", nil)
	}

	baseURL := resolvedBaseURL
	events, sourceURL, err := scraper.FetchEventsSchedule(c.Request.Context(), baseURL, month, year, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "events",
		Payload:    events,
		Sources:    []string{sourceURL},
	}, nil
}

func getEventsCalendar(c *gin.Context) (endpointResult, error) {
	baseURL := resolvedBaseURL
	events, sourceURL, err := scraper.FetchEventsCalendar(c.Request.Context(), baseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "events",
		Payload:    events,
		Sources:    []string{sourceURL},
	}, nil
}

func getCurrentAuctions(c *gin.Context) (endpointResult, error) {
	pageInput := strings.TrimSpace(c.Param("page"))
	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}

	baseURL := resolvedBaseURL
	auctions, sourceURL, err := scraper.FetchCurrentAuctions(c.Request.Context(), baseURL, page, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    []string{sourceURL},
	}, nil
}

func getAllCurrentAuctions(c *gin.Context) (endpointResult, error) {
	auctions, sources, err := scraper.FetchAllCurrentAuctions(c.Request.Context(), resolvedBaseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func getAuctionHistory(c *gin.Context) (endpointResult, error) {
	pageInput := strings.TrimSpace(c.Param("page"))
	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}

	baseURL := resolvedBaseURL
	auctions, sourceURL, err := scraper.FetchAuctionHistory(c.Request.Context(), baseURL, page, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    []string{sourceURL},
	}, nil
}

func getAllAuctionHistory(c *gin.Context) (endpointResult, error) {
	auctions, sources, err := scraper.FetchAllAuctionHistory(c.Request.Context(), resolvedBaseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func getCurrentAuctionsDetails(c *gin.Context) (endpointResult, error) {
	pageInput := strings.TrimSpace(c.Param("page"))
	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}

	auctions, sources, err := scraper.FetchCurrentAuctionsDetails(c.Request.Context(), resolvedBaseURL, page, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func getAllCurrentAuctionsDetails(c *gin.Context) (endpointResult, error) {
	auctions, sources, err := scraper.FetchAllCurrentAuctionsDetails(c.Request.Context(), resolvedBaseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func getAuctionHistoryDetails(c *gin.Context) (endpointResult, error) {
	pageInput := strings.TrimSpace(c.Param("page"))
	page, pageErr := validation.ParsePage(pageInput)
	if pageErr != nil {
		return endpointResult{}, pageErr
	}

	auctions, sources, err := scraper.FetchAuctionHistoryDetails(c.Request.Context(), resolvedBaseURL, page, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func getAllAuctionHistoryDetails(c *gin.Context) (endpointResult, error) {
	auctions, sources, err := scraper.FetchAllAuctionHistoryDetails(c.Request.Context(), resolvedBaseURL, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "auctions",
		Payload:    auctions,
		Sources:    sources,
	}, nil
}

func getAuctionDetail(c *gin.Context) (endpointResult, error) {
	auctionIDInput := strings.TrimSpace(c.Param("id"))
	auctionID, idErr := validation.ParseAuctionID(auctionIDInput)
	if idErr != nil {
		return endpointResult{}, idErr
	}

	baseURL := resolvedBaseURL
	auction, sources, err := scraper.FetchAuctionDetail(c.Request.Context(), baseURL, auctionID, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "auction",
		Payload:    auction,
		Sources:    sources,
	}, nil
}

func getTransfers(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
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

	baseURL := resolvedBaseURL
	transfers, sourceURL, err := scraper.FetchTransfers(
		c.Request.Context(),
		baseURL,
		scraper.TransfersFilters{
			WorldID:   worldID,
			WorldName: canonicalWorld,
			MinLevel:  minLevel,
			Page:      page,
		},
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "transfers",
		Payload:    transfers,
		Sources:    []string{sourceURL},
	}, nil
}

func getAllTransfers(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
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

	transfers, sources, err := scraper.FetchAllTransfers(
		c.Request.Context(),
		resolvedBaseURL,
		scraper.TransfersFilters{
			WorldID:   worldID,
			WorldName: canonicalWorld,
			MinLevel:  minLevel,
		},
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "transfers",
		Payload:    transfers,
		Sources:    sources,
	}, nil
}

func getBanishments(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
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

	baseURL := resolvedBaseURL
	banishments, sourceURL, err := scraper.FetchBanishments(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		page,
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "banishments",
		Payload:    banishments,
		Sources:    []string{sourceURL},
	}, nil
}

func getAllBanishments(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.BanishmentsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			banishments, sources, err := scraper.FetchAllBanishments(c.Request.Context(), resolvedBaseURL, world.Name, world.ID, resolvedOpts)
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

	banishments, sources, err := scraper.FetchAllBanishments(
		c.Request.Context(),
		resolvedBaseURL,
		canonicalWorld,
		worldID,
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "banishments",
		Payload:    banishments,
		Sources:    sources,
	}, nil
}

func getDeaths(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
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

	baseURL := resolvedBaseURL
	deaths, sourceURL, err := scraper.FetchDeaths(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		scraper.DeathsFilters{
			MinLevel: levelFilter,
			PvPOnly:  pvpOnly,
			Page:     page,
		},
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "deaths",
		Payload:    deaths,
		Sources:    []string{sourceURL},
	}, nil
}

func getAllDeaths(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
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
	}

	if isAllWorldsToken(worldInput) {
		worlds := validator.AllWorlds()
		results := make([]domain.DeathsResult, 0, len(worlds))
		allSources := make([]string, 0)
		for _, world := range worlds {
			deaths, sources, err := scraper.FetchAllDeaths(c.Request.Context(), resolvedBaseURL, world.Name, world.ID, filters, resolvedOpts)
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

	deaths, sources, err := scraper.FetchAllDeaths(
		c.Request.Context(),
		resolvedBaseURL,
		canonicalWorld,
		worldID,
		filters,
		resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}

	return endpointResult{
		PayloadKey: "deaths",
		Payload:    deaths,
		Sources:    sources,
	}, nil
}

func bootstrapValidator(ctx context.Context) (*validation.Validator, error) {
	baseURL := strings.TrimRight(resolvedBaseURL, "/")
	client := scraper.NewClient(resolvedOpts)
	refreshStarted := time.Now()

	worlds, err := discoverWorlds(ctx, client, baseURL)
	if err != nil {
		scraper.ValidatorRefresh.WithLabelValues("error").Inc()
		scraper.ValidatorRefreshDuration.Observe(time.Since(refreshStarted).Seconds())
		return nil, err
	}

	mappings := make([]scraper.WorldMapping, 0, len(worlds))
	for _, w := range worlds {
		mappings = append(mappings, scraper.WorldMapping{ID: w.ID, Name: w.Name})
	}
	scraper.UpdateWorldMappings(mappings)

	validator := validation.NewValidator(worlds)
	categories := validator.AllCategories()

	scraper.ValidatorRefresh.WithLabelValues("ok").Inc()
	scraper.ValidatorRefreshDuration.Observe(time.Since(refreshStarted).Seconds())
	scraper.WorldsDiscovered.Set(float64(len(worlds)))
	scraper.DiscoveredCount.WithLabelValues("worlds").Set(float64(len(worlds)))
	scraper.DiscoveredCount.WithLabelValues("towns").Set(0)
	scraper.DiscoveredCount.WithLabelValues("categories").Set(float64(len(categories)))
	return validator, nil
}

func discoverWorlds(ctx context.Context, client *scraper.Client, baseURL string) ([]validation.World, error) {
	sourceURL := fmt.Sprintf("%s/api/worlds", baseURL)
	started := time.Now()
	var payload struct {
		Worlds []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"worlds"`
	}
	err := client.FetchJSON(ctx, sourceURL, &payload)
	if err != nil {
		scraper.DiscoveryTotal.WithLabelValues("worlds", "error").Inc()
		scraper.DiscoveryDuration.WithLabelValues("worlds").Observe(time.Since(started).Seconds())
		return nil, err
	}

	worlds := make([]validation.World, 0, len(payload.Worlds))
	for _, row := range payload.Worlds {
		if row.ID <= 0 || strings.TrimSpace(row.Name) == "" {
			continue
		}
		worlds = append(worlds, validation.World{ID: row.ID, Name: strings.TrimSpace(row.Name)})
	}
	if len(worlds) == 0 {
		scraper.DiscoveryTotal.WithLabelValues("worlds", "error").Inc()
		scraper.DiscoveryDuration.WithLabelValues("worlds").Observe(time.Since(started).Seconds())
		return nil, validation.NewError(validation.ErrorUpstreamUnknown, "validator world bootstrap failed: no worlds discovered", nil)
	}

	scraper.DiscoveryTotal.WithLabelValues("worlds", "ok").Inc()
	scraper.DiscoveryDuration.WithLabelValues("worlds").Observe(time.Since(started).Seconds())
	return worlds, nil
}

func startValidatorRefresh() {
	interval := time.Duration(getEnvInt("VALIDATOR_REFRESH_INTERVAL_SECONDS", 3600)) * time.Second
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			validator, err := bootstrapValidator(context.Background())
			if err != nil {
				log.Printf("validator refresh failed: %v", err)
				continue
			}
			currentValidator.Store(validator)
		}
	}()
}

func getValidator() *validation.Validator {
	validator := currentValidator.Load()
	if validator == nil {
		panic("validator is not initialized")
	}
	return validator
}

func scrapeFetchOptions() scraper.FetchOptions {
	return scraper.FetchOptions{
		FlareSolverrURL: getEnv("FLARESOLVERR_URL", ""),
		MaxTimeoutMs:    getEnvInt("SCRAPE_MAX_TIMEOUT_MS", defaultScrapeTimeoutMS),
		CDPURL:          getEnv("CDP_URL", ""),
	}
}

func isAllWorldsToken(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "all")
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(getEnv(key, ""))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func postUpstreamRaw(c *gin.Context) (endpointResult, error) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "path is required", err)
	}
	if !strings.HasPrefix(req.Path, "/api/") {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "path must start with /api/", nil)
	}

	client := scraper.NewClient(resolvedOpts)
	sourceURL := fmt.Sprintf("%s%s", strings.TrimRight(resolvedBaseURL, "/"), req.Path)
	var raw any
	if err := client.FetchJSON(c.Request.Context(), sourceURL, &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "raw",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}

func postCharactersBatch(c *gin.Context) (endpointResult, error) {
	var req struct {
		Names []string `json:"names" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "names is required", err)
	}
	if len(req.Names) > 50 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "max 50 names per batch", nil)
	}
	if len(req.Names) == 0 {
		return endpointResult{
			PayloadKey: "characters",
			Payload:    []domain.CharacterResult{},
			Sources:    []string{},
		}, nil
	}

	results, sources, err := scraper.FetchCharactersBatch(c.Request.Context(), resolvedBaseURL, req.Names, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "characters",
		Payload:    results,
		Sources:    sources,
	}, nil
}

func postCharactersCompare(c *gin.Context) (endpointResult, error) {
	var req struct {
		Names []string `json:"names" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "names is required", err)
	}
	if len(req.Names) != 2 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "exactly 2 names required", nil)
	}

	results, sources, err := scraper.FetchCharactersBatch(c.Request.Context(), resolvedBaseURL, req.Names, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	if len(results) < 2 {
		return endpointResult{Sources: sources}, validation.NewError(validation.ErrorEntityNotFound, "one or both characters not found", nil)
	}

	signals := compareCharacters(results[0], results[1])
	return endpointResult{
		PayloadKey: "comparison",
		Payload: domain.ComparisonResult{
			Characters: results,
			Signals:    signals,
		},
		Sources: sources,
	}, nil
}

func compareCharacters(a, b domain.CharacterResult) domain.ComparisonSignals {
	sameAccount := false
	if len(a.OtherCharacters) > 0 && len(b.OtherCharacters) > 0 {
		aNames := make(map[string]bool, len(a.OtherCharacters))
		for _, oc := range a.OtherCharacters {
			aNames[strings.ToLower(strings.TrimSpace(oc.Name))] = true
		}
		bNames := make(map[string]bool, len(b.OtherCharacters))
		for _, oc := range b.OtherCharacters {
			bNames[strings.ToLower(strings.TrimSpace(oc.Name))] = true
		}
		if len(aNames) == len(bNames) && len(aNames) > 0 {
			match := true
			for k := range aNames {
				if !bNames[k] {
					match = false
					break
				}
			}
			sameAccount = match
		}
	}

	sameVipTime := a.CharacterInfo.VIPTime > 0 && a.CharacterInfo.VIPTime == b.CharacterInfo.VIPTime
	sameAccountCreated := a.CharacterInfo.AccountCreated != "" && a.CharacterInfo.AccountCreated == b.CharacterInfo.AccountCreated
	sameCreated := a.CharacterInfo.Created != "" && a.CharacterInfo.Created == b.CharacterInfo.Created
	sameLoyaltyPoints := a.CharacterInfo.LoyaltyPoints > 0 && a.CharacterInfo.LoyaltyPoints == b.CharacterInfo.LoyaltyPoints

	sameHouse := false
	if a.CharacterInfo.House != nil && b.CharacterInfo.House != nil {
		sameHouse = a.CharacterInfo.House.HouseID == b.CharacterInfo.House.HouseID && a.CharacterInfo.House.HouseID > 0
	}

	sameGuild := false
	if a.CharacterInfo.Guild != nil && b.CharacterInfo.Guild != nil {
		sameGuild = a.CharacterInfo.Guild.ID == b.CharacterInfo.Guild.ID && a.CharacterInfo.Guild.ID > 0
	}

	sameOutfit := false
	if a.CharacterInfo.Outfit != nil && b.CharacterInfo.Outfit != nil {
		oa, ob := a.CharacterInfo.Outfit, b.CharacterInfo.Outfit
		sameOutfit = oa.LookType == ob.LookType && oa.LookHead == ob.LookHead &&
			oa.LookBody == ob.LookBody && oa.LookLegs == ob.LookLegs &&
			oa.LookFeet == ob.LookFeet && oa.LookAddons == ob.LookAddons
	}

	return domain.ComparisonSignals{
		SameAccount:        sameAccount,
		SameVipTime:        sameVipTime,
		SameAccountCreated: sameAccountCreated,
		SameHouse:          sameHouse,
		SameGuild:          sameGuild,
		SameOutfit:         sameOutfit,
		SameCreated:        sameCreated,
		SameLoyaltyPoints:  sameLoyaltyPoints,
	}
}

func postHighscoresCrossWorld(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	categoryInput := strings.TrimSpace(c.Param("category"))
	category, categoryOK := validator.ResolveHighscoreCategory(categoryInput)
	if !categoryOK {
		return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, "highscore category does not exist", nil)
	}

	var req struct {
		Vocations []int `json:"vocations" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "vocations is required", err)
	}

	worlds := validator.AllWorlds()
	results, sources, err := scraper.FetchHighscoresCrossWorldAllVocations(
		c.Request.Context(), resolvedBaseURL, category, req.Vocations, worlds, resolvedOpts,
	)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "highscores",
		Payload:    results,
		Sources:    sources,
	}, nil
}

func postHighscoresMultiCategory(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	var req struct {
		Categories []string `json:"categories" binding:"required"`
		Vocations  []int    `json:"vocations" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "categories and vocations are required", err)
	}
	if len(req.Categories) > 10 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "max 10 categories", nil)
	}

	worlds := validator.AllWorlds()
	allSources := make([]string, 0)
	resultMap := make(map[string]map[string][]domain.HighscoresResult)

	for _, catInput := range req.Categories {
		category, categoryOK := validator.ResolveHighscoreCategory(strings.TrimSpace(catInput))
		if !categoryOK {
			return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, fmt.Sprintf("category not found: %s", catInput), nil)
		}
		results, sources, err := scraper.FetchHighscoresCrossWorldAllVocations(
			c.Request.Context(), resolvedBaseURL, category, req.Vocations, worlds, resolvedOpts,
		)
		if err != nil {
			return endpointResult{Sources: allSources}, err
		}
		allSources = append(allSources, sources...)
		resultMap[category.Slug] = results
	}

	return endpointResult{
		PayloadKey: "highscores",
		Payload:    resultMap,
		Sources:    allSources,
	}, nil
}

func postGuildsBatch(c *gin.Context) (endpointResult, error) {
	var req struct {
		Names []string `json:"names" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "names is required", err)
	}
	if len(req.Names) > 20 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "max 20 guild names per batch", nil)
	}

	results, sources, err := scraper.FetchGuildsBatch(c.Request.Context(), resolvedBaseURL, req.Names, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "guilds",
		Payload:    results,
		Sources:    sources,
	}, nil
}

func postAuctionsFilter(c *gin.Context) (endpointResult, error) {
	var req struct {
		Vocation int `json:"vocation"`
		MinLevel int `json:"minLevel"`
		MaxLevel int `json:"maxLevel"`
		World    int `json:"world"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "invalid filter parameters", err)
	}

	query := url.Values{}
	if req.Vocation > 0 {
		query.Set("vocation", strconv.Itoa(req.Vocation))
	}
	if req.MinLevel > 0 {
		query.Set("minLevel", strconv.Itoa(req.MinLevel))
	}
	if req.MaxLevel > 0 {
		query.Set("maxLevel", strconv.Itoa(req.MaxLevel))
	}
	if req.World > 0 {
		query.Set("world", strconv.Itoa(req.World))
	}

	sourceURL := fmt.Sprintf("%s/api/bazaar?%s", strings.TrimRight(resolvedBaseURL, "/"), query.Encode())
	client := scraper.NewClient(resolvedOpts)
	var raw any
	if err := client.FetchJSON(c.Request.Context(), sourceURL, &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "auctions",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}

func postKillstatisticsBatch(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	var req struct {
		Worlds []string `json:"worlds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "worlds is required", err)
	}
	if len(req.Worlds) > 20 {
		return endpointResult{}, validation.NewError(validation.ErrorInvalidParameter, "max 20 worlds per batch", nil)
	}

	validWorlds := make([]validation.World, 0, len(req.Worlds))
	for _, w := range req.Worlds {
		name, id, ok := validator.WorldExists(w)
		if !ok {
			return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, fmt.Sprintf("world not found: %s", w), nil)
		}
		validWorlds = append(validWorlds, validation.World{Name: name, ID: id})
	}

	results, sources, err := scraper.FetchAllWorldsKillstatistics(c.Request.Context(), resolvedBaseURL, validWorlds, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: sources}, err
	}
	return endpointResult{
		PayloadKey: "killstatistics",
		Payload:    results,
		Sources:    sources,
	}, nil
}

func getBans(c *gin.Context) (endpointResult, error) {
	sourceURL := fmt.Sprintf("%s/api/bans", strings.TrimRight(resolvedBaseURL, "/"))
	client := scraper.NewClient(resolvedOpts)
	var raw any
	if err := client.FetchJSON(c.Request.Context(), sourceURL, &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "bans",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}

func getNewsAll(c *gin.Context) (endpointResult, error) {
	sourceURL := fmt.Sprintf("%s/api/news", strings.TrimRight(resolvedBaseURL, "/"))
	client := scraper.NewClient(resolvedOpts)
	var raw any
	if err := client.FetchJSON(c.Request.Context(), sourceURL, &raw); err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}
	return endpointResult{
		PayloadKey: "news",
		Payload:    raw,
		Sources:    []string{sourceURL},
	}, nil
}

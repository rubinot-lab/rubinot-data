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
		v1.GET("/events/schedule", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getEventsSchedule(c)
		}))
		v1.GET("/auctions/current/all", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAllCurrentAuctions(c)
		}))
		v1.GET("/auctions/current/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getCurrentAuctions(c)
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

func getHighscores(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	categoryInput := strings.TrimSpace(c.Param("category"))
	vocationInput := strings.TrimSpace(c.Param("vocation"))
	pageInput := strings.TrimSpace(c.Param("page"))

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

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

	baseURL := resolvedBaseURL
	highscores, sourceURL, err := scraper.FetchHighscores(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		category,
		vocation,
		page,
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

func getAllHighscores(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	categoryInput := strings.TrimSpace(c.Param("category"))
	vocationInput := strings.TrimSpace(c.Param("vocation"))

	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	category, categoryOK := validator.ResolveHighscoreCategory(categoryInput)
	if !categoryOK {
		return endpointResult{}, validation.NewError(validation.ErrorHighscoreCategoryDoesNotExist, "highscore category does not exist", nil)
	}

	vocation, vocationOK := validator.ResolveHighscoreVocation(vocationInput)
	if !vocationOK {
		return endpointResult{}, validation.NewError(validation.ErrorVocationDoesNotExist, "vocation does not exist", nil)
	}

	highscores, sourceURL, err := scraper.FetchAllHighscores(
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
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
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

	deaths, sources, err := scraper.FetchAllDeaths(
		c.Request.Context(),
		resolvedBaseURL,
		canonicalWorld,
		worldID,
		scraper.DeathsFilters{
			MinLevel: levelFilter,
			PvPOnly:  pvpOnly,
		},
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

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/scraper"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultRubinotBaseURL  = "https://www.rubinot.com.br"
	defaultScrapeTimeoutMS = 120000
	defaultServiceVersion  = "dev"
)

var (
	resolvedBaseURL string
	resolvedOpts    scraper.FetchOptions
)

func NewRouter() (*gin.Engine, error) {
	resolvedBaseURL = getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	resolvedOpts = scrapeFetchOptions()

	validator, err := bootstrapValidator(context.Background())
	if err != nil {
		return nil, err
	}

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
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/v1")
	{
		v1.GET("/worlds", handleEndpoint(getWorlds))
		v1.GET("/world/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getWorld(c, validator)
		}))
		v1.GET("/highscores/:world", redirectHighscoresWorld)
		v1.GET("/highscores/:world/:category", redirectHighscoresCategory)
		v1.GET("/highscores/:world/:category/:vocation/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getHighscores(c, validator)
		}))
		v1.GET("/killstatistics/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getKillstatistics(c, validator)
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
		v1.GET("/auctions/current/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getCurrentAuctions(c)
		}))
		v1.GET("/auctions/history/:page", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAuctionHistory(c)
		}))
		v1.GET("/auctions/:id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getAuctionDetail(c)
		}))
		v1.GET("/deaths/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getDeaths(c, validator)
		}))
		v1.GET("/banishments/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getBanishments(c, validator)
		}))
		v1.GET("/transfers", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getTransfers(c, validator)
		}))
		v1.GET("/character/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getCharacter(c)
		}))
		v1.GET("/guild/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getGuild(c)
		}))
		v1.GET("/guilds/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getGuilds(c, validator)
		}))
		v1.GET("/house/:world/:house_id", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getHouse(c, validator)
		}))
		v1.GET("/houses/:world/:town", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getHouses(c, validator)
		}))
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
	worldInput := strings.TrimSpace(c.Param("world"))
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	baseURL := resolvedBaseURL
	guilds, sourceURL, err := scraper.FetchGuilds(c.Request.Context(), baseURL, canonicalWorld, worldID, resolvedOpts)
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "guilds",
		Payload:    guilds,
		Sources:    []string{sourceURL},
	}, nil
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

	canonicalWorld, _, worldOK := validator.WorldExists(worldInput)
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

func getDeaths(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	canonicalWorld, worldID, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}

	guildFilter := strings.TrimSpace(c.Query("guild"))
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
			Guild:    guildFilter,
			MinLevel: levelFilter,
			PvPOnly:  pvpOnly,
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

func bootstrapValidator(ctx context.Context) (*validation.Validator, error) {
	baseURL := strings.TrimRight(resolvedBaseURL, "/")
	sourceURL := fmt.Sprintf("%s/?subtopic=latestdeaths", baseURL)

	started := time.Now()
	html, err := scraper.NewClient(resolvedOpts).Fetch(ctx, sourceURL)
	if err != nil {
		scraper.ValidatorRefresh.WithLabelValues("error").Inc()
		scraper.ValidatorRefreshDuration.Observe(time.Since(started).Seconds())
		return nil, err
	}

	worlds, err := validation.ParseLatestDeathsWorldOptions(html)
	if err != nil {
		scraper.ValidatorRefresh.WithLabelValues("error").Inc()
		scraper.ValidatorRefreshDuration.Observe(time.Since(started).Seconds())
		return nil, validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("validator world bootstrap failed: %v", err), err)
	}

	scraper.ValidatorRefresh.WithLabelValues("ok").Inc()
	scraper.ValidatorRefreshDuration.Observe(time.Since(started).Seconds())
	scraper.WorldsDiscovered.Set(float64(len(worlds)))

	return validation.NewValidator(worlds), nil
}

func scrapeFetchOptions() scraper.FetchOptions {
	return scraper.FetchOptions{
		FlareSolverrURL: getEnv("FLARESOLVERR_URL", ""),
		MaxTimeoutMs:    getEnvInt("SCRAPE_MAX_TIMEOUT_MS", defaultScrapeTimeoutMS),
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

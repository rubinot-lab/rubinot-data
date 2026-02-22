package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/scraper"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultRubinotBaseURL  = "https://www.rubinot.com.br"
	defaultFlareSolverrURL = "http://flaresolverr.network.svc.cluster.local:8191/v1"
	defaultScrapeTimeoutMS = 120000
	defaultServiceVersion  = "dev"
)

func NewRouter() (*gin.Engine, error) {
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
		v1.GET("/deaths/:world", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getDeaths(c, validator)
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
	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)

	worlds, sourceURL, err := scraper.FetchWorlds(c.Request.Context(), baseURL, scrapeFetchOptions())
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	world, sourceURL, err := scraper.FetchWorld(c.Request.Context(), baseURL, canonicalWorld, scrapeFetchOptions())
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	character, sourceURL, err := scraper.FetchCharacter(c.Request.Context(), baseURL, canonicalName, scrapeFetchOptions())
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	guild, sourceURL, err := scraper.FetchGuild(c.Request.Context(), baseURL, canonicalName, scrapeFetchOptions())
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	guilds, sourceURL, err := scraper.FetchGuilds(c.Request.Context(), baseURL, canonicalWorld, worldID, scrapeFetchOptions())
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	houses, sourceURL, err := scraper.FetchHouses(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		canonicalTown,
		townID,
		scrapeFetchOptions(),
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	house, sourceURL, err := scraper.FetchHouse(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		houseID,
		validator.AllTowns(),
		scrapeFetchOptions(),
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	highscores, sourceURL, err := scraper.FetchHighscores(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		category,
		vocation,
		page,
		scrapeFetchOptions(),
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	killstatistics, sourceURL, err := scraper.FetchKillstatistics(
		c.Request.Context(),
		baseURL,
		canonicalWorld,
		worldID,
		scrapeFetchOptions(),
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	news, sources, err := scraper.FetchNewsByID(c.Request.Context(), baseURL, newsID, scrapeFetchOptions())
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	newsList, sourceURL, err := scraper.FetchNewsArchive(c.Request.Context(), baseURL, archiveDays, scrapeFetchOptions())
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
	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	newsList, sourceURL, err := scraper.FetchNewsLatest(c.Request.Context(), baseURL, scrapeFetchOptions())
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
	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	newsList, sourceURL, err := scraper.FetchNewsTicker(c.Request.Context(), baseURL, scrapeFetchOptions())
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "newslist",
		Payload:    newsList,
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

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
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
		scrapeFetchOptions(),
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
	baseURL := strings.TrimRight(getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL), "/")
	sourceURL := fmt.Sprintf("%s/?subtopic=latestdeaths", baseURL)

	html, err := scraper.NewClient(scrapeFetchOptions()).Fetch(ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	worlds, err := validation.ParseLatestDeathsWorldOptions(html)
	if err != nil {
		return nil, validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("validator world bootstrap failed: %v", err), err)
	}

	return validation.NewValidator(worlds), nil
}

func scrapeFetchOptions() scraper.FetchOptions {
	return scraper.FetchOptions{
		FlareSolverrURL: getEnv("FLARESOLVERR_URL", defaultFlareSolverrURL),
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

package api

import (
	"context"
	"fmt"
	"net/http"
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
		v1.GET("/world/:name", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getWorld(c, validator)
		}))
		v1.GET("/houses/:world/:town", handleEndpoint(func(c *gin.Context) (endpointResult, error) {
			return getHouses(c, validator)
		}))
	}

	return router, nil
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

func getHouses(c *gin.Context, validator *validation.Validator) (endpointResult, error) {
	worldInput := strings.TrimSpace(c.Param("world"))
	townInput := strings.TrimSpace(c.Param("town"))

	canonicalWorld, _, worldOK := validator.WorldExists(worldInput)
	if !worldOK {
		return endpointResult{}, validation.NewError(validation.ErrorWorldDoesNotExist, "world does not exist", nil)
	}
	canonicalTown, _, townOK := validator.TownExists(townInput)
	if !townOK {
		return endpointResult{}, validation.NewError(validation.ErrorTownDoesNotExist, "town does not exist", nil)
	}

	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)
	houses, sourceURL, err := scraper.FetchHouses(c.Request.Context(), baseURL, canonicalWorld, canonicalTown, scrapeFetchOptions())
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "houses",
		Payload:    houses,
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

package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/scraper"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultRubinotBaseURL  = "https://www.rubinot.com.br"
	defaultFlareSolverrURL = "http://flaresolverr.network.svc.cluster.local:8191/v1"
	defaultScrapeTimeoutMS = 120000
	defaultServiceVersion  = "dev"
)

func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(metricsMiddleware())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "rubinot-data api up"})
	})
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/versions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "rubinot-data",
			"version": getEnv("APP_VERSION", defaultServiceVersion),
			"commit":  getEnv("APP_COMMIT", defaultAPICommit),
		})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/v1")
	{
		v1.GET("/world/:name", handleEndpoint(getWorld))
		v1.GET("/houses/:world/:town", handleEndpoint(getHouses))
	}

	return r
}

func getWorld(c *gin.Context) (endpointResult, error) {
	name := strings.TrimSpace(c.Param("name"))
	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)

	world, sourceURL, err := scraper.FetchWorld(c.Request.Context(), baseURL, name, scrapeFetchOptions())
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "world",
		Payload:    world,
		Sources:    []string{sourceURL},
	}, nil
}

func getHouses(c *gin.Context) (endpointResult, error) {
	world := strings.TrimSpace(c.Param("world"))
	town := strings.TrimSpace(c.Param("town"))
	baseURL := getEnv("RUBINOT_BASE_URL", defaultRubinotBaseURL)

	houses, sourceURL, err := scraper.FetchHouses(c.Request.Context(), baseURL, world, town, scrapeFetchOptions())
	if err != nil {
		return endpointResult{Sources: []string{sourceURL}}, err
	}

	return endpointResult{
		PayloadKey: "houses",
		Payload:    houses,
		Sources:    []string{sourceURL},
	}, nil
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
